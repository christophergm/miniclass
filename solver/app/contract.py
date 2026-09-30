"""Versioned, self-contained solver wire contract (SPEC §§17.3, 17.8)."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

CONTRACT_VERSION = "v1"

QUALITY_TOP = "top"
QUALITY_HIGH = "high"
QUALITY_ACCEPTABLE = "acceptable"
QUALITY_NEUTRAL = "neutral"
QUALITY_UNWANTED = "unwanted"
QUALITY_LEVELS = (
    QUALITY_UNWANTED,
    QUALITY_NEUTRAL,
    QUALITY_ACCEPTABLE,
    QUALITY_HIGH,
)

RANKED_RESPONSE = "ranked"
INTERESTED_RESPONSE = "interested"
NOT_INTERESTED_RESPONSE = "not_interested"
VERY_INTERESTED_RATING = "very_interested"


class ContractError(ValueError):
    pass


@dataclass(frozen=True)
class Offering:
    id: str
    capacity: int
    min_grade_ordinal: int
    max_grade_ordinal: int
    interest_area_id: str | None


@dataclass(frozen=True)
class RankedChoice:
    offering_id: str
    response: str
    rank: int | None


@dataclass(frozen=True)
class RankedChoices:
    choices: tuple[RankedChoice, ...]


@dataclass(frozen=True)
class InterestRating:
    interest_area_id: str
    rating: str


@dataclass(frozen=True)
class Participant:
    id: str
    grade_ordinal: int
    ranked_choices: RankedChoices | None
    interest_profile: tuple[InterestRating, ...]


def _required_string(document: dict[str, Any], key: str) -> str:
    value = document.get(key)
    if not isinstance(value, str) or not value:
        raise ContractError(f"{key} must be a non-empty string")
    return value


def parse_request(document: Any) -> tuple[int, float, int, tuple[Participant, ...], tuple[Offering, ...]]:
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
    raw_quality_config = document.get("quality_config")
    if not isinstance(raw_quality_config, dict):
        raise ContractError("quality_config must be an object")
    high_rank_max = raw_quality_config.get("high_rank_max")
    if not isinstance(high_rank_max, int) or isinstance(high_rank_max, bool) or high_rank_max < 2:
        raise ContractError("quality_config.high_rank_max must be an integer of at least two")

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
                    item.get("interest_area_id"),
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
    if any(offering.interest_area_id is not None and (not isinstance(offering.interest_area_id, str) or not offering.interest_area_id) for offering in offerings):
        raise ContractError("offering interest_area_id must be a non-empty string when present")

    raw_participants = document.get("participants")
    if not isinstance(raw_participants, list):
        raise ContractError("participants must be an array")
    offering_ids = {offering.id for offering in offerings}
    participants = []
    for item in raw_participants:
        if not isinstance(item, dict):
            raise ContractError("participants must contain objects")
        participant_id = _required_string(item, "id")
        grade_ordinal = item.get("grade_ordinal")
        if not isinstance(grade_ordinal, int) or grade_ordinal <= 0:
            raise ContractError("participants require a positive grade_ordinal")
        ranked_choices = _parse_ranked_choices(item, offering_ids)
        interest_profile = _parse_interest_profile(item)
        participants.append(Participant(participant_id, grade_ordinal, ranked_choices, interest_profile))
    participants = tuple(sorted(participants, key=lambda value: value.id))
    if len({participant.id for participant in participants}) != len(participants):
        raise ContractError("participant ids must be unique")
    return seed, float(limit), high_rank_max, participants, offerings


def _parse_ranked_choices(item: dict[str, Any], offering_ids: set[str]) -> RankedChoices | None:
    if "ranked_choices" not in item or item["ranked_choices"] is None:
        return None
    raw_submission = item["ranked_choices"]
    if not isinstance(raw_submission, dict):
        raise ContractError("ranked_choices must be an object when present")
    raw_choices = raw_submission.get("choices")
    if not isinstance(raw_choices, list) or any(not isinstance(choice, dict) for choice in raw_choices):
        raise ContractError("ranked_choices.choices must be an array of objects")
    choices = []
    seen_offerings = set()
    seen_ranks = set()
    for raw_choice in raw_choices:
        offering_id = _required_string(raw_choice, "offering_id")
        response = raw_choice.get("response")
        rank = raw_choice.get("rank")
        if offering_id not in offering_ids:
            raise ContractError("ranked choices must reference request offerings")
        if offering_id in seen_offerings:
            raise ContractError("ranked choices must contain at most one response per offering")
        seen_offerings.add(offering_id)
        if response == RANKED_RESPONSE:
            if not isinstance(rank, int) or isinstance(rank, bool) or rank <= 0:
                raise ContractError("ranked choices require a positive rank for ranked responses")
            if rank in seen_ranks:
                raise ContractError("ranked choice ranks must be unique per participant")
            seen_ranks.add(rank)
        elif response in (INTERESTED_RESPONSE, NOT_INTERESTED_RESPONSE):
            if rank is not None:
                raise ContractError("only ranked responses may include rank")
        else:
            raise ContractError("ranked choice response must be ranked, interested, or not_interested")
        choices.append(RankedChoice(offering_id, response, rank))
    return RankedChoices(tuple(sorted(choices, key=lambda choice: choice.offering_id)))


def _parse_interest_profile(item: dict[str, Any]) -> tuple[InterestRating, ...]:
    raw_profile = item.get("interest_profile", [])
    if not isinstance(raw_profile, list) or any(not isinstance(rating, dict) for rating in raw_profile):
        raise ContractError("interest_profile must be an array of objects")
    profile = []
    seen_areas = set()
    for raw_rating in raw_profile:
        interest_area_id = _required_string(raw_rating, "interest_area_id")
        rating = raw_rating.get("rating")
        if interest_area_id in seen_areas:
            raise ContractError("interest_profile must contain at most one rating per interest area")
        seen_areas.add(interest_area_id)
        if rating not in (VERY_INTERESTED_RATING, INTERESTED_RESPONSE, NOT_INTERESTED_RESPONSE):
            raise ContractError("interest profile rating must be very_interested, interested, or not_interested")
        profile.append(InterestRating(interest_area_id, rating))
    return tuple(sorted(profile, key=lambda rating: rating.interest_area_id))


def realized_quality(participant: Participant, offering: Offering, high_rank_max: int) -> str:
    """Map the governing preference model onto SPEC §17.4's quality scale."""
    if participant.ranked_choices is not None:
        choices = {choice.offering_id: choice for choice in participant.ranked_choices.choices}
        choice = choices.get(offering.id)
        if choice is None:
            return QUALITY_NEUTRAL
        if choice.response == NOT_INTERESTED_RESPONSE:
            return QUALITY_UNWANTED
        if choice.response == INTERESTED_RESPONSE:
            return QUALITY_ACCEPTABLE
        if choice.rank == 1:
            return QUALITY_TOP
        if choice.rank is not None and choice.rank <= high_rank_max:
            return QUALITY_HIGH
        return QUALITY_ACCEPTABLE

    if offering.interest_area_id is None:
        return QUALITY_NEUTRAL
    ratings = {rating.interest_area_id: rating.rating for rating in participant.interest_profile}
    rating = ratings.get(offering.interest_area_id)
    if rating == VERY_INTERESTED_RATING:
        return QUALITY_HIGH
    if rating == INTERESTED_RESPONSE:
        return QUALITY_ACCEPTABLE
    if rating == NOT_INTERESTED_RESPONSE:
        return QUALITY_UNWANTED
    return QUALITY_NEUTRAL


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
