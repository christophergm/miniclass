"""Stateless, versioned HTTP boundary for the MiniClass solver sidecar."""

from __future__ import annotations

import json
import os
import random
from collections import defaultdict
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from ortools.sat.python import cp_model

from .contract import (
    EXCEPTION_RULE_CAPACITY,
    EXCEPTION_RULE_EXCLUSION,
    EXCEPTION_RULE_GRADE,
    QUALITY_LEVELS,
    AuthorizedPinnedException,
    ContractError,
    Offering,
    Participant,
    PinnedPlacement,
    Placement,
    canonical_response,
    parse_request,
    realized_quality,
)


def solve(document: object) -> dict[str, object]:
    """Solve the v0 feasibility model from its self-contained request snapshot.

    Hard rules remain exactly-one assignment, offering capacity, and grade
    windows (SPEC §17.1). Quality objectives are optimized worst-outcome first
    and fixed at each optimum (SPEC §17.4.2); the seeded decision strategy then
    breaks only genuine objective ties (SPEC §17.8).
    """
    seed, deterministic_limit, high_rank_max, participants, offerings, pins, exclusions, exceptions, _prior_placements = parse_request(document)
    diagnostics = _constraint_diagnostics(pins, exclusions, exceptions, participants, offerings)
    if diagnostics:
        # Failed re-solves deliberately yield no candidate assignments. The
        # caller can therefore keep its current draft intact while displaying
        # the stale pin or exception explanation (SPEC §§17.9–17.10).
        return canonical_response(seed=seed, status="infeasible", assignments=[], conflict_diagnostics=diagnostics)

    model = cp_model.CpModel()
    decisions: dict[tuple[str, str], cp_model.IntVar] = {}
    qualities: dict[tuple[str, str], str] = {}
    exception_rules = {(exception.participant_id, exception.offering_id, exception.rule) for exception in exceptions}
    exclusion_pairs = {(exclusion.participant_id, exclusion.offering_id) for exclusion in exclusions}

    for participant in participants:
        eligible = []
        for offering in offerings:
            key = (participant.id, offering.id)
            grade_allowed = offering.min_grade_ordinal <= participant.grade_ordinal <= offering.max_grade_ordinal
            exclusion_allowed = key not in exclusion_pairs
            if (not grade_allowed and (participant.id, offering.id, EXCEPTION_RULE_GRADE) not in exception_rules) or (
                not exclusion_allowed and (participant.id, offering.id, EXCEPTION_RULE_EXCLUSION) not in exception_rules
            ):
                continue
            if grade_allowed or (participant.id, offering.id, EXCEPTION_RULE_GRADE) in exception_rules:
                decision = model.NewBoolVar(f"assign_{participant.id}_{offering.id}")
                decisions[key] = decision
                qualities[key] = realized_quality(participant, offering, high_rank_max)
                eligible.append(decision)
        # Exactly-one deliberately makes a participant with no eligible offering
        # an infeasible model rather than returning a partial draft.
        model.AddExactlyOne(eligible)

    for offering in offerings:
        model.Add(
            sum(decision for (participant_id, offering_id), decision in decisions.items() if offering_id == offering.id)
            <= max(
                offering.capacity,
                sum(1 for pin in pins if pin.offering_id == offering.id),
            )
        )

    for pin in pins:
        # Pins are solver constraints, not output decoration: their seats are
        # counted by capacity and their quality is fixed before optimization.
        model.Add(decisions[(pin.participant_id, pin.offering_id)] == 1)

    ordered_decisions = sorted(decisions.items())
    random.Random(seed).shuffle(ordered_decisions)
    if ordered_decisions:
        model.AddDecisionStrategy(
            [decision for _, decision in ordered_decisions],
            cp_model.CHOOSE_FIRST,
            cp_model.SELECT_MAX_VALUE,
        )

    solver = _new_solver(seed, deterministic_limit)
    status = cp_model.UNKNOWN
    for quality in QUALITY_LEVELS:
        objective = sum(decision for key, decision in decisions.items() if qualities[key] == quality)
        model.Minimize(objective)
        status = solver.Solve(model)
        if status != cp_model.OPTIMAL:
            break
        # Equality is the lexicographic guard: a later level can never buy a
        # better result by making any preceding quality level worse.
        model.Add(objective == int(solver.ObjectiveValue()))

    if status in (cp_model.OPTIMAL, cp_model.FEASIBLE):
        assignments = [
            {"participant_id": participant_id, "offering_id": offering_id, "realized_quality": qualities[(participant_id, offering_id)]}
            for (participant_id, offering_id), decision in decisions.items()
            if solver.Value(decision)
        ]
        return canonical_response(seed=seed, status="optimal" if status == cp_model.OPTIMAL else "feasible", assignments=assignments)
    if status == cp_model.INFEASIBLE:
        return canonical_response(seed=seed, status="infeasible", assignments=[])
    if status == cp_model.MODEL_INVALID:
        return canonical_response(seed=seed, status="model_invalid", assignments=[])
    return canonical_response(seed=seed, status="unknown", assignments=[])


