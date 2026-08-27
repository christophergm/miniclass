# MiniClass root Makefile — the single entry point for local development.
#
# Every target is a delegation: `$(MAKE) -C backend <target>`, `cd frontend &&
# bun run <script>`, or a script under scripts/. There is exactly one
# implementation of every action, and backend/Makefile's target names are
# unchanged, so WORKFLOW.md's contract and the CI jobs that use
# `working-directory: backend` stay valid.
#
# Names are noun-first and grouped so that `make help` is self-documenting and
# the component boundaries are visible. The four quality-gate verbs — test,
# lint, format, check — stay bare, because they are the ones typed most.
#
# See docs/adr/0011-local-development-orchestration-and-environment-contract.md
# for why development runs as two unsupervised terminals, and QUICKSTART.md for
# the ordered path from a fresh clone.

.DEFAULT_GOAL := help

# Ports, for the signposts below. Every delegate loads .env itself — Make's
# `-include ../.env` in backend/Makefile, `load_env` in scripts/*.sh, and Docker
# Compose reading the project directory — so nothing here is exported. `?=`
# repeats .env.example's defaults only for a checkout that has no .env yet.
-include .env
PORT ?= 8080
VITE_PORT ?= 5173

# The committed output of `make generate`. CI checks the whole tree with
# `git diff --exit-code` because its checkout is clean; locally that would fail
# on any work in progress, so the drift gate is scoped to these paths instead.
# The claim is the same one: generated output is deterministic and committed.
GENERATED_PATHS := backend/internal/db/gen backend/openapi.json

# gate runs one CI-equivalent check. On failure it names the CI check the gate
# maps to, so a local failure reads the way the CI summary will, and then the
# remedy, so the next command is not a guess.
#
#   $(1) CI check name   $(2) command   $(3) remedy
#
# Arguments must not contain a comma: $(call) splits on them.
define gate
@echo ""
@echo "==> $(1)"
@$(2) || { echo ""; echo "FAILED: $(1) (CI check: \"$(1)\") — $(3)"; exit 1; }
endef

.PHONY: help setup tools-install generate smoke \
	db-up db-down db-migrate db-rollback db-status db-migration-new db-seed db-reset \
	dev dev-backend dev-frontend token-mint \
	test test-backend test-frontend test-migrations \
	lint lint-backend lint-frontend format build-frontend check

##@ Setup

help: ## List every command, grouped
	@echo "MiniClass. Every command below runs from the repository root."
	@awk 'BEGIN { FS = ":.*## " } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5); next } \
		/^[a-zA-Z0-9_-]+:.*## / { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' \
		$(firstword $(MAKEFILE_LIST))
	@echo ""
	@echo "First time here? QUICKSTART.md is the ordered path."

setup: ## Prepare a checkout: .env, signing keys, bun install, PostgreSQL, migrations
	@./scripts/setup.sh

tools-install: ## Install the pinned Go tools (air, sqlc, goose, golangci-lint)
	@$(MAKE) -C backend tools-install

generate: ## Regenerate the committed backend artifacts (sqlc, openapi.json)
	@$(MAKE) -C backend generate

smoke: ## Run the full-stack smoke test in throwaway processes
	@./scripts/smoke-test.sh

##@ Database

db-up: ## Start PostgreSQL and wait for it to be healthy
	@docker compose up -d --wait postgres

db-down: ## Stop the local database services; the data volume survives
	@docker compose down

db-migrate: ## Apply every pending migration
	@$(MAKE) -C backend migrate-up

db-rollback: ## Roll back the most recent migration
	@$(MAKE) -C backend migrate-down

db-status: ## Show which migrations are applied
	@$(MAKE) -C backend migrate-status

db-migration-new: ## Create a migration (make db-migration-new NAME=add_widgets)
	@if [ -z "$(NAME)" ]; then \
		echo "NAME is required: make db-migration-new NAME=add_widgets"; \
		exit 1; \
	fi
	@$(MAKE) -C backend migrate-create NAME=$(NAME)

db-seed: ## Create a fresh synthetic organisation and bind the local admin login
	@$(MAKE) -C backend seed

db-reset: ## Drop, migrate, seed, and refresh the login (make db-reset CONFIRM=1)
	@if [ "$(CONFIRM)" != "1" ]; then \
		echo "Refusing to reset: this destroys every row in DATABASE_URL."; \
		echo "Re-run with the confirmation: make db-reset CONFIRM=1"; \
		exit 1; \
	fi
	@echo "Stop the API first if it is running: its connection pool outlives the schema this replaces."
	@$(MAKE) -C backend reset-db RESET_DB_CONFIRM=1
	@$(MAKE) token-mint

##@ Development

dev: ## Print how to run the two development processes
	@echo "MiniClass runs as two long-lived processes, one per terminal:"
	@echo ""
	@echo "  make dev-backend     API on http://localhost:$(PORT)"
	@echo "  make dev-frontend    app on http://localhost:$(VITE_PORT)"
	@echo ""
	@echo "Nothing supervises them, so each hot-reloads and logs to its own terminal."
	@echo "A checkout with no data yet needs 'make setup' and 'make db-seed' first."

dev-backend: db-up ## Run the API with hot reload; needs PostgreSQL
	@$(MAKE) -C backend dev

dev-frontend: token-mint ## Run the Vite dev server with hot reload; needs a fresh dev token
	@cd frontend && bun run dev

token-mint: ## Refresh VITE_DEV_TOKEN in .env when it is stale (FORCE=1 always mints)
	@DEV_TOKEN_FORCE=$(if $(filter 1,$(FORCE)),1,0) ./scripts/login.sh

##@ Quality gates

test: test-backend test-frontend ## Run both test suites

test-backend: db-up ## Run the Go unit and integration tests
	@$(MAKE) -C backend test

test-frontend: ## Run the frontend tests once
	@cd frontend && bun run test -- --run

test-migrations: ## Apply, roll back, and reapply every migration on a scratch database
	@$(MAKE) -C backend migration-round-trip

lint: lint-backend lint-frontend ## Lint both components

lint-backend: ## Run golangci-lint and the depguard boundary proof
	@$(MAKE) -C backend lint

lint-frontend: ## Run ESLint
	@cd frontend && bun run lint

format: ## Check Go formatting and run the vet analyzer
	@$(MAKE) -C backend format

build-frontend: ## Type-check and build the production frontend bundle
	@cd frontend && bun run build

check: db-up ## Run all nine CI gates in CI order, failing fast
	@echo "Running the nine gates CI publishes. Fail-fast; each failure names its CI check."
	$(call gate,Backend tests,$(MAKE) test-backend,fix the failing test)
	$(call gate,Backend lint,$(MAKE) lint-backend,fix the reported findings then rerun 'make lint-backend')
	$(call gate,Backend format,$(MAKE) format,run 'gofmt -w .' in backend/ and address the vet findings)
	$(call gate,Generated code drift,$(MAKE) generate && $(MAKE) -C backend generated-code-drift && git diff --exit-code -- $(GENERATED_PATHS),run 'make generate' and commit the result)
	$(call gate,Migration round-trip,$(MAKE) test-migrations,the down migration does not restore the schema its up migration replaced)
	$(call gate,Frontend tests,$(MAKE) test-frontend,fix the failing test)
	$(call gate,Frontend build,$(MAKE) build-frontend,fix the type or build error above)
	$(call gate,Frontend lint,$(MAKE) lint-frontend,fix the reported findings then rerun 'make lint-frontend')
	$(call gate,Repository formatting,git diff --check,remove the trailing whitespace or conflict marker it names)
	@echo ""
	@echo "All nine gates passed. This is what CI runs."
