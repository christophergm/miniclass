# Top-level development Makefile for miniclass.
#
# Deliberately minimal: it exposes the handful of commands that get a checkout
# to a working, logged-in local stack and points at the two development
# processes. The full root entry point, including `make check`, is DX-4.
#
# See QUICKSTART.md for the ordered path, and
# docs/adr/0011-local-development-orchestration-and-environment-contract.md for
# why each step is a script rather than a recipe.
.PHONY: setup seed login dev reset

setup: ## Prepare the local development environment
	@./scripts/setup.sh

seed: ## Create the synthetic organisation and bind the local admin identity
	@$(MAKE) -C backend seed

login: ## Mint or refresh the local development bearer token in .env
	@./scripts/login.sh

reset: ## Drop and rebuild the local database, then re-seed and re-login (requires CONFIRM=1)
	@if [ "$(CONFIRM)" != "1" ]; then echo "Refusing to reset the database. Re-run with CONFIRM=1."; exit 1; fi
	@echo "Stop the API first if it is running: its connection pool outlives the schema this replaces."
	@$(MAKE) -C backend reset-db RESET_DB_CONFIRM=1
	@DEV_TOKEN_FORCE=1 ./scripts/login.sh

dev: ## Print how to run the development processes
	@echo "MiniClass runs as two long-lived processes, one per terminal:"
	@echo ""
	@echo "  make -C backend dev          API with hot reload"
	@echo "  cd frontend && bun run dev   app with hot reload"
	@echo ""
	@echo "Run 'make setup', 'make seed' and 'make login' first; see QUICKSTART.md."
	@echo "Verify the stack with ./scripts/smoke-test.sh."
