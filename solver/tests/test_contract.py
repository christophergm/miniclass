import json
from pathlib import Path

from app.contract import CONTRACT_VERSION
from app.server import solve


def test_repeated_fixture_solve_is_byte_identical() -> None:
    request = json.loads((Path(__file__).parent / "fixtures" / "solve-request-v1.json").read_text())
    assert request["version"] == CONTRACT_VERSION
    first = solve(request)
    assert first == solve(request)
    assert first == {
        "version": CONTRACT_VERSION,
        "seed": 41,
        "status": "optimal",
        "assignments": [
            {"participant_id": "student-a", "offering_id": "offering-a"},
            {"participant_id": "student-b", "offering_id": "offering-b"},
        ],
        "conflict_diagnostics": [],
    }
    assert json.dumps(first, separators=(",", ":"), sort_keys=True) == json.dumps(solve(request), separators=(",", ":"), sort_keys=True)


def test_feasibility_enforces_capacity_and_exactly_one_assignment() -> None:
    result = solve(
        {
            "version": CONTRACT_VERSION,
            "seed": 7,
            "max_deterministic_time": 1,
            "offerings": [
                {"id": "art", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1},
                {"id": "music", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1},
            ],
            "participants": [
                {"id": "alex", "grade_ordinal": 1},
                {"id": "blair", "grade_ordinal": 1},
            ],
        }
    )

    assert result["status"] == "optimal"
    assert len(result["assignments"]) == 2
    assert {assignment["participant_id"] for assignment in result["assignments"]} == {"alex", "blair"}
    assert len({assignment["offering_id"] for assignment in result["assignments"]}) == 2


def test_feasibility_enforces_grade_windows() -> None:
    result = solve(
        {
            "version": CONTRACT_VERSION,
            "seed": 7,
            "max_deterministic_time": 1,
            "offerings": [
                {"id": "junior-art", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1},
                {"id": "senior-art", "capacity": 1, "min_grade_ordinal": 2, "max_grade_ordinal": 2},
            ],
            "participants": [
                {"id": "junior", "grade_ordinal": 1},
                {"id": "senior", "grade_ordinal": 2},
            ],
        }
    )

    assert result["status"] == "optimal"
    assert result["assignments"] == [
        {"participant_id": "junior", "offering_id": "junior-art"},
        {"participant_id": "senior", "offering_id": "senior-art"},
    ]


def test_infeasible_model_returns_no_partial_assignment() -> None:
    result = solve(
        {
            "version": CONTRACT_VERSION,
            "seed": 7,
            "max_deterministic_time": 1,
            "offerings": [{"id": "art", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1}],
            "participants": [
                {"id": "alex", "grade_ordinal": 1},
                {"id": "blair", "grade_ordinal": 1},
            ],
        }
    )

    assert result == {
        "version": CONTRACT_VERSION,
        "seed": 7,
        "status": "infeasible",
        "assignments": [],
        "conflict_diagnostics": [],
    }


def test_input_order_does_not_change_seeded_result() -> None:
    request = {
        "version": CONTRACT_VERSION,
        "seed": 0,
        "max_deterministic_time": 1,
        "offerings": [
            {"id": "art", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1},
            {"id": "music", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1},
        ],
        "participants": [
            {"id": "alex", "grade_ordinal": 1},
            {"id": "blair", "grade_ordinal": 1},
        ],
    }
    reordered = {**request, "offerings": list(reversed(request["offerings"])), "participants": list(reversed(request["participants"]))}

    first = solve(request)
    assert first == solve(reordered)
    assert first["assignments"] != solve({**request, "seed": 1})["assignments"]
