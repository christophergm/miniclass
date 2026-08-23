.PHONY: help dev build test migrate-up migrate-down migrate-create seed reset-db sqlc install-tools

# Load environment variables
include ../.env
export

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

install-tools: ## Install required Go tools
	@echo "Installing development tools..."
	go install github.com/air-verse/air@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	@echo "Tools installed!"

dev: ## Run API server with hot reload (requires air)
	air -c .air.toml

build: ## Build the API binary
	go build -o bin/api ./cmd/api

migrate-up: ## Run all pending migrations
	go run ./cmd/migrate up

migrate-down: ## Rollback last migration
	go run ./cmd/migrate down

migrate-create: ## Create a new migration (usage: make migrate-create NAME=create_users)
	@if [ -z "$(NAME)" ]; then echo "Error: NAME is required. Usage: make migrate-create NAME=your_migration_name"; exit 1; fi
	goose -dir migrations create $(NAME) sql

seed: ## Load seed data
	@echo "Loading seed data..."
	go run ./cmd/seed

reset-db: ## Drop, recreate, migrate, and seed database
	@echo "Resetting database..."
	@echo "Dropping database..."
	psql "$(DATABASE_URL)" -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@echo "Running migrations..."
	$(MAKE) migrate-up
	@echo "Seeding data..."
	$(MAKE) seed
	@echo "Database reset complete!"

sqlc: ## Generate sqlc code from queries
	sqlc generate

test: ## Run integration tests
	@echo "Running integration tests..."
	go test -v ./tests/integration/... -count=1

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./tests/integration/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
