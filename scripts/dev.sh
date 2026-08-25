#!/bin/sh
set -eu

# Dev orchestration script for MinisClass local development.
# - Starts Postgres via docker compose (compose.yaml in repo root)
# - Generates a local ES256 keypair for the backend verifier if missing
# - Writes a minimal repo-level .env and frontend/.env if they don't exist
# - Runs backend and frontend in the background, streaming logs to .tmp/

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd -P)
cd "$ROOT_DIR"

TMPDIR="$ROOT_DIR/.tmp"
mkdir -p "$TMPDIR"

# Default config values (safe for local dev only)
DB_URL="postgres://miniclass:miniclass_dev_password@localhost:5432/miniclass?sslmode=disable"
API_PORT="8080"
AUTH_ISSUER="http://localhost:8080"
AUTH_AUDIENCE="authenticated"
KEY_ID="local"

# 1) Generate local keypair if missing
PRIVPATH="$ROOT_DIR/backend/scripts/dev_private.pem"
PUBPATH="$ROOT_DIR/backend/scripts/dev_public.pem"
if [ ! -f "$PRIVPATH" ] || [ ! -f "$PUBPATH" ]; then
  echo "Generating local ES256 keypair..."
  mkdir -p "$(dirname "$PRIVPATH")"
  # Generate a P-256 private key and public key
  if command -v openssl >/dev/null 2>&1; then
    openssl ecparam -name prime256v1 -genkey -noout -out "$PRIVPATH"
    openssl ec -in "$PRIVPATH" -pubout -out "$PUBPATH"
    echo "Generated: $PRIVPATH, $PUBPATH"
  else
    echo "openssl is required to generate a keypair. Please install openssl and re-run." >&2
    exit 1
  fi
fi

# 2) Ensure repo-level .env (backend config). Do not overwrite if it exists.
if [ ! -f "$ROOT_DIR/.env" ]; then
  echo "Writing $ROOT_DIR/.env (local development)."
  # Escape newlines in PEMs into literal \n so the backend ParsePrivateKeyPEM handles them.
  PRIV_ESCAPED=$(awk '{printf "%s\\n", $0}' "$PRIVPATH")
  PUB_ESCAPED=$(awk '{printf "%s\\n", $0}' "$PUBPATH")

  cat > "$ROOT_DIR/.env" <<EOF
DATABASE_URL=$DB_URL
PORT=$API_PORT
AUTH_PROVIDER=local
AUTH_ISSUER=$AUTH_ISSUER
AUTH_AUDIENCE=$AUTH_AUDIENCE
AUTH_LOCAL_PRIVATE_KEY=$PRIV_ESCAPED
AUTH_LOCAL_PUBLIC_KEY=$PUB_ESCAPED
AUTH_LOCAL_KEY_ID=$KEY_ID
EOF
  echo "Wrote $ROOT_DIR/.env"
else
  echo "$ROOT_DIR/.env already exists, leaving it alone."
fi

# 3) Mint a local dev token using the backend devtoken tool
DEV_TOKEN=""
if command -v go >/dev/null 2>&1; then
  echo "Minting a local dev token..."
  # Run the devtoken command from the backend module directory so `go run` finds its go.mod.
  if DEV_TOKEN=$(cd "$ROOT_DIR/backend" && go run ./cmd/devtoken --email dev@example.com --subject local:dev --private-key-file "$PRIVPATH" --issuer "$AUTH_ISSUER" --audience "$AUTH_AUDIENCE" --key-id "$KEY_ID"); then
    echo "Dev token minted."
  else
    echo "Failed to mint dev token; continuing without VITE_DEV_TOKEN."
    DEV_TOKEN=""
  fi
else
  echo "go is not available; skipping dev token mint."
fi

# 4) Ensure frontend .env with minimal values so the app can initialize in dev
FRONTEND_ENV="$ROOT_DIR/frontend/.env"
if [ ! -f "$FRONTEND_ENV" ]; then
  echo "Writing $FRONTEND_ENV (frontend dev env)."
  cat > "$FRONTEND_ENV" <<EOF