def _constraint_diagnostics(
    pins: tuple[PinnedPlacement, ...],
    exclusions: tuple[Placement, ...],
    exceptions: tuple[AuthorizedPinnedException, ...],
    participants: tuple[Participant, ...],
    offerings: tuple[Offering, ...],
) -> list[dict[str, object]]:
    """Explain stale constraints before a solve could otherwise hide them."""
    participants_by_id = {participant.id: participant for participant in participants}
    offerings_by_id = {offering.id: offering for offering in offerings}
    diagnostics: list[dict[str, object]] = []
    pins_by_participant: dict[str, list[PinnedPlacement]] = defaultdict(list)
    pins_by_pair = {(pin.participant_id, pin.offering_id) for pin in pins}
    exclusions_by_pair: dict[tuple[str, str], int] = defaultdict(int)

    for exclusion in exclusions:
        exclusions_by_pair[(exclusion.participant_id, exclusion.offering_id)] += 1
        if exclusion.participant_id not in participants_by_id:
            diagnostics.append(_placement_diagnostic("exclusion-participant-not-participating", exclusion))
        if exclusion.offering_id not in offerings_by_id:
            diagnostics.append(_placement_diagnostic("exclusion-offering-not-found", exclusion))
    for pair, count in exclusions_by_pair.items():
        if count > 1:
            diagnostics.append(_placement_diagnostic("exclusion-duplicate", Placement(*pair)))

    exceptions_by_pair_rule: dict[tuple[str, str, str], int] = defaultdict(int)
    for exception in exceptions:
        key = (exception.participant_id, exception.offering_id, exception.rule)
        exceptions_by_pair_rule[key] += 1
        placement = Placement(exception.participant_id, exception.offering_id)
        if (exception.participant_id, exception.offering_id) not in pins_by_pair:
            diagnostics.append(_placement_diagnostic("pinned-exception-pin-not-found", placement))
            continue
        participant = participants_by_id.get(exception.participant_id)
        offering = offerings_by_id.get(exception.offering_id)
        if participant is None:
            diagnostics.append(_placement_diagnostic("pinned-exception-participant-not-participating", placement))
        elif offering is None:
            diagnostics.append(_placement_diagnostic("pinned-exception-offering-not-found", placement))
        elif exception.rule == EXCEPTION_RULE_GRADE and offering.min_grade_ordinal <= participant.grade_ordinal <= offering.max_grade_ordinal:
            diagnostics.append(_placement_diagnostic("pinned-exception-grade-not-needed", placement))
        elif exception.rule == EXCEPTION_RULE_EXCLUSION and (exception.participant_id, exception.offering_id) not in exclusions_by_pair:
            diagnostics.append(_placement_diagnostic("pinned-exception-exclusion-not-needed", placement))
    for (participant_id, offering_id, _rule), count in exceptions_by_pair_rule.items():
        if count > 1:
            diagnostics.append(_placement_diagnostic("pinned-exception-duplicate", Placement(participant_id, offering_id)))

    for pin in pins:
        participant = participants_by_id.get(pin.participant_id)
        offering = offerings_by_id.get(pin.offering_id)
        if participant is None:
            diagnostics.append(_pin_diagnostic("pin-participant-not-participating", pin))
        if offering is None:
            diagnostics.append(_pin_diagnostic("pin-offering-not-found", pin))
        if participant is None or offering is None:
            continue
        if not offering.min_grade_ordinal <= participant.grade_ordinal <= offering.max_grade_ordinal and (
            pin.participant_id,
            pin.offering_id,
            EXCEPTION_RULE_GRADE,
        ) not in exceptions_by_pair_rule:
            diagnostics.append(_pin_diagnostic("pin-grade-out-of-range", pin))
            continue
        if (pin.participant_id, pin.offering_id) in exclusions_by_pair and (
            pin.participant_id,
            pin.offering_id,
            EXCEPTION_RULE_EXCLUSION,
        ) not in exceptions_by_pair_rule:
            diagnostics.append(_pin_diagnostic("pin-excluded", pin))
            continue
        pins_by_participant[pin.participant_id].append(pin)

    for participant_id, participant_pins in pins_by_participant.items():
        if len(participant_pins) > 1:
            diagnostics.append(
                {
                    "code": "pin-conflicting-placement",
                    "participant_ids": [participant_id],
                    "offering_ids": sorted({pin.offering_id for pin in participant_pins}),
                }
            )

    pins_by_offering: dict[str, list[PinnedPlacement]] = defaultdict(list)
    for pin in pins:
        if pin.participant_id in participants_by_id and pin.offering_id in offerings_by_id:
            pins_by_offering[pin.offering_id].append(pin)
    for offering_id, offering_pins in pins_by_offering.items():
        required = max(0, len(offering_pins) - offerings_by_id[offering_id].capacity)
        capacity_exceptions = [
            exception
            for exception in exceptions
            if exception.offering_id == offering_id and exception.rule == EXCEPTION_RULE_CAPACITY and (exception.participant_id, exception.offering_id) in pins_by_pair
        ]
        if len(capacity_exceptions) < required:
            diagnostics.append(
                {
                    "code": "pin-capacity-exceeded",
                    "participant_ids": sorted(pin.participant_id for pin in offering_pins),
                    "offering_ids": [offering_id],
                }
            )
        elif len(capacity_exceptions) > required:
            for exception in capacity_exceptions[required:]:
                diagnostics.append(_placement_diagnostic("pinned-exception-capacity-not-needed", Placement(exception.participant_id, exception.offering_id)))
    return diagnostics


