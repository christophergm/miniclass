# Top-level development Makefile for miniclass
.PHONY: dev stop ensure

DEV_SCRIPT := ./scripts/dev.sh
STOP_SCRIPT := ./scripts/stop.sh

dev: ensure
	@echo "Starting local development environment..."
	@sh $(DEV_SCRIPT)

ensure:
	@echo "Ensuring prerequisite tools are available (docker, openssl, node/bun)..."
	@command -v docker >/dev/null 2>&1 || (echo "docker is required" >&2; exit 1)
	@command -v openssl >/dev/null 2>&1 || (echo "openssl is required" >&2; exit 1)
	@command -v go >/dev/null 2>&1 || (echo "go is required to run backend (go run). If you prefer to use air, install air and dev will use it." >&2; exit 1)
	@echo "Prerequisites look good."
