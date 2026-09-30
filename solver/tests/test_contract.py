import json
from pathlib import Path

from app.contract import CONTRACT_VERSION
from app.server import solve


def test_repeated_fixture_solve_is_byte_identical() -> None:
    request = json.loads((Path(__file__).parent / "fixtures" / "solve-request-v1.json").read_text())
    assert request["version"] == CONTRACT_VERSION
    first = solve(request)
    assert first == solve(request)
    assert first == {"version": CONTRACT_VERSION, "seed": 41, "status": "unknown", "assignments": []}
    assert json.dumps(first, separators=(",", ":"), sort_keys=True) == json.dumps(solve(request), separators=(",", ":"), sort_keys=True)
