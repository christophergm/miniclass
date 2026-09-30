import json
from pathlib import Path

import pytest

from app.contract import CONTRACT_VERSION
from app.server import solve


def test_repeated_fixture_solve_is_byte_identical() -> None:
    request_document = json.loads((Path(__file__).parent / "fixtures" / "solve-request-v1.json").read_text())
    assert request_document["version"] == CONTRACT_VERSION
    first = solve(request_document)
    assert first == solve(request_document)
    assert first == {
        "version": CONTRACT_VERSION,
        "seed": 41,
        "status": "optimal",
        "assignments": [
            {"participant_id": "student-a", "offering_id": "offering-a", "realized_quality": "high"},
            {"participant_id": "student-b", "offering_id": "offering-b", "realized_quality": "acceptable"},
        ],
        "conflict_diagnostics": [],
    }
    assert json.dumps(first, separators=(",", ":"), sort_keys=True) == json.dumps(solve(request_document), separators=(",", ":"), sort_keys=True)


@pytest.mark.parametrize(
    ("participant", "offering", "expected_quality"),
    [
        ({"ranked_choices": {"choices": [{"offering_id": "offering", "response": "ranked", "rank": 1}]}}, {}, "top"),
        ({"ranked_choices": {"choices": [{"offering_id": "offering", "response": "ranked", "rank": 2}]}}, {}, "high"),
        ({"ranked_choices": {"choices": [{"offering_id": "offering", "response": "ranked", "rank": 4}]}}, {}, "acceptable"),
        ({"ranked_choices": {"choices": [{"offering_id": "offering", "response": "interested"}]}}, {}, "acceptable"),
        ({"ranked_choices": {"choices": [{"offering_id": "offering", "response": "not_interested"}]}}, {}, "unwanted"),
        ({"ranked_choices": {"choices": []}}, {}, "neutral"),
        ({"interest_profile": [{"interest_area_id": "area", "rating": "very_interested"}]}, {"interest_area_id": "area"}, "high"),
        ({"interest_profile": [{"interest_area_id": "area", "rating": "interested"}]}, {"interest_area_id": "area"}, "acceptable"),
        ({"interest_profile": [{"interest_area_id": "area", "rating": "not_interested"}]}, {"interest_area_id": "area"}, "unwanted"),
        ({"interest_profile": []}, {"interest_area_id": "area"}, "neutral"),
        ({"interest_profile": [{"interest_area_id": "area", "rating": "very_interested"}]}, {}, "neutral"),
    ],
)
def test_preference_models_map_to_realized_quality(participant: dict[str, object], offering: dict[str, object], expected_quality: str) -> None:
    result = solve(
        request(
            offerings=[{"id": "offering", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1, **offering}],
            participants=[{"id": "student", "grade_ordinal": 1, **participant}],
        )
    )

    assert result["status"] == "optimal"
    assert result["assignments"] == [{"participant_id": "student", "offering_id": "offering", "realized_quality": expected_quality}]


def test_ranked_choices_precede_interest_profile() -> None:
    result = solve(
        request(
            offerings=[{"id": "offering", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1, "interest_area_id": "area"}],
            participants=[
                {
                    "id": "student",
                    "grade_ordinal": 1,
                    "ranked_choices": {"choices": []},
                    "interest_profile": [{"interest_area_id": "area", "rating": "not_interested"}],
                }
            ],
        )
    )

    assert result["assignments"][0]["realized_quality"] == "neutral"


def test_feasibility_enforces_capacity_and_exactly_one_assignment() -> None:
    result = solve(
        request(
            offerings=[
                {"id": "art", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1},
                {"id": "music", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1},
            ],
            participants=[{"id": "alex", "grade_ordinal": 1}, {"id": "blair", "grade_ordinal": 1}],
        )
    )

    assert result["status"] == "optimal"
    assert len(result["assignments"]) == 2
    assert {assignment["participant_id"] for assignment in result["assignments"]} == {"alex", "blair"}
    assert len({assignment["offering_id"] for assignment in result["assignments"]}) == 2


def test_feasibility_enforces_grade_windows() -> None:
    result = solve(
        request(
            offerings=[
                {"id": "junior-art", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1},
                {"id": "senior-art", "capacity": 1, "min_grade_ordinal": 2, "max_grade_ordinal": 2},
            ],
            participants=[{"id": "junior", "grade_ordinal": 1}, {"id": "senior", "grade_ordinal": 2}],
        )
    )

    assert result["status"] == "optimal"
    assert result["assignments"] == [
        {"participant_id": "junior", "offering_id": "junior-art", "realized_quality": "neutral"},
        {"participant_id": "senior", "offering_id": "senior-art", "realized_quality": "neutral"},
    ]


