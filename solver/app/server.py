"""Stateless, versioned HTTP boundary for the MiniClass solver sidecar."""

from __future__ import annotations

import json
import random
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from ortools.sat.python import cp_model

from .contract import ContractError, canonical_response, parse_request


def solve(document: object) -> dict[str, object]:
    """Solve the v0 feasibility model from its self-contained request snapshot.

    The only v0 hard rules are exactly-one assignment, offering capacity, and
    grade windows (SPEC §17.1). A seeded, fixed CP-SAT decision strategy is the
    explicit tie-breaker until preference objectives arrive in the next phase.
    """
    seed, deterministic_limit, participants, offerings = parse_request(document)
    model = cp_model.CpModel()
    decisions: dict[tuple[str, str], cp_model.IntVar] = {}

    for participant in participants:
        eligible = []
        for offering in offerings:
            if offering.min_grade_ordinal <= participant.grade_ordinal <= offering.max_grade_ordinal:
                decision = model.NewBoolVar(f"assign_{participant.id}_{offering.id}")
                decisions[(participant.id, offering.id)] = decision
                eligible.append(decision)
        # Exactly-one deliberately makes a participant with no eligible offering
        # an infeasible model rather than returning a partial draft.
        model.AddExactlyOne(eligible)

    for offering in offerings:
        model.Add(
            sum(decision for (participant_id, offering_id), decision in decisions.items() if offering_id == offering.id)
            <= offering.capacity
        )

    ordered_decisions = sorted(decisions.items())
    random.Random(seed).shuffle(ordered_decisions)
    if ordered_decisions:
        model.AddDecisionStrategy(
            [decision for _, decision in ordered_decisions],
            cp_model.CHOOSE_FIRST,
            cp_model.SELECT_MAX_VALUE,
        )

    solver = cp_model.CpSolver()
    solver.parameters.num_search_workers = 1
    solver.parameters.search_branching = cp_model.FIXED_SEARCH
    # Presolve can choose a symmetric solution before the fixed strategy runs;
    # disable it so the seeded decision order is the actual v0 tie-breaker.
    solver.parameters.cp_model_presolve = False
    solver.parameters.random_seed = seed & 0x7FFFFFFF
    solver.parameters.max_deterministic_time = deterministic_limit
    status = solver.Solve(model)

    if status in (cp_model.OPTIMAL, cp_model.FEASIBLE):
        assignments = [
            {"participant_id": participant_id, "offering_id": offering_id}
            for (participant_id, offering_id), decision in decisions.items()
            if solver.Value(decision)
        ]
        return canonical_response(seed=seed, status="optimal" if status == cp_model.OPTIMAL else "feasible", assignments=assignments)
    if status == cp_model.INFEASIBLE:
        return canonical_response(seed=seed, status="infeasible", assignments=[])
    if status == cp_model.MODEL_INVALID:
        return canonical_response(seed=seed, status="model_invalid", assignments=[])
    return canonical_response(seed=seed, status="unknown", assignments=[])


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


def main() -> None:
    ThreadingHTTPServer(("0.0.0.0", 8090), Handler).serve_forever()


if __name__ == "__main__":
    main()
