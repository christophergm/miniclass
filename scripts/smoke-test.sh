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
env_backup=""
env_modified=0
seed_output=""
claim_url="${SMOKE_CLAIM_URL:-}"

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

  if [[ "$env_modified" -eq 1 && -n "$env_backup" && -f "$env_backup" ]]; then
    cp "$env_backup" "$ENV_FILE" || true
  fi

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

OWNER_EMAIL="${SMOKE_OWNER_EMAIL:-${DEV_ADMIN_EMAIL:-owner@example.test}}"

POSTGRES_USER="${POSTGRES_USER:-miniclass}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-miniclass_dev_password}"
POSTGRES_DB="${POSTGRES_DB:-miniclass}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
PORT="${PORT:-8080}"
VITE_PORT="${VITE_PORT:-5173}"
DATABASE_URL="${DATABASE_URL:-postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable}"
[[ -n "${APP_DATABASE_URL:-}" ]] || die "APP_DATABASE_URL is required; see .env.example"
API_BASE_URL="${API_BASE_URL:-http://localhost:${PORT}}"
API_BASE_URL="${API_BASE_URL%/}"
API_BASE_URL="${API_BASE_URL%/api}"
FRONTEND_URL="http://localhost:${VITE_PORT}"

export DATABASE_URL APP_DATABASE_URL PORT API_BASE_URL
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

# Mint an unresolved local identity before starting Vite. The seed below binds
# this identity through the same claim endpoint a browser uses. Keeping the
# token in .env only for this process is necessary because frontend/package.json
# sources that file before Vite starts; cleanup restores the developer's file.
echo "Minting a temporary local identity for the claim check..."
env_backup="$LOG_DIR/env.backup"
cp "$ENV_FILE" "$env_backup" || die "could not back up $ENV_FILE"
smoke_token="$(cd "$ROOT_DIR/backend" && go run ./cmd/devtoken \
  -subject "$(dev_admin_subject "$OWNER_EMAIL")" \
  -email "$OWNER_EMAIL" \
  -lifetime 1h)" || die "could not mint the temporary claim-check token"
[[ -n "$smoke_token" ]] || die "the temporary claim-check token was empty"
env_modified=1
env_set "$ENV_FILE" VITE_DEV_TOKEN "$smoke_token"
export VITE_DEV_TOKEN="$smoke_token"

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

