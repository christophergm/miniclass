.PHONY: help dev build test format lint generate migrate-up migrate-down migrate-status migrate-create seed reset-db sqlc openapi generated-code-drift migration-round-trip install-tools verify-depguard

GOLANGCI_LINT_VERSION := v1.64.8
SQLC_VERSION := v1.27.0
GOOSE_VERSION := v3.19.2

# Load environment variables
-include ../.env
export

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

install-tools: ## Install required Go tools
	@echo "Installing development tools..."
	go install github.com/air-verse/air@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
	go install github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION)
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@echo "Tools installed!"

dev: ## Run API server with hot reload (requires air)
	air -c .air.toml

build: ## Build the API binary
	go build -o bin/api ./cmd/api

migrate-up: ## Run all pending migrations
	go run ./cmd/migrate up

migrate-down: ## Rollback last migration
	go run ./cmd/migrate down

migrate-status: ## Show which migrations are applied
	go run ./cmd/migrate status

migrate-create: ## Create a new migration (usage: make migrate-create NAME=create_users)
	@if [ -z "$(NAME)" ]; then echo "Error: NAME is required. Usage: make migrate-create NAME=your_migration_name"; exit 1; fi
	goose -dir migrations create $(NAME) sql

SEED_ORGANIZATION_NAME ?= Synthetic Academy
# The one local identity. .env supplies DEV_ADMIN_EMAIL through the include
# above; the fallback keeps `make seed` working in a checkout with no .env yet.
# The subject is derived rather than configured separately, because the invited
# email and the token's email claim must agree or the claim is refused and the
# login silently does not exist. Set SEED_OWNER_SUBJECT= (empty) to seed an
# unclaimed invitation, as a human-claimed flow would.
DEV_ADMIN_EMAIL ?= owner@example.test
SEED_OWNER_EMAIL ?= $(DEV_ADMIN_EMAIL)
SEED_OWNER_SUBJECT ?= local:$(SEED_OWNER_EMAIL)
SEED_HOMEROOM_LABEL ?= homeroom
SEED_CLAIM_BASE_URL ?= http://localhost:5173/claim

seed: ## Create a fresh synthetic organization and roster, and bind the Owner login
	@echo "Creating fresh synthetic seed organization..."
	go run ./cmd/seed \
		-organization-name "$(SEED_ORGANIZATION_NAME)" \
		-owner-email "$(SEED_OWNER_EMAIL)" \
		-owner-subject "$(SEED_OWNER_SUBJECT)" \
		-homeroom-label "$(SEED_HOMEROOM_LABEL)" \
		-claim-base-url "$(SEED_CLAIM_BASE_URL)"

reset-db: ## Drop, recreate, migrate, and seed the local database (requires RESET_DB_CONFIRM=1)
	@if [ "$(RESET_DB_CONFIRM)" != "1" ]; then echo "Refusing to reset database. Re-run with RESET_DB_CONFIRM=1."; exit 1; fi
	@echo "Resetting database..."
	@echo "Dropping database..."
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@echo "Running migrations..."
	$(MAKE) migrate-up
	@echo "Seeding data..."
	$(MAKE) seed
	@echo "Database reset complete!"

openapi: ## Generate the committed OpenAPI contract
	go run ./cmd/openapi

generated-code-drift: ## Verify generated OpenAPI output is deterministic and committed
	@temp_dir="$${TMPDIR:-$${TMP:-$${TEMP:-$${RUNNER_TEMP:-/tmp}}}}"; \
	first="$$(mktemp "$$temp_dir/miniclass-openapi.XXXXXX")"; \
	second="$$(mktemp "$$temp_dir/miniclass-openapi.XXXXXX")"; \
	trap 'rm -f "$$first" "$$second"' EXIT; \
	go run ./cmd/openapi -output "$$first"; \
	go run ./cmd/openapi -output "$$second"; \
	cmp "$$first" "$$second"; \
	cmp "$$second" openapi.json

migration-round-trip: ## Apply, roll back, and reapply every migration on a scratch database
	@if [ -z "$(MIGRATION_ROUNDTRIP_DATABASE_URL)" ]; then \
		echo "MIGRATION_ROUNDTRIP_DATABASE_URL is unset. Add the MIGRATION_ROUNDTRIP_* keys from .env.example to your .env."; \
		exit 1; \
	fi
	./scripts/migration-round-trip.sh

test: ## Run all backend unit and integration tests
	@echo "Running backend tests..."
	go test -race -v ./... -count=1

format: ## Verify Go formatting and run the Go vet analyzer
	@files="$$(gofmt -l .)"; \
	if [ -n "$$files" ]; then \
		echo "Unformatted Go files:"; \
		echo "$$files"; \
		exit 1; \
	fi
	go vet ./...

lint: ## Run the pinned golangci-lint configuration
	@command -v golangci-lint >/dev/null || { echo "golangci-lint $(GOLANGCI_LINT_VERSION) is required; run make install-tools"; exit 1; }
	@golangci-lint version | grep -F "$(GOLANGCI_LINT_VERSION)" >/dev/null || { echo "golangci-lint $(GOLANGCI_LINT_VERSION) is required"; golangci-lint version; exit 1; }
	golangci-lint run
	$(MAKE) verify-depguard

verify-depguard: ## Prove protected imports fail outside their allowlisted packages
	./scripts/verify-depguard.sh

sqlc: ## Generate sqlc code from queries
	@command -v sqlc >/dev/null || { echo "sqlc $(SQLC_VERSION) is required; run make install-tools"; exit 1; }
	@# The version is part of the generated output, so an unpinned sqlc rewrites
	@# every file in internal/db/gen and the drift gate blames the change under
	@# review. Refuse instead, the way the lint target refuses a stray
	@# golangci-lint; CI installs exactly this version.
	@sqlc version | grep -F "$(SQLC_VERSION)" >/dev/null || { echo "sqlc $(SQLC_VERSION) is required; run make install-tools"; sqlc version; exit 1; }
	@if [ -d sql/queries ]; then sqlc generate; else echo "No sql/queries directory; sqlc generation skipped."; fi

generate: ## Regenerate all committed backend artifacts
	$(MAKE) sqlc
	$(MAKE) openapi
	go generate ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./tests/integration/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
