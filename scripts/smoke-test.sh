#!/usr/bin/env bash
set -Eeuo pipefail

MINICLASS_SCRIPT="scripts/smoke-test.sh"
ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib.sh
. "$ROOT_DIR/scripts/lib.sh"

ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
LOG_DIR="$(log_dir smoke "${SMOKE_LOG_DIR:-}")"
TIMEOUT_SECONDS="${SMOKE_TIMEOUT_SECONDS:-60}"

backend_pid=""
frontend_pid=""
compose_available=0

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
    stop_process_tree "$pid"
  done

  exit "$status"
}
trap cleanup EXIT

require_command docker curl go bun awk
docker compose version >/dev/null 2>&1 || die "Docker Compose is required"
compose_available=1

# load_env checks the file before sourcing it. A value containing a space, such
# as an inline PEM key, is reported by name instead of reaching the shell as
# "EC: command not found" and killing this script under set -e.
load_env "$ENV_FILE"

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
# The browser reaches the API through the Vite dev proxy, so the client's base
# URL stays empty and only the node-side proxy target names the backend origin.
# frontend/package.json's dev script re-sources .env, so these two agree with it
# only because both are derived from the same file.
export VITE_API_URL=""
export API_PROXY_TARGET="$API_BASE_URL"

require_free_port "$PORT" "API"
require_free_port "$VITE_PORT" "frontend dev server"

# Job control puts each background job in its own process group, so cleanup can
# signal the whole tree rather than just the launcher. See stop_process_tree.
set -m

echo "Starting PostgreSQL and Adminer..."
docker compose --env-file "$ENV_FILE" up -d postgres adminer >"$LOG_DIR/compose-up.log" 2>&1

echo "Waiting for PostgreSQL..."
wait_for_postgres "$ENV_FILE" "$POSTGRES_USER" "$POSTGRES_DB" "$TIMEOUT_SECONDS" "$LOG_DIR/postgres-ready.log"

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

# curl --fail writes nothing on an error status, so re-request without it: the
# response body is usually an RFC 9457 problem document naming the cause, and
# quoting it here saves a round trip through the logs.
health_response="$(cat "$LOG_DIR/api-health.json" 2>/dev/null || true)"
if [[ -z "$health_response" ]]; then
  health_response="$(curl --silent "$API_BASE_URL/api/health" 2>/dev/null || true)"
fi
printf '%s' "$health_response" | grep -Eq '"status"[[:space:]]*:[[:space:]]*"healthy"' \
  || die "GET $API_BASE_URL/api/health did not report a healthy API; it returned: ${health_response:-<no response>}"
printf '%s' "$health_response" | grep -Eq '"database"[[:space:]]*:[[:space:]]*"connected"' \
  || die "GET $API_BASE_URL/api/health did not report a connected database; it returned: $health_response"
echo "Backend health: $health_response"

echo "Starting frontend at $FRONTEND_URL..."
(cd "$ROOT_DIR/frontend" && exec bun run dev -- --host 127.0.0.1) >"$LOG_DIR/frontend.log" 2>&1 &
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
grep -Fq 'All systems operational' "$ROOT_DIR/frontend/src/features/health/HealthCheck.tsx" \
  || die "frontend health page is missing the expected operational status"

# The browser calls a relative /api because VITE_API_URL is empty, so the request
# the client actually makes is same-origin on the Vite port and reaches the API
# only through the dev proxy. Exercising that path here is what keeps the CSP's
# connect-src at 'self' honest.
echo "Checking the Vite API proxy..."
curl --fail --silent --show-error "$FRONTEND_URL/api/health" >"$LOG_DIR/proxied-api-health.json" 2>"$LOG_DIR/proxied-api-health.err" \
  || die "the Vite dev proxy did not forward $FRONTEND_URL/api/health to the API"
grep -Eq '"status"[[:space:]]*:[[:space:]]*"healthy"' "$LOG_DIR/proxied-api-health.json" \
  || die "GET $FRONTEND_URL/api/health did not report a healthy API"

# GET /api/health is deliberately unauthenticated, so everything above proves
# only that the process is up. Exercising one authenticated route is what proves
# the whole local identity chain — signing key, token subject, seeded
# organisation, bound membership — actually lines up. It is conditional because
# the smoke test must not mutate the developer's database by seeding.
authenticated_check="skipped: VITE_DEV_TOKEN is empty (run 'make login')"
if [[ -n "${VITE_DEV_TOKEN:-}" ]]; then
  echo "Checking an authenticated route..."
  curl --silent --show-error --max-time 10 \
    -H "Authorization: Bearer ${VITE_DEV_TOKEN}" \
    "$API_BASE_URL/api/me" >"$LOG_DIR/api-me.json" 2>"$LOG_DIR/api-me.err" \
    || die "GET $API_BASE_URL/api/me could not be reached"
  me_response="$(cat "$LOG_DIR/api-me.json")"
  case "$me_response" in
    *'"no-organization"'*)
      die "GET $API_BASE_URL/api/me returned no-organization; the dev token's subject is not bound. Run 'make seed'."
      ;;
    *'"invalid-token"'*)
      die "GET $API_BASE_URL/api/me rejected the dev token; it is expired or signed by another key. Run 'make login --force'."
      ;;
    *'"role"'*) ;;
    *)
      die "GET $API_BASE_URL/api/me did not return a principal; it returned: ${me_response:-<no response>}"
      ;;
  esac
  authenticated_check="$API_BASE_URL/api/me resolves the seeded principal: $me_response"
fi

echo
echo "Automated checks passed:"
echo "  - PostgreSQL is ready and migrations are applied"
echo "  - $API_BASE_URL/api/health reports healthy/connected without a token"
echo "  - $FRONTEND_URL/health is served by Vite"
echo "  - frontend health page contains 'All systems operational'"
echo "  - $FRONTEND_URL/api/health is proxied to the API, so connect-src 'self' suffices"
echo "  - $authenticated_check"
echo
echo "Manual browser check: open $FRONTEND_URL/health and confirm"
echo "  'All systems operational', 'Connected', and the current API version are visible."
echo "Adminer: http://localhost:${ADMINER_PORT:-8081} (server: postgres)"