if [[ -z "$claim_url" ]]; then
  existing_me="$(curl --silent --show-error --max-time 10 \
    -H "Authorization: Bearer $smoke_token" \
    "$API_BASE_URL/api/me" 2>/dev/null || true)"
  if printf '%s' "$existing_me" | grep -Eq '"role"[[:space:]]*:[[:space:]]*"'; then
    # A normal developer run has already used make db-seed to bind the Owner.
    # Invite a synthetic administrator so the smoke test remains repeatable
    # without trying to create a second organization for the same subject.
    claim_email="smoke-admin-$(date +%s)-$$@example.test"
    echo "Creating a synthetic administrator invitation for the claim check..."
    invitation_response="$(curl --fail --silent --show-error --max-time 10 \
      -H "Authorization: Bearer $smoke_token" \
      -H 'Content-Type: application/json' \
      -d "{\"email\":\"$claim_email\",\"role\":\"administrator\"}" \
      "$API_BASE_URL/api/administrators")" \
      || die "could not create the smoke-test administrator invitation"
    claim_url="$(printf '%s\n' "$invitation_response" | sed -n 's/.*"claim_url":"\([^"]*\)".*/\1/p')"
    claim_bearer="$(cd "$ROOT_DIR/backend" && go run ./cmd/devtoken \
      -subject "$(dev_admin_subject "$claim_email")" \
      -email "$claim_email" \
      -lifetime 1h)" || die "could not mint the synthetic administrator claim token"
  else
    echo "Seeding an unclaimed Owner invitation..."
    seed_output="$(cd "$ROOT_DIR" && make db-seed SEED_OWNER_SUBJECT= 2>&1 | tee "$LOG_DIR/seed.log")" \
      || die "could not create the smoke-test seed organization"
    claim_url="$(printf '%s\n' "$seed_output" | sed -n 's/^Owner invitation claim URL: //p')"
  fi
fi
[[ -n "$claim_url" ]] || die "the seed output did not contain an Owner invitation claim URL"

claim_bearer="${claim_bearer:-$smoke_token}"
claim_email="${claim_email:-$OWNER_EMAIL}"

claim_token="$(printf '%s\n' "$claim_url" | sed -n 's/.*[?&]token=\([^&]*\).*/\1/p')"
[[ -n "$claim_token" ]] || die "the claim URL has no token query parameter: $claim_url"

# Vite's history fallback serves the application shell for the generated URL;
# fetching it and its source modules proves the printed URL reaches the actual
# claim route and that the route still reads the query parameter. This catches
# the original backend/frontend path-vs-query regression before the API claim.
echo "Following the invitation claim URL..."
curl --fail --silent --show-error "$claim_url" >"$LOG_DIR/claim-page.html" 2>"$LOG_DIR/claim-page.err" \
  || die "frontend did not serve the invitation claim URL $claim_url"
grep -Fq '/src/main.tsx' "$LOG_DIR/claim-page.html" \
  || die "the invitation claim URL did not return the frontend application shell"
curl --fail --silent --show-error "$FRONTEND_URL/src/App.tsx" >"$LOG_DIR/app-source.tsx" 2>"$LOG_DIR/app-source.err" \
  || die "could not inspect the frontend route source"
grep -Fq '/claim"' "$LOG_DIR/app-source.tsx" \
  || die "the frontend claim route no longer accepts the backend's /claim?token= URL"
curl --fail --silent --show-error "$FRONTEND_URL/src/features/auth/ClaimInvitationPage.tsx" >"$LOG_DIR/claim-source.tsx" 2>"$LOG_DIR/claim-source.err" \
  || die "could not inspect the frontend claim page source"
grep -Fq 'useSearchParams' "$LOG_DIR/claim-source.tsx" \
  || die "the frontend claim page no longer reads the invitation query parameter"

echo "Completing the invitation claim through the Vite proxy..."
curl --fail --silent --show-error --max-time 10 \
  -H "Authorization: Bearer $claim_bearer" \
  -H 'Content-Type: application/json' \
  -d "{\"token\":\"$claim_token\"}" \
  "$FRONTEND_URL/api/auth/claim" >"$LOG_DIR/claim-response.json" 2>"$LOG_DIR/claim-response.err" \
  || die "the generated invitation claim could not be completed"

echo "Checking the claimed identity and seeded school year through the Vite proxy..."
curl --fail --silent --show-error --max-time 10 \
  -H "Authorization: Bearer $claim_bearer" \
  "$FRONTEND_URL/api/me" >"$LOG_DIR/api-me.json" 2>"$LOG_DIR/api-me.err" \
  || die "GET $FRONTEND_URL/api/me failed after claiming the invitation"
me_response="$(cat "$LOG_DIR/api-me.json")"
printf '%s' "$me_response" | grep -Eq '"email"[[:space:]]*:[[:space:]]*"'"$claim_email"'"' \
  || die "GET $FRONTEND_URL/api/me did not return the claimed email: $me_response"
printf '%s' "$me_response" | grep -Eq '"role"[[:space:]]*:[[:space:]]*"(owner|administrator)"' \
  || die "GET $FRONTEND_URL/api/me did not return an administrator membership: $me_response"

echo "Enrolling and verifying MFA for the smoke-test administrator..."
mfa_enrollment_response="$(curl --fail --silent --show-error --max-time 10 \
  -H "Authorization: Bearer $claim_bearer" \
  -H 'Content-Type: application/json' \
  -d '{}' \
  "$FRONTEND_URL/api/auth/mfa/enroll")" \
  || die "POST $FRONTEND_URL/api/auth/mfa/enroll failed for the smoke-test administrator"
mfa_recovery_code="$(printf '%s' "$mfa_enrollment_response" | sed -n 's/.*"recovery_codes"[[:space:]]*:[[:space:]]*\["\([^"]*\)".*/\1/p')"
[[ -n "$mfa_recovery_code" ]] || die "MFA enrollment did not return a recovery code"
mfa_session_response="$(curl --fail --silent --show-error --max-time 10 \
  -H "Authorization: Bearer $claim_bearer" \
  -H 'Content-Type: application/json' \
  -d "{\"recovery_code\":\"$mfa_recovery_code\"}" \
  "$FRONTEND_URL/api/auth/mfa/verify")" \
  || die "POST $FRONTEND_URL/api/auth/mfa/verify failed for the smoke-test administrator"
admin_bearer="$(printf '%s' "$mfa_session_response" | sed -n 's/.*"session_token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
[[ -n "$admin_bearer" ]] || die "MFA verification did not return an administrative session"

curl --fail --silent --show-error --max-time 10 \
  -H "Authorization: Bearer $admin_bearer" \
  "$FRONTEND_URL/api/school-years" >"$LOG_DIR/school-years.json" 2>"$LOG_DIR/school-years.err" \
  || die "GET $FRONTEND_URL/api/school-years failed after claiming the invitation"
school_years_response="$(cat "$LOG_DIR/school-years.json")"
printf '%s' "$school_years_response" | grep -Eq '"label"[[:space:]]*:[[:space:]]*"Synthetic School Year"' \
  || die "GET $FRONTEND_URL/api/school-years did not return the seeded year: $school_years_response"

# The browser calls a relative /api because VITE_API_URL is empty, so the
# request the client actually makes is same-origin on the Vite port and reaches
# the API only through this proxy.
echo "Checking the Vite API proxy..."
curl --fail --silent --show-error "$FRONTEND_URL/api/health" >"$LOG_DIR/proxied-api-health.json" 2>"$LOG_DIR/proxied-api-health.err" \
  || die "the Vite dev proxy did not forward $FRONTEND_URL/api/health to the API"
grep -Eq '"status"[[:space:]]*:[[:space:]]*"healthy"' "$LOG_DIR/proxied-api-health.json" \
  || die "GET $FRONTEND_URL/api/health did not report a healthy API"

echo
echo "Automated checks passed:"
echo "  - PostgreSQL is ready and migrations are applied"
echo "  - $API_BASE_URL/api/health reports healthy/connected without a token"
echo "  - $FRONTEND_URL/health is served by Vite"
echo "  - frontend health page contains 'All systems operational'"
echo "  - $FRONTEND_URL/api/health is proxied to the API, so connect-src 'self' suffices"
echo "  - $claim_url loads the frontend claim route"
echo "  - the invitation claim completes through $FRONTEND_URL/api/auth/claim"
echo "  - $FRONTEND_URL/api/me resolves the claimed administrator membership"
echo "  - $FRONTEND_URL/api/school-years returns Synthetic School Year"
echo
echo "Manual browser check: open $FRONTEND_URL/health and confirm"
echo "  'All systems operational', 'Connected', and the current API version are visible."
echo "Adminer: http://localhost:${ADMINER_PORT:-8081} (server: postgres)"
