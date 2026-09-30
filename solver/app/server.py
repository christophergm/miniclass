"""Stateless, versioned HTTP boundary for the MiniClass solver sidecar."""

from __future__ import annotations

import json
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from .contract import ContractError, canonical_response, parse_request


def solve(document: object) -> dict[str, object]:
    """Validate the v1 boundary without implementing placement semantics.

    Issue #236 owns the sidecar and deterministic contract infrastructure. The
    first CP-SAT feasibility formulation belongs to issue #237, which is blocked
    by this boundary. Preserve the request seed in a canonical response so both
    sides can exercise the real HTTP contract without implying an assignment.
    """
    seed, _, _, _ = parse_request(document)
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
