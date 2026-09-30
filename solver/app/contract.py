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


@dataclass(frozen=True)
class Participant:
    id: str
    acceptable_offering_ids: tuple[str, ...]


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
    offerings = tuple(sorted((Offering(_required_string(item, "id"), item.get("capacity")) for item in raw_offerings), key=lambda value: value.id))
    if len({offering.id for offering in offerings}) != len(offerings) or any(not isinstance(offering.capacity, int) or offering.capacity < 0 for offering in offerings):
        raise ContractError("offering ids must be unique and capacities must be non-negative integers")

    raw_participants = document.get("participants")
    if not isinstance(raw_participants, list):
        raise ContractError("participants must be an array")
    participants = []
    offering_ids = {offering.id for offering in offerings}
    for item in raw_participants:
        participant_id = _required_string(item, "id")
        choices = item.get("acceptable_offering_ids")
        if not isinstance(choices, list) or any(not isinstance(choice, str) or not choice for choice in choices):
            raise ContractError("acceptable_offering_ids must be an array of non-empty strings")
        if len(set(choices)) != len(choices) or not set(choices).issubset(offering_ids):
            raise ContractError("acceptable_offering_ids must be unique offering identifiers")
        participants.append(Participant(participant_id, tuple(sorted(choices))))
    participants = tuple(sorted(participants, key=lambda value: value.id))
    if len({participant.id for participant in participants}) != len(participants):
        raise ContractError("participant ids must be unique")
    return seed, float(limit), participants, offerings


def canonical_response(*, seed: int, status: str, assignments: list[dict[str, str]]) -> dict[str, Any]:
    return {
        "version": CONTRACT_VERSION,
        "seed": seed,
        "status": status,
        "assignments": sorted(assignments, key=lambda assignment: assignment["participant_id"]),
    }