def test_infeasible_model_returns_no_partial_assignment() -> None:
    result = solve(
        request(
            offerings=[{"id": "art", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1}],
            participants=[{"id": "alex", "grade_ordinal": 1}, {"id": "blair", "grade_ordinal": 1}],
        )
    )

    assert result == {
        "version": CONTRACT_VERSION,
        "seed": 7,
        "status": "infeasible",
        "assignments": [],
        "conflict_diagnostics": [],
    }


def test_lexicographic_objective_protects_unwanted_before_aggregate_satisfaction() -> None:
    result = solve(
        request(
            offerings=[
                {"id": "popular", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1},
                {"id": "fallback", "capacity": 2, "min_grade_ordinal": 1, "max_grade_ordinal": 1},
            ],
            participants=[
                ranked_participant("a", [("popular", "ranked", 1), ("fallback", "not_interested", None)]),
                ranked_participant("b", [("popular", "ranked", 1), ("fallback", "interested", None)]),
                ranked_participant("c", [("popular", "ranked", 1), ("fallback", "interested", None)]),
            ],
        )
    )

    assert assignment_offerings(result) == {"a": "popular", "b": "fallback", "c": "fallback"}
    assert assignment_qualities(result) == {"a": "top", "b": "acceptable", "c": "acceptable"}


def test_lexicographic_objective_protects_neutral_before_acceptable() -> None:
    result = solve(
        request(
            offerings=[
                {"id": "contested", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1},
                {"id": "fallback", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1},
            ],
            participants=[
                ranked_participant("acceptable", [("contested", "ranked", 1), ("fallback", "interested", None)]),
                ranked_participant("neutral", [("contested", "ranked", 2)]),
            ],
        )
    )

    assert assignment_offerings(result) == {"acceptable": "fallback", "neutral": "contested"}
    assert assignment_qualities(result) == {"acceptable": "acceptable", "neutral": "high"}


def test_lexicographic_objective_protects_acceptable_before_high_without_maximizing_top() -> None:
    result = solve(
        request(
            offerings=[
                {"id": "contested", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1},
                {"id": "fallback", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1},
            ],
            participants=[
                ranked_participant("high", [("contested", "ranked", 2), ("fallback", "interested", None)]),
                ranked_participant("top", [("contested", "ranked", 1), ("fallback", "ranked", 2)]),
            ],
        )
    )

    assert assignment_offerings(result) == {"high": "contested", "top": "fallback"}
    assert assignment_qualities(result) == {"high": "high", "top": "high"}


def test_input_order_does_not_change_seeded_result() -> None:
    original = request(
        offerings=[
            {"id": "art", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1},
            {"id": "music", "capacity": 1, "min_grade_ordinal": 1, "max_grade_ordinal": 1},
        ],
        participants=[{"id": "alex", "grade_ordinal": 1}, {"id": "blair", "grade_ordinal": 1}],
        seed=0,
    )
    reordered = {**original, "offerings": list(reversed(original["offerings"])), "participants": list(reversed(original["participants"]))}

    first = solve(original)
    assert first == solve(reordered)
    assert first["assignments"] != solve({**original, "seed": 1})["assignments"]


def request(*, offerings: list[dict[str, object]], participants: list[dict[str, object]], seed: int = 7) -> dict[str, object]:
    return {
        "version": CONTRACT_VERSION,
        "seed": seed,
        "max_deterministic_time": 1,
        "quality_config": {"high_rank_max": 3},
        "offerings": offerings,
        "participants": participants,
    }


def ranked_participant(student_id: str, choices: list[tuple[str, str, int | None]]) -> dict[str, object]:
    return {
        "id": student_id,
        "grade_ordinal": 1,
        "ranked_choices": {
            "choices": [
                {"offering_id": offering_id, "response": response, **({"rank": rank} if rank is not None else {})}
                for offering_id, response, rank in choices
            ]
        },
    }


def assignment_offerings(result: dict[str, object]) -> dict[str, str]:
    return {assignment["participant_id"]: assignment["offering_id"] for assignment in result["assignments"]}  # type: ignore[index]


def assignment_qualities(result: dict[str, object]) -> dict[str, str]:
    return {assignment["participant_id"]: assignment["realized_quality"] for assignment in result["assignments"]}  # type: ignore[index]
