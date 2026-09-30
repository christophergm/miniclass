"""Versioned, self-contained solver wire contract (SPEC §§17.3, 17.8)."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

CONTRACT_VERSION = "v1"


class ContractError(ValueError):
    pass


@dataclass(frozen=True)
class Offering:
    id: str
    capacity: int
    min_grade_ordinal: int
    max_grade_ordinal: int


@dataclass(frozen=True)
class Participant:
    id: str
    grade_ordinal: int


def _required_string(document: dict[str, Any], key: str) -> str:
    value = document.get(key)
    if not isinstance(value, str) or not value:
        raise ContractError(f"{key} must be a non-empty string")
    return value


def parse_request(document: Any) -> tuple[int, float, tuple[Participant, ...], tuple[Offering, ...]]:
    if not isinstance(document, dict):
        raise ContractError("request must be an object")
    if document.get("version") != CONTRACT_VERSION:
        raise ContractError(f"version must be {CONTRACT_VERSION!r}")
    seed = document.get("seed")
    if not isinstance(seed, int) or isinstance(seed, bool) or not -(2**63) <= seed < 2**63:
        raise ContractError("seed must be a signed 64-bit integer")
    limit = document.get("max_deterministic_time")
    if not isinstance(limit, (int, float)) or isinstance(limit, bool) or limit <= 0:
        raise ContractError("max_deterministic_time must be positive")

    raw_offerings = document.get("offerings")
    if not isinstance(raw_offerings, list) or not raw_offerings:
        raise ContractError("offerings must be a non-empty array")
    if any(not isinstance(item, dict) for item in raw_offerings):
        raise ContractError("offerings must contain objects")
    offerings = tuple(
        sorted(
            (
                Offering(
                    _required_string(item, "id"),
                    item.get("capacity"),
                    item.get("min_grade_ordinal"),
                    item.get("max_grade_ordinal"),
                )
                for item in raw_offerings
            ),
            key=lambda value: value.id,
        )
    )
    if len({offering.id for offering in offerings}) != len(offerings):
        raise ContractError("offering ids must be unique")
    if any(
        not isinstance(offering.capacity, int)
        or offering.capacity < 0
        or not isinstance(offering.min_grade_ordinal, int)
        or offering.min_grade_ordinal <= 0
        or not isinstance(offering.max_grade_ordinal, int)
        or offering.max_grade_ordinal < offering.min_grade_ordinal
        for offering in offerings
    ):
        raise ContractError("offerings require non-negative capacity and an ordered positive grade window")

    raw_participants = document.get("participants")
    if not isinstance(raw_participants, list):
        raise ContractError("participants must be an array")
    participants = []
    for item in raw_participants:
        if not isinstance(item, dict):
            raise ContractError("participants must contain objects")
        participant_id = _required_string(item, "id")
        grade_ordinal = item.get("grade_ordinal")
        if not isinstance(grade_ordinal, int) or grade_ordinal <= 0:
            raise ContractError("participants require a positive grade_ordinal")
        participants.append(Participant(participant_id, grade_ordinal))
    participants = tuple(sorted(participants, key=lambda value: value.id))
    if len({participant.id for participant in participants}) != len(participants):
        raise ContractError("participant ids must be unique")
    return seed, float(limit), participants, offerings


def canonical_response(*, seed: int, status: str, assignments: list[dict[str, str]], conflict_diagnostics: list[dict[str, Any]] | None = None) -> dict[str, Any]:
    return {
        "version": CONTRACT_VERSION,
        "seed": seed,
        "status": status,
        "assignments": sorted(assignments, key=lambda assignment: assignment["participant_id"]),
        # The v0 model intentionally emits none. Retaining the field now keeps
        # later conflict-set extraction backward compatible at this boundary.
        "conflict_diagnostics": conflict_diagnostics or [],
    }
