#!/usr/bin/env bash
# Put a usable local login in .env: mint a development bearer token for the
# DEV_ADMIN_EMAIL identity, or keep the existing one when it is still fresh.
#
# Idempotent, and safe to make a prerequisite of anything: it re-mints only when
# VITE_DEV_TOKEN is absent, unreadable, or close enough to expiry that a working
# session would end mid-task, and it says which of those happened.
#
# See docs/adr/0011-local-development-orchestration-and-environment-contract.md.
set -Eeuo pipefail

MINICLASS_SCRIPT="scripts/login.sh"
ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib.sh
. "$ROOT_DIR/scripts/lib.sh"

cd "$ROOT_DIR"

ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
# Thirty days. A local token is signed by a key on this machine and is refused
# outright in production by auth.ErrLocalProduction, so its lifetime is not a
# security boundary; cmd/devtoken's five-minute default is right for a test and
# wrong for a person, who otherwise re-mints several times an hour and learns to
# distrust every authentication failure.
LIFETIME="${DEV_TOKEN_LIFETIME:-720h}"
# Re-mint a day out rather than at expiry, so a token never dies mid-session.
MINIMUM_REMAINING_SECONDS="${DEV_TOKEN_MINIMUM_REMAINING_SECONDS:-86400}"
FORCE="${DEV_TOKEN_FORCE:-0}"

usage() {
  cat <<'USAGE'
Usage: ./scripts/login.sh [--force]

Refreshes VITE_DEV_TOKEN in .env when it is missing, unreadable, or expiring
within 24 hours. --force always mints a new token.

Environment overrides:
  DEV_TOKEN_LIFETIME                    token lifetime passed to cmd/devtoken (default 720h)
  DEV_TOKEN_MINIMUM_REMAINING_SECONDS   re-mint threshold (default 86400)
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --force) FORCE=1 ;;
    -h | --help)
      usage
      exit 0
      ;;
    *) die "unknown argument: $1" ;;
  esac
  shift
done

require_command go openssl awk curl

load_env "$ENV_FILE"

SUBJECT="$(dev_admin_subject)"
EMAIL="$DEV_ADMIN_EMAIL"

if [[ -z "${AUTH_LOCAL_PRIVATE_KEY_FILE:-}" && -z "${AUTH_LOCAL_PRIVATE_KEY:-}" ]]; then
  die "no local signing key is configured; set AUTH_LOCAL_PRIVATE_KEY_FILE in .env and run ./scripts/setup.sh"
fi

# 1. Decide whether the token in .env is still worth keeping.

now="$(date +%s)"
reason=""
existing="${VITE_DEV_TOKEN:-}"

if [[ "$FORCE" == "1" ]]; then
  reason="--force was given"
elif [[ -z "$existing" ]]; then
  reason="VITE_DEV_TOKEN is empty"
elif ! expiry="$(jwt_expiry "$existing")"; then
  reason="VITE_DEV_TOKEN is not a readable JWT"
elif [[ "$expiry" -le "$now" ]]; then
  reason="VITE_DEV_TOKEN expired $(( (now - expiry) / 3600 ))h ago"
elif [[ $(( expiry - now )) -lt "$MINIMUM_REMAINING_SECONDS" ]]; then
  reason="VITE_DEV_TOKEN expires in $(( (expiry - now) / 3600 ))h"
fi

if [[ -z "$reason" ]]; then
  log "Keeping the existing dev token for $SUBJECT; it expires in $(( (expiry - now) / 86400 ))d."
else
  log "Minting a dev token for $SUBJECT ($reason)."
  token="$(cd "$ROOT_DIR/backend" && go run ./cmd/devtoken \
    -subject "$SUBJECT" \
    -email "$EMAIL" \
    -lifetime "$LIFETIME")" || die "could not mint a development token"
  [[ -n "$token" ]] || die "cmd/devtoken produced no token"

  env_set "$ENV_FILE" VITE_DEV_TOKEN "$token"
  export VITE_DEV_TOKEN="$token"

  if minted_expiry="$(jwt_expiry "$token")"; then
    log "Wrote VITE_DEV_TOKEN to .env; it expires in $(( (minted_expiry - now) / 86400 ))d."
  else
    log "Wrote VITE_DEV_TOKEN to .env."
  fi
  log "Restart the Vite dev server to pick it up: VITE_* values are inlined when it starts."
fi

# 2. If the API happens to be running, say whether the token actually works.
#
# A token is only half of a login: it names a subject, and that subject reaches
# an organisation only once the seed has bound it. Checking here is what turns
# "403 no-organization" from a mystery into a named next command.

PORT="${PORT:-8080}"
API_BASE_URL="${API_BASE_URL:-http://localhost:$PORT}"
API_BASE_URL="${API_BASE_URL%/}"
API_BASE_URL="${API_BASE_URL%/api}"

if ! port_in_use "$PORT"; then
  log ""
  log "The API is not running, so the token was not exercised. Next:"
  log "  make dev-backend     API on $API_BASE_URL"
  log "  make dev-frontend    app on http://localhost:${VITE_PORT:-5173}"
  log ""
  log "If the API then answers 403 no-organization, run 'make db-seed' to bind $SUBJECT."
  exit 0
fi

response="$(curl --silent --show-error --max-time 10 \
  -H "Authorization: Bearer ${VITE_DEV_TOKEN}" \
  "$API_BASE_URL/api/me" 2>/dev/null || true)"

case "$response" in
  *'"no-organization"'*)
    log ""
    warn "the token verifies, but $SUBJECT has no organization membership."
    warn "run 'make db-seed' to create the synthetic organisation and bind it."
    exit 1
    ;;
  *'"multiple-organizations"'*)
    log ""
    warn "$SUBJECT holds more than one membership, which resolves to no tenant at all."
    warn "run 'make db-reset CONFIRM=1' to start from one organisation."
    exit 1
    ;;
  *'"role"'*)
    log ""
    log "GET $API_BASE_URL/api/me: $response"
    ;;
  "")
    log ""
    warn "GET $API_BASE_URL/api/me returned nothing; is the API still starting?"
    ;;
  *)
    log ""
    warn "GET $API_BASE_URL/api/me did not return a principal: $response"
    warn "if the database was just reset, the running API is pooling connections to the schema that was replaced; restart it."
    exit 1
    ;;
esac
