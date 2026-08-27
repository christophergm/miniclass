#!/usr/bin/env bash
# Prepare a working local development environment from a fresh checkout.
#
# Idempotent by construction: it creates what is missing and reports what is
# stale, but never rewrites a file a developer may have edited. In particular an
# existing .env is left byte-identical.
#
# See docs/adr/0011-local-development-orchestration-and-environment-contract.md.
set -Eeuo pipefail

MINICLASS_SCRIPT="scripts/setup.sh"
ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib.sh
. "$ROOT_DIR/scripts/lib.sh"

cd "$ROOT_DIR"

EXAMPLE_FILE="$ROOT_DIR/.env.example"
ENV_FILE="$ROOT_DIR/.env"
# Key paths in .env are relative to backend/, because that is the working
# directory of every process that reads them. See ADR 0011.
KEY_PATH_BASE="$ROOT_DIR/backend"
TIMEOUT_SECONDS="${SETUP_TIMEOUT_SECONDS:-60}"
LOG_DIR="$(log_dir setup "${SETUP_LOG_DIR:-}")"

# resolve_key_path turns a key path as written in .env into an absolute path.
# Relative paths are resolved against KEY_PATH_BASE rather than this script's
# working directory, so that the one value in .env means the same thing here as
# it does to the backend process that consumes it.
#
# The containing directory is created so the result can be made physical. An
# unresolved ../ in a message about a key path is exactly the confusion the
# relative-path convention already risks.
resolve_key_path() {
  local value="$1"
  local directory

  case "$value" in
    "") return 0 ;;
    /*) ;;
    *) value="$KEY_PATH_BASE/$value" ;;
  esac

  directory="$(dirname -- "$value")"
  mkdir -p "$directory" || die "could not create $directory"
  printf '%s/%s\n' "$(cd -- "$directory" && pwd -P)" "$(basename -- "$value")"
}

# 1. Prerequisites.

log "Checking prerequisites..."
require_command docker openssl go bun awk
docker compose version >/dev/null 2>&1 || die "Docker Compose is required"
docker info >/dev/null 2>&1 || die "the Docker daemon is not running"

# 2. The environment file.

[[ -f "$EXAMPLE_FILE" ]] || die "missing $EXAMPLE_FILE"

example_violations="$(env_violations "$EXAMPLE_FILE")"
if [[ -n "$example_violations" ]]; then
  echo ".env.example violates the environment invariant, so it cannot be copied:" >&2
  echo "$example_violations" >&2
  die "no value may contain whitespace or '#'; move the value into .secrets/ and reference its path"
fi

if [[ -f "$ENV_FILE" ]]; then
  log "Keeping the existing .env unchanged."

  env_file_violations="$(env_violations "$ENV_FILE")"
  if [[ -n "$env_file_violations" ]]; then
    warn "the existing .env violates the environment invariant and cannot be sourced by a POSIX shell:"
    echo "$env_file_violations" >&2
    warn "move each offending value into a file under .secrets/ and reference its path, or delete .env and re-run this script"
  fi

  # A key added to .env.example after a developer copied it is silently absent
  # from their .env, where it reads as an empty value rather than an error.
  missing_keys="$(comm -23 \
    <(env_keys "$EXAMPLE_FILE" | sort) \
    <(env_keys "$ENV_FILE" | sort))"
  if [[ -n "$missing_keys" ]]; then
    warn "these keys are in .env.example but not in your .env:"
    while IFS= read -r key; do
      echo "  $key" >&2
    done <<<"$missing_keys"
    warn "add them by hand; this script will not edit an existing .env"
  fi
else
  log "Creating .env from .env.example..."
  cp "$EXAMPLE_FILE" "$ENV_FILE"
fi

load_env "$ENV_FILE"

POSTGRES_USER="${POSTGRES_USER:-miniclass}"
POSTGRES_DB="${POSTGRES_DB:-miniclass}"

# 3. Signing keys.
#
# The keys are files rather than inline PEM because a PEM header contains spaces,
# which no quoting style can make safe for Make, a POSIX shell and godotenv at
# once. .secrets/ is gitignored explicitly, not merely covered by *.pem.

public_key_path="$(resolve_key_path "${AUTH_LOCAL_PUBLIC_KEY_FILE:-}")"
private_key_path="$(resolve_key_path "${AUTH_LOCAL_PRIVATE_KEY_FILE:-}")"

if [[ -z "$public_key_path" || -z "$private_key_path" ]]; then
  warn "AUTH_LOCAL_PUBLIC_KEY_FILE or AUTH_LOCAL_PRIVATE_KEY_FILE is unset in .env; skipping key generation"
elif [[ -f "$public_key_path" && -f "$private_key_path" ]]; then
  log "Signing keys already present."
elif [[ -f "$public_key_path" || -f "$private_key_path" ]]; then
  die "only one of $private_key_path and $public_key_path exists; delete it and re-run so a matching pair is generated"
else
  log "Generating a local ES256 signing keypair..."
  key_directory="$(dirname -- "$private_key_path")"
  # Only tighten a directory that exists for the keys. A developer who points the
  # key path at the repository root should not have the repository chmod 700'd.
  if [[ "$key_directory" != "$ROOT_DIR" && "$key_directory" != "$KEY_PATH_BASE" ]]; then
    chmod 700 "$key_directory"
  fi
  openssl ecparam -name prime256v1 -genkey -noout -out "$private_key_path" \
    >"$LOG_DIR/openssl.log" 2>&1 || die "could not generate a private key (see $LOG_DIR/openssl.log)"
  chmod 600 "$private_key_path"
  openssl ec -in "$private_key_path" -pubout -out "$public_key_path" \
    >>"$LOG_DIR/openssl.log" 2>&1 || die "could not derive the public key (see $LOG_DIR/openssl.log)"
  chmod 644 "$public_key_path"
  log "Wrote $private_key_path and $public_key_path"
fi

# 4. Frontend dependencies.

log "Installing frontend dependencies..."
(cd "$ROOT_DIR/frontend" && bun install) >"$LOG_DIR/bun-install.log" 2>&1 \
  || die "bun install failed (see $LOG_DIR/bun-install.log)"

# 5. PostgreSQL.

log "Starting PostgreSQL..."
docker compose --env-file "$ENV_FILE" up -d postgres >"$LOG_DIR/compose-up.log" 2>&1 \
  || die "could not start PostgreSQL (see $LOG_DIR/compose-up.log)"

log "Waiting for PostgreSQL..."
wait_for_postgres "$ENV_FILE" "$POSTGRES_USER" "$POSTGRES_DB" "$TIMEOUT_SECONDS" "$LOG_DIR/postgres-ready.log"

# 6. Migrations.

log "Applying database migrations..."
(cd "$ROOT_DIR/backend" && go run ./cmd/migrate up) >"$LOG_DIR/migrations.log" 2>&1 \
  || die "database migrations failed (see $LOG_DIR/migrations.log)"

log ""
log "Setup complete. Logs: $LOG_DIR"
log ""
log "Seed a synthetic organisation and a login with 'make db-seed', then run the two"
log "development processes in separate terminals:"
log "  make dev-backend     (API on http://localhost:${PORT:-8080})"
log "  make dev-frontend    (app on http://localhost:${VITE_PORT:-5173})"
log ""
log "Verify the whole stack with 'make smoke'."
