#!/bin/zsh
# Invoke with: zsh scripts/reset-owner-password.zsh
# Disable tracing before reading secrets, even when invoked with zsh -x.
unsetopt XTRACE VERBOSE
setopt ERR_EXIT NO_UNSET PIPE_FAIL

if ! command -v node >/dev/null 2>&1; then
  print -u2 -- 'Node.js 18+ is required.'
  exit 1
fi

export OWNER_AUTH_USER_ID="${OWNER_AUTH_USER_ID:-65686ecf-1324-4d50-a875-877aa230bc83}"

if [[ -z "${SUPABASE_URL:-}" ]]; then
  printf 'Production Supabase project URL (https://PROJECT_REF.supabase.co): '
  IFS= read -r SUPABASE_URL
fi
export SUPABASE_URL

trap 'unset SUPABASE_SERVICE_ROLE_KEY NEW_OWNER_PASSWORD' EXIT

if [[ -z "${SUPABASE_SERVICE_ROLE_KEY:-}" ]]; then
  printf 'Supabase service-role key: '
  IFS= read -r -s SUPABASE_SERVICE_ROLE_KEY
  printf '\n'
fi
export SUPABASE_SERVICE_ROLE_KEY

if [[ -z "${NEW_OWNER_PASSWORD:-}" ]]; then
  printf 'New owner password: '
  IFS= read -r -s NEW_OWNER_PASSWORD
  printf '\n'
fi
export NEW_OWNER_PASSWORD

node "${0:A:h}/reset-owner-password.mjs"
