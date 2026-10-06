"""Compile synthetic CSV solver scenarios into the v1 sidecar contract.

The harness deliberately has no database dependency.  Scenario CSVs use the
opaque IDs the contract receives; presentation names do not belong in a solver
snapshot (SPEC §§17.3, 17.8; §8.7).
"""

from __future__ import annotations

import csv
from collections import defaultdict
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Iterable

from app.contract import CONTRACT_VERSION

SCENARIOS = Path(__file__).parent / "scenarios"


@dataclass(frozen=True)
class ExpectedScenario:
    status: str
    assignments: tuple[dict[str, str], ...]


def compile_request(
    directories: Iterable[Path], *, seed: int, max_deterministic_time: float = 1, high_rank_max: int = 3
) -> dict[str, object]:
    """Merge normalized CSV inputs into a self-contained v1 solve request.

    More than one directory may be supplied, so focused input layers can be
    composed without an application database or a production roster export.
    Each input directory supplies students, offerings, interest ratings, and
    ranked choices; pins are optional.
    """
    directories = tuple(directories)
    if not directories:
        raise ValueError("at least one scenario directory is required")

    students = _rows(directories, "students.csv", ("id", "grade_ordinal"))
    offerings = _rows(
        directories,
        "offerings.csv",
        ("id", "capacity", "min_grade_ordinal", "max_grade_ordinal", "interest_area_id"),
    )
    interest_ratings = _rows(directories, "interest-ratings.csv", ("participant_id", "interest_area_id", "rating"))
    ranked_choices = _rows(directories, "ranked-choices.csv", ("participant_id", "offering_id", "response", "rank"))
    pins = _rows(directories, "pins.csv", ("participant_id", "offering_id"), required=False)
    exclusions = _rows(directories, "exclusions.csv", ("participant_id", "offering_id"), required=False)
    exceptions = _rows(directories, "authorized-pinned-exceptions.csv", ("participant_id", "offering_id", "rule"), required=False)
    prior_placements = _rows(directories, "prior-placements.csv", ("participant_id", "offering_id"), required=False)

    ratings_by_participant: dict[str, list[dict[str, str]]] = defaultdict(list)
    for rating in interest_ratings:
        ratings_by_participant[rating["participant_id"]].append(
            {"interest_area_id": rating["interest_area_id"], "rating": rating["rating"]}
        )
    choices_by_participant: dict[str, list[dict[str, object]]] = defaultdict(list)
    for choice in ranked_choices:
        compiled_choice: dict[str, object] = {"offering_id": choice["offering_id"], "response": choice["response"]}
        if choice["rank"]:
            compiled_choice["rank"] = _integer(choice["rank"], "ranked-choices.csv rank")
        choices_by_participant[choice["participant_id"]].append(compiled_choice)

    participants = []
    for student in students:
        participant: dict[str, Any] = {
            "id": student["id"],
            "grade_ordinal": _integer(student["grade_ordinal"], "students.csv grade_ordinal"),
            "interest_profile": ratings_by_participant[student["id"]],
        }
        if student["id"] in choices_by_participant:
            participant["ranked_choices"] = {"choices": choices_by_participant[student["id"]]}
        participants.append(participant)

    return {
        "version": CONTRACT_VERSION,
        "seed": seed,
        "max_deterministic_time": max_deterministic_time,
        "quality_config": {"high_rank_max": high_rank_max},
        "participants": participants,
        "offerings": [
            {
                "id": offering["id"],
                "capacity": _integer(offering["capacity"], "offerings.csv capacity"),
                "min_grade_ordinal": _integer(offering["min_grade_ordinal"], "offerings.csv min_grade_ordinal"),
                "max_grade_ordinal": _integer(offering["max_grade_ordinal"], "offerings.csv max_grade_ordinal"),
                **({"interest_area_id": offering["interest_area_id"]} if offering["interest_area_id"] else {}),
            }
            for offering in offerings
        ],
        "pins": [{"participant_id": pin["participant_id"], "offering_id": pin["offering_id"]} for pin in pins],
        "exclusions": [{"participant_id": exclusion["participant_id"], "offering_id": exclusion["offering_id"]} for exclusion in exclusions],
        "authorized_pinned_exceptions": [
            {"participant_id": exception["participant_id"], "offering_id": exception["offering_id"], "rule": exception["rule"]} for exception in exceptions
        ],
        "prior_placements": [{"participant_id": placement["participant_id"], "offering_id": placement["offering_id"]} for placement in prior_placements],
    }


def expected_result(directory: Path) -> ExpectedScenario:
    rows = _read_csv(directory / "expected.csv", ("status", "participant_id", "offering_id", "realized_quality"))
    if not rows:
        raise ValueError(f"{directory / 'expected.csv'} must contain a status row")
    statuses = {row["status"] for row in rows}
    if len(statuses) != 1 or not next(iter(statuses)):
        raise ValueError(f"{directory / 'expected.csv'} must declare one non-empty status")
    assignments = []
    for row in rows:
        values = (row["participant_id"], row["offering_id"], row["realized_quality"])
        if any(values) and not all(values):
            raise ValueError(f"{directory / 'expected.csv'} assignments must be complete")
        if all(values):
            assignments.append(
                {"participant_id": row["participant_id"], "offering_id": row["offering_id"], "realized_quality": row["realized_quality"]}
            )
    return ExpectedScenario(next(iter(statuses)), tuple(assignments))


def _rows(directories: Iterable[Path], filename: str, headers: tuple[str, ...], *, required: bool = True) -> list[dict[str, str]]:
    rows = []
    for directory in directories:
        path = directory / filename
        if not path.exists():
            if required:
                raise ValueError(f"scenario input is missing {path}")
            continue
        rows.extend(_read_csv(path, headers))
    return rows


def _read_csv(path: Path, headers: tuple[str, ...]) -> list[dict[str, str]]:
    with path.open(newline="", encoding="utf-8") as source:
        reader = csv.DictReader(source)
        if reader.fieldnames != list(headers):
            raise ValueError(f"{path} headers must be {', '.join(headers)}")
        return [{header: (row[header] or "").strip() for header in headers} for row in reader]


def _integer(value: str, field: str) -> int:
    try:
        return int(value)
    except ValueError as error:
        raise ValueError(f"{field} must be an integer") from error
