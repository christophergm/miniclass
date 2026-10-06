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
    DIAGNOSTIC_SCOPE_GLOBAL_CONFLICT,
    DIAGNOSTIC_SCOPE_OFFERING_OBSTACLE,
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
    and fixed at each optimum (SPEC §17.4.2). Among those optima, the solver
    preserves unpinned prior placements where it can (SPEC §17.9); the seeded
    decision strategy then breaks only genuine remaining ties (SPEC §17.8).
    """
    seed, deterministic_limit, high_rank_max, participants, offerings, pins, exclusions, exceptions, prior_placements = parse_request(document)
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

    if status == cp_model.OPTIMAL:
        pinned_participants = {pin.participant_id for pin in pins}
        retained_placements = sum(
            decision
            for placement in prior_placements
            if placement.participant_id not in pinned_participants
            if (decision := decisions.get((placement.participant_id, placement.offering_id))) is not None
        )
        # Stability is deliberately below every quality level: it cannot retain
        # a placement by buying even one worse preference outcome. Pins are
        # excluded because they are fixed input constraints, not a choice to
        # preserve. Baseline rows that are no longer eligible are inert.
        model.Maximize(retained_placements)
        status = solver.Solve(model)
        if status == cp_model.OPTIMAL:
            model.Add(retained_placements == int(solver.ObjectiveValue()))

    if status in (cp_model.OPTIMAL, cp_model.FEASIBLE):
        assignments = [
            {"participant_id": participant_id, "offering_id": offering_id, "realized_quality": qualities[(participant_id, offering_id)]}
            for (participant_id, offering_id), decision in decisions.items()
            if solver.Value(decision)
        ]
        return canonical_response(seed=seed, status="optimal" if status == cp_model.OPTIMAL else "feasible", assignments=assignments)
    if status == cp_model.INFEASIBLE:
        # The feasibility model has no rules beyond the graph represented
        # below. Reporting the deficient eligibility/capacity set therefore
        # keeps this path actionable instead of returning a bare infeasibility
        # (SPEC §17.10).
        return canonical_response(
            seed=seed,
            status="infeasible",
            assignments=[],
            conflict_diagnostics=_infeasibility_diagnostics(participants, offerings, pins, exclusions, exceptions),
        )
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
                _diagnostic(
                    "pin-conflicting-placement",
                    DIAGNOSTIC_SCOPE_GLOBAL_CONFLICT,
                    [participant_id],
                    [pin.offering_id for pin in participant_pins],
                )
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
                _diagnostic(
                    "pin-capacity-exceeded",
                    DIAGNOSTIC_SCOPE_GLOBAL_CONFLICT,
                    [pin.participant_id for pin in offering_pins],
                    [offering_id],
                    required_capacity=len(offering_pins),
                    available_capacity=offerings_by_id[offering_id].capacity + len(capacity_exceptions),
                )
            )
        elif len(capacity_exceptions) > required:
            for exception in capacity_exceptions[required:]:
                diagnostics.append(_placement_diagnostic("pinned-exception-capacity-not-needed", Placement(exception.participant_id, exception.offering_id)))
    return diagnostics


def _pin_diagnostic(code: str, pin: PinnedPlacement) -> dict[str, object]:
    return _diagnostic(code, DIAGNOSTIC_SCOPE_GLOBAL_CONFLICT, [pin.participant_id], [pin.offering_id])


def _placement_diagnostic(code: str, placement: Placement) -> dict[str, object]:
    return _diagnostic(code, DIAGNOSTIC_SCOPE_GLOBAL_CONFLICT, [placement.participant_id], [placement.offering_id])


def _diagnostic(
    code: str,
    scope: str,
    participant_ids: list[str],
    offering_ids: list[str],
    *,
    required_capacity: int | None = None,
    available_capacity: int | None = None,
) -> dict[str, object]:
    diagnostic: dict[str, object] = {
        "code": code,
        "scope": scope,
        "participant_ids": sorted(set(participant_ids)),
        "offering_ids": sorted(set(offering_ids)),
    }
    if required_capacity is not None:
        diagnostic["required_capacity"] = required_capacity
    if available_capacity is not None:
        diagnostic["available_capacity"] = available_capacity
    return diagnostic


def _infeasibility_diagnostics(
    participants: tuple[Participant, ...],
    offerings: tuple[Offering, ...],
    pins: tuple[PinnedPlacement, ...],
    exclusions: tuple[Placement, ...],
    exceptions: tuple[AuthorizedPinnedException, ...],
) -> list[dict[str, object]]:
    """Return local eligibility obstacles and near-minimal capacity conflicts.

    Pins were already validated before this function is reached. They consume
    their seats first, while a capacity exception consumes only its authorised
    excess and never creates an ordinary spare seat (SPEC §§16.3, 17.9).
    """
    offering_by_id = {offering.id: offering for offering in offerings}
    pinned_by_participant = {pin.participant_id: pin.offering_id for pin in pins}
    pinned_by_offering: dict[str, int] = defaultdict(int)
    for pin in pins:
        pinned_by_offering[pin.offering_id] += 1
    exception_rules = {(exception.participant_id, exception.offering_id, exception.rule) for exception in exceptions}
    exclusion_pairs = {(exclusion.participant_id, exclusion.offering_id) for exclusion in exclusions}

    remaining_capacity = {
        offering.id: max(0, offering.capacity - pinned_by_offering[offering.id])
        for offering in offerings
    }
    eligible: dict[str, list[str]] = {}
    diagnostics: list[dict[str, object]] = []
    for participant in participants:
        if participant.id in pinned_by_participant:
            continue
        eligible_offerings = []
        grade_obstacles = []
        exclusion_obstacles = []
        for offering in offerings:
            pair = (participant.id, offering.id)
            grade_allowed = offering.min_grade_ordinal <= participant.grade_ordinal <= offering.max_grade_ordinal
            excluded = pair in exclusion_pairs
            if not grade_allowed:
                grade_obstacles.append(offering.id)
            if excluded:
                exclusion_obstacles.append(offering.id)
            if (not grade_allowed and (participant.id, offering.id, EXCEPTION_RULE_GRADE) not in exception_rules) or (
                excluded and (participant.id, offering.id, EXCEPTION_RULE_EXCLUSION) not in exception_rules
            ):
                continue
            eligible_offerings.append(offering.id)
        if not eligible_offerings:
            if grade_obstacles:
                diagnostics.append(_diagnostic("grade-eligibility-obstacle", DIAGNOSTIC_SCOPE_OFFERING_OBSTACLE, [participant.id], grade_obstacles))
            if exclusion_obstacles:
                diagnostics.append(_diagnostic("exclusion-obstacle", DIAGNOSTIC_SCOPE_OFFERING_OBSTACLE, [participant.id], exclusion_obstacles))
            diagnostics.append(_diagnostic("no-eligible-offering", DIAGNOSTIC_SCOPE_OFFERING_OBSTACLE, [participant.id], list(offering_by_id)))
            continue
        eligible[participant.id] = eligible_offerings

    match, slot_owner, slots_by_offering = _maximum_capacity_matching(eligible, remaining_capacity)
    unmatched = sorted(participant_id for participant_id in eligible if participant_id not in match)
    if not unmatched:
        return diagnostics

    # Alternating reachability from each unmatched participant identifies a
    # Hall-deficient component. Each is a focused, near-minimal global conflict
    # rather than a misleading report about every offering in the session.
    seen_conflicts: set[tuple[tuple[str, ...], tuple[str, ...]]] = set()
    for start in unmatched:
        reachable_participants = {start}
        reachable_slots: set[tuple[str, int]] = set()
        pending = [start]
        while pending:
            participant_id = pending.pop()
            for offering_id in eligible[participant_id]:
                for slot in slots_by_offering[offering_id]:
                    if slot in reachable_slots:
                        continue
                    reachable_slots.add(slot)
                    owner = slot_owner.get(slot)
                    if owner is not None and owner not in reachable_participants:
                        reachable_participants.add(owner)
                        pending.append(owner)
        involved_offerings = sorted({offering_id for offering_id, _ in reachable_slots})
        signature = (tuple(sorted(reachable_participants)), tuple(involved_offerings))
        if signature in seen_conflicts:
            continue
        seen_conflicts.add(signature)
        diagnostics.append(
            _diagnostic(
                "capacity-shortage",
                DIAGNOSTIC_SCOPE_GLOBAL_CONFLICT,
                list(signature[0]),
                list(signature[1]),
                required_capacity=len(signature[0]),
                available_capacity=len(reachable_slots),
            )
        )
    return diagnostics


def _maximum_capacity_matching(
    eligible: dict[str, list[str]], remaining_capacity: dict[str, int]
) -> tuple[dict[str, tuple[str, int]], dict[tuple[str, int], str], dict[str, list[tuple[str, int]]]]:
    """Find a deterministic maximum bipartite matching after pins consume seats."""
    slots_by_offering = {
        offering_id: [(offering_id, index) for index in range(capacity)]
        for offering_id, capacity in remaining_capacity.items()
    }
    slot_owner: dict[tuple[str, int], str] = {}
    match: dict[str, tuple[str, int]] = {}

    def assign(participant_id: str, visited: set[tuple[str, int]]) -> bool:
        for offering_id in eligible[participant_id]:
            for slot in slots_by_offering[offering_id]:
                if slot in visited:
                    continue
                visited.add(slot)
                owner = slot_owner.get(slot)
                if owner is None or assign(owner, visited):
                    slot_owner[slot] = participant_id
                    match[participant_id] = slot
                    return True
        return False

    for participant_id in sorted(eligible):
        assign(participant_id, set())
    return match, slot_owner, slots_by_offering


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
