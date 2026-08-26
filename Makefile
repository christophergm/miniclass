# Top-level development Makefile for miniclass.
#
# Deliberately minimal: it exposes setup and points at the two development
# processes. The full root entry point, including `make check`, is DX-4.
.PHONY: setup dev

setup: ## Prepare the local development environment
	@./scripts/setup.sh

dev: ## Print how to run the development processes
	@echo "MiniClass runs as two long-lived processes, one per terminal:"
	@echo ""
	@echo "  make -C backend dev          API with hot reload"
	@echo "  cd frontend && bun run dev   app with hot reload"
	@echo ""
	@echo "Run 'make setup' first. Verify the stack with ./scripts/smoke-test.sh."