VITE_SUPABASE_URL=$AUTH_ISSUER
VITE_SUPABASE_ANON_KEY=localdevkey
VITE_API_URL=http://localhost:8080
VITE_DEV_TOKEN=$DEV_TOKEN
EOF
else
  echo "$FRONTEND_ENV already exists, leaving it alone. To inject a dev token, add VITE_DEV_TOKEN=<token> to it."
fi

# 4) Start Postgres (docker compose)
if command -v docker >/dev/null 2>&1 && command -v docker-compose >/dev/null 2>&1; then
  # some systems name the binary docker-compose; recent docker uses `docker compose` subcommand.
  echo "Using docker-compose binary to start Postgres..."
  docker-compose -f "$ROOT_DIR/compose.yaml" up -d
elif command -v docker >/dev/null 2>&1; then
  echo "Using 'docker compose' to start Postgres..."
  docker compose -f "$ROOT_DIR/compose.yaml" up -d
else
  echo "docker is required to start Postgres (compose). Please install docker and re-run." >&2
  exit 1
fi

# Give Postgres a few seconds to start
echo "Waiting for Postgres to start..."
sleep 5

# 5) Run migrations and seed (best-effort)
echo "Running backend migrations (best-effort)..."
if make -C "$ROOT_DIR/backend" migrate-up >/dev/null 2>&1; then
  echo "Migrations applied."
else
  echo "migrate-up failed or returned non-zero; continuing anyway."
fi

if make -C "$ROOT_DIR/backend" seed >/dev/null 2>&1; then
  echo "Seed applied."
else
  echo "Seed failed or returned non-zero; continuing anyway."
fi

# # 6) Start backend and frontend, stream logs
# BACKEND_LOG="$TMPDIR/backend.log"
# FRONTEND_LOG="$TMPDIR/frontend.log"

# echo "Starting backend (logs -> $BACKEND_LOG)..."
# (
#   cd "$ROOT_DIR/backend"
#   if command -v air >/dev/null 2>&1; then
#     air -c .air.toml >>"$BACKEND_LOG" 2>&1 &
#   else
#     # fallback to go run (hot reload won't be available)
#     (go run ./cmd/api) >>"$BACKEND_LOG" 2>&1 &
#   fi
#   echo $! > "$TMPDIR/backend.pid"
# )

# # Give the backend a short moment to bind its port
# sleep 1

# echo "Starting frontend (logs -> $FRONTEND_LOG)..."
# (
#   cd "$ROOT_DIR/frontend"
#   if command -v bun >/dev/null 2>&1; then
#     bun install >>"$FRONTEND_LOG" 2>&1 || true
#     bun run dev >>"$FRONTEND_LOG" 2>&1 &
#   else
#     npm install >>"$FRONTEND_LOG" 2>&1 || true
#     npm run dev >>"$FRONTEND_LOG" 2>&1 &
#   fi
#   echo $! > "$TMPDIR/frontend.pid"
# )

# # 7) Tail logs until interrupted
# echo "Tailing backend and frontend logs. Press Ctrl-C to stop."

# trap 'echo "Stopping dev environment..."; if [ -f "$TMPDIR/frontend.pid" ]; then kill "$(cat "$TMPDIR/frontend.pid")" >/dev/null 2>&1 || true; fi; if [ -f "$TMPDIR/backend.pid" ]; then kill "$(cat "$TMPDIR/backend.pid")" >/dev/null 2>&1 || true; fi; exit 0' INT TERM

# # Print last 200 lines and then follow
# if command -v tail >/dev/null 2>&1; then
#   tail -n 200 -F "$BACKEND_LOG" "$FRONTEND_LOG"
# else
#   # Simple fallback: print files and sleep in a loop
#   while :; do
#     clear
#     echo "--- backend ---"
#     cat "$BACKEND_LOG" || true
#     echo "\n--- frontend ---"
#     cat "$FRONTEND_LOG" || true
#     sleep 2
#   done
# fi
