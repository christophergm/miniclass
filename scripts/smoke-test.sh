#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
LOG_DIR="${SMOKE_LOG_DIR:-${TMPDIR:-/tmp}/miniclass-smoke-$(date +%Y%m%d-%H%M%S)}"
TIMEOUT_SECONDS="${SMOKE_TIMEOUT_SECONDS:-60}"

backend_pid=""
frontend_pid=""
compose_available=0

die() {
  echo "Smoke test failed: $*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "'$1' is required"
}

cleanup() {
  status=$?

  if [[ "$status" -ne 0 ]]; then
    echo >&2
    echo "Smoke-test logs: $LOG_DIR" >&2
    for log_file in "$LOG_DIR"/*.log; do
      [[ -f "$log_file" ]] || continue
      echo >&2
      echo "--- $(basename "$log_file") ---" >&2
      tail -n 40 "$log_file" >&2 || true
    done
    if [[ "$compose_available" -eq 1 ]]; then
      echo >&2
      echo "--- docker compose logs ---" >&2
      docker compose --env-file "$ENV_FILE" logs --no-color --tail=40 postgres adminer >&2 || true
    fi
  else
    echo "Smoke test passed. Logs: $LOG_DIR"
  fi

  for pid in "$backend_pid" "$frontend_pid"; do
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
    fi
  done

  exit "$status"
}
trap cleanup EXIT

[[ -f "$ENV_FILE" ]] || die "missing $ENV_FILE; run 'cp .env.example .env' first"

require_command docker
require_command curl
require_command go
require_command npm
docker compose version >/dev/null 2>&1 || die "Docker Compose is required"
compose_available=1

set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

POSTGRES_USER="${POSTGRES_USER:-miniclass}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-miniclass_dev_password}"
POSTGRES_DB="${POSTGRES_DB:-miniclass}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
PORT="${PORT:-8080}"
VITE_PORT="${VITE_PORT:-5173}"
DATABASE_URL="${DATABASE_URL:-postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable}"
API_BASE_URL="${API_BASE_URL:-http://localhost:${PORT}}"
API_BASE_URL="${API_BASE_URL%/}"
API_BASE_URL="${API_BASE_URL%/api}"
FRONTEND_URL="http://localhost:${VITE_PORT}"

export DATABASE_URL PORT API_BASE_URL
# The client appends /api/health, so this value is the backend origin.
export VITE_API_URL="$API_BASE_URL"

mkdir -p "$LOG_DIR"

echo "Starting PostgreSQL and Adminer..."
docker compose --env-file "$ENV_FILE" up -d postgres adminer >"$LOG_DIR/compose-up.log" 2>&1

echo "Waiting for PostgreSQL..."
for _ in $(seq 1 "$TIMEOUT_SECONDS"); do
  if docker compose --env-file "$ENV_FILE" exec -T postgres pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" >"$LOG_DIR/postgres-ready.log" 2>&1; then
    break
  fi
  sleep 1
done
docker compose --env-file "$ENV_FILE" exec -T postgres pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" >"$LOG_DIR/postgres-ready.log" 2>&1 \
  || die "PostgreSQL did not become ready within ${TIMEOUT_SECONDS}s"

echo "Applying database migrations..."
(cd "$ROOT_DIR/backend" && go run ./cmd/migrate up) >"$LOG_DIR/migrations.log" 2>&1 \
  || die "database migrations failed"

echo "Starting backend at $API_BASE_URL..."
(cd "$ROOT_DIR/backend" && exec go run ./cmd/api) >"$LOG_DIR/backend.log" 2>&1 &
backend_pid=$!

echo "Waiting for backend health..."
for _ in $(seq 1 "$TIMEOUT_SECONDS"); do
  if curl --fail --silent --show-error "$API_BASE_URL/api/health" >"$LOG_DIR/api-health.json" 2>"$LOG_DIR/api-health.err"; then
    break
  fi
  sleep 1
done

health_response="$(cat "$LOG_DIR/api-health.json" 2>/dev/null || true)"
printf '%s' "$health_response" | grep -Eq '"status"[[:space:]]*:[[:space:]]*"healthy"' \
  || die "GET $API_BASE_URL/api/health did not report a healthy API"
printf '%s' "$health_response" | grep -Eq '"database"[[:space:]]*:[[:space:]]*"connected"' \
  || die "GET $API_BASE_URL/api/health did not report a connected database"
echo "Backend health: $health_response"

echo "Starting frontend at $FRONTEND_URL..."
(cd "$ROOT_DIR/frontend" && exec npm run dev -- --host 127.0.0.1) >"$LOG_DIR/frontend.log" 2>&1 &
frontend_pid=$!

echo "Waiting for frontend route..."
for _ in $(seq 1 "$TIMEOUT_SECONDS"); do
  if curl --fail --silent --show-error "$FRONTEND_URL/health" >"$LOG_DIR/frontend-health.html" 2>"$LOG_DIR/frontend-health.err"; then
    break
  fi
  sleep 1
done
curl --fail --silent --show-error "$FRONTEND_URL/health" >"$LOG_DIR/frontend-health.html" 2>"$LOG_DIR/frontend-health.err" \
  || die "frontend did not serve $FRONTEND_URL/health"

echo
echo "Automated checks passed:"
echo "  - PostgreSQL is ready and migrations are applied"
echo "  - $API_BASE_URL/api/health reports healthy/connected"
echo "  - $FRONTEND_URL/health is served by Vite"
echo
echo "Manual browser check: open $FRONTEND_URL/health and confirm"
echo "  'All systems operational', 'Connected', and the current API version are visible."
echo "Adminer: http://localhost:${ADMINER_PORT:-8081} (server: postgres)"
