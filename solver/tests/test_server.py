from __future__ import annotations

from unittest.mock import Mock, patch

import pytest

from app.server import Handler, new_server, port_from_environment


def test_server_uses_injected_port(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("PORT", "18765")

    assert port_from_environment() == 18765


@pytest.mark.parametrize("port", ["", "not-a-port", "0", "65536"])
def test_server_rejects_invalid_port(monkeypatch: pytest.MonkeyPatch, port: str) -> None:
    monkeypatch.setenv("PORT", port)

    with pytest.raises(ValueError, match="PORT must"):
        port_from_environment()


def test_server_binds_all_interfaces_on_configured_port() -> None:
    with patch("app.server.ThreadingHTTPServer") as http_server:
        new_server(18765)

    http_server.assert_called_once_with(("0.0.0.0", 18765), Handler)


def test_health_endpoint_reports_healthy() -> None:
    handler = object.__new__(Handler)
    handler.path = "/health"
    handler._write = Mock()

    handler.do_GET()

    handler._write.assert_called_once_with(200, {"status": "healthy"})