def _pin_diagnostic(code: str, pin: PinnedPlacement) -> dict[str, object]:
    return {"code": code, "participant_ids": [pin.participant_id], "offering_ids": [pin.offering_id]}


def _placement_diagnostic(code: str, placement: Placement) -> dict[str, object]:
    return {"code": code, "participant_ids": [placement.participant_id], "offering_ids": [placement.offering_id]}


def _new_solver(seed: int, deterministic_limit: float) -> cp_model.CpSolver:
    solver = cp_model.CpSolver()
    solver.parameters.num_search_workers = 1
    solver.parameters.search_branching = cp_model.FIXED_SEARCH
    # Presolve can choose a symmetric solution before the fixed strategy runs;
    # disable it so the seeded decision order is the actual tie-breaker.
    solver.parameters.cp_model_presolve = False
    solver.parameters.random_seed = seed & 0x7FFFFFFF
    solver.parameters.max_deterministic_time = deterministic_limit
    return solver


class Handler(BaseHTTPRequestHandler):
    def do_GET(self) -> None:  # noqa: N802
        if self.path != "/health":
            self.send_error(HTTPStatus.NOT_FOUND)
            return
        self._write(HTTPStatus.OK, {"status": "healthy"})

    def do_POST(self) -> None:  # noqa: N802
        if self.path != "/v1/solve":
            self.send_error(HTTPStatus.NOT_FOUND)
            return
        try:
            length = int(self.headers.get("Content-Length", "0"))
            document = json.loads(self.rfile.read(length))
            self._write(HTTPStatus.OK, solve(document))
        except (ContractError, json.JSONDecodeError) as error:
            self._write(HTTPStatus.BAD_REQUEST, {"error": str(error)})

    def _write(self, status: HTTPStatus, body: object) -> None:
        payload = json.dumps(body, separators=(",", ":"), sort_keys=True).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def log_message(self, format: str, *args: object) -> None:  # noqa: A002
        return


def port_from_environment() -> int:
    value = os.environ.get("PORT", "8090")
    try:
        port = int(value)
    except ValueError as error:
        raise ValueError("PORT must be an integer") from error
    if not 1 <= port <= 65535:
        raise ValueError("PORT must be between 1 and 65535")
    return port


def new_server(port: int) -> ThreadingHTTPServer:
    return ThreadingHTTPServer(("0.0.0.0", port), Handler)


def main() -> None:
    new_server(port_from_environment()).serve_forever()


if __name__ == "__main__":
    main()
