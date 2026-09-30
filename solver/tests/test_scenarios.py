import json
from time import monotonic

import pytest

from app.server import solve
from scenario_harness import SCENARIOS, compile_request, expected_result


@pytest.mark.parametrize(
    ("name", "seed"),
    [
        ("capacity", 11),
        ("grade-eligibility", 12),
        ("ranked-preferences", 13),
        ("interest-preferences", 14),
        ("lexicographic", 15),
        ("pins", 16),
        ("infeasible", 17),
        ("determinism", 18),
        ("expected-scale", 19),
    ],
)
def test_synthetic_csv_scenarios(name: str, seed: int) -> None:
    directory = SCENARIOS / name
    request = compile_request([directory], seed=seed)
    expected = expected_result(directory)

    result = solve(request)

    assert result["status"] == expected.status
    if expected.assignments:
        assert result["assignments"] == list(expected.assignments)
    if expected.status == "infeasible":
        assert result["assignments"] == []
    if result["status"] == "optimal":
        _assert_complete_eligible_capacity_respecting_result(request, result)


def test_csv_inputs_compose_supported_rules_without_a_database() -> None:
    request = compile_request(
        [
            SCENARIOS / "capacity",
            SCENARIOS / "grade-eligibility",
            SCENARIOS / "ranked-preferences",
            SCENARIOS / "interest-preferences",
            SCENARIOS / "lexicographic",
            SCENARIOS / "pins",
        ],
        seed=21,
    )
    result = solve(request)

    assert result["status"] == "optimal"
    _assert_complete_eligible_capacity_respecting_result(request, result)
    assert {assignment["participant_id"] for assignment in result["assignments"]} == {
        "student-cap-001",
        "student-cap-002",
        "student-grade-001",
        "student-grade-002",
        "student-ranked-001",
        "student-ranked-002",
        "student-interest-001",
        "student-interest-002",
        "student-lex-001",
        "student-lex-002",
        "student-lex-003",
        "student-pin-001",
        "student-pin-002",
    }


def test_same_csv_scenario_and_seed_are_byte_identical() -> None:
    request = compile_request([SCENARIOS / "determinism"], seed=18)
    first = json.dumps(solve(request), separators=(",", ":"), sort_keys=True)

    assert first == json.dumps(solve(request), separators=(",", ":"), sort_keys=True)


def test_expected_scale_csv_scenario_meets_full_and_incremental_budgets() -> None:
    request = compile_request([SCENARIOS / "expected-scale"], seed=19)

    started = monotonic()
    full_result = solve(request)
    full_elapsed = monotonic() - started
    started = monotonic()
    incremental_result = solve({**request, "seed": 20})
    incremental_elapsed = monotonic() - started

    assert full_result["status"] == "optimal"
    assert incremental_result["status"] == "optimal"
    assert full_elapsed < 10
    assert incremental_elapsed < 2


def _assert_complete_eligible_capacity_respecting_result(request: dict[str, object], result: dict[str, object]) -> None:
    assignments = result["assignments"]
    participants = {participant["id"]: participant for participant in request["participants"]}  # type: ignore[index]
    offerings = {offering["id"]: offering for offering in request["offerings"]}  # type: ignore[index]

    assert {assignment["participant_id"] for assignment in assignments} == set(participants)  # type: ignore[index]
    used_capacity: dict[str, int] = {}
    for assignment in assignments:  # type: ignore[union-attr]
        participant = participants[assignment["participant_id"]]
        offering = offerings[assignment["offering_id"]]
        assert offering["min_grade_ordinal"] <= participant["grade_ordinal"] <= offering["max_grade_ordinal"]
        used_capacity[assignment["offering_id"]] = used_capacity.get(assignment["offering_id"], 0) + 1
    assert all(used_capacity[offering_id] <= offerings[offering_id]["capacity"] for offering_id in used_capacity)
