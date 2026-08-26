#!/usr/bin/env bash
# Shared helpers for the repository's development scripts. Sourced, never run.
#
# The environment contract these helpers enforce is recorded in
# docs/adr/0011-local-development-orchestration-and-environment-contract.md.

# die prints a failure message and exits non-zero. Callers set MINICLASS_SCRIPT
# so the message names the script the developer actually invoked.
die() {
  echo "${MINICLASS_SCRIPT:-$(basename -- "$0")} failed: $*" >&2
  exit 1
}

# log prints a progress line. Progress goes to stdout so that a failure message
# on stderr is the only thing a caller redirecting stdout away will see.
log() {
  echo "$*"
}

# warn prints a non-fatal problem.
warn() {
  echo "warning: $*" >&2
}

# require_command fails unless every named executable is on PATH.
require_command() {
  local missing=()
  local name
  for name in "$@"; do
    command -v "$name" >/dev/null 2>&1 || missing+=("$name")
  done
  if [[ "${#missing[@]}" -ne 0 ]]; then
    die "missing required command(s): ${missing[*]}"
  fi
}

# log_dir creates and prints a timestamped log directory for one script run.
# The first argument names the script; the second, if given, overrides the
# location entirely.
log_dir() {
  local label="$1"
  local override="${2:-}"
  local directory

  if [[ -n "$override" ]]; then
    directory="$override"
  else
    directory="${TMPDIR:-/tmp}/miniclass-${label}-$(date +%Y%m%d-%H%M%S)"
  fi

  mkdir -p "$directory" || die "could not create log directory $directory"
  printf '%s\n' "$directory"
}

# env_violations prints one diagnostic per line in the given file that a POSIX
# shell, GNU Make and godotenv would not all agree about.
#
# The invariant is that no value may contain whitespace or '#'. A value with a
# space is a syntax error to a POSIX shell sourcing the file; quoting it to fix
# that makes the quotes part of the value for GNU Make's include. A '#' starts a
# comment for GNU Make but not for a shell. Only a value with neither is read
# identically by all three, so this check is what keeps the file loadable at all.
env_violations() {
  local file="$1"

  awk '
    /^[[:space:]]*$/ { next }
    /^[[:space:]]*#/ { next }
    {
      if ($0 !~ /^[A-Za-z_][A-Za-z0-9_]*=/) {
        printf "line %d: not a NAME=value assignment: %s\n", NR, $0
        next
      }
      equals = index($0, "=")
      value = substr($0, equals + 1)
      if (value ~ /[[:space:]]/) {
        printf "line %d: value of %s contains whitespace\n", NR, substr($0, 1, equals - 1)
      }
      if (value ~ /#/) {
        printf "line %d: value of %s contains #\n", NR, substr($0, 1, equals - 1)
      }
    }
  ' "$file"
}

# env_set writes NAME=value into the given file, replacing the first existing
# assignment in place and appending only when the name is absent. Every other
# line, including comments and ordering, is preserved byte for byte, because a
# developer's .env is a file they edit and this is a tool that edits it behind
# their back.
#
# The invariant is checked here rather than trusted: a value with whitespace or
# '#' would make the whole file unloadable for every consumer, and the write is
# the last moment at which that can be refused by name.
env_set() {
  local file="$1"
  local key="$2"
  local value="$3"
  local temporary

  [[ -f "$file" ]] || die "missing $file; run ./scripts/setup.sh first"
  case "$value" in
    *[[:space:]]* | *"#"*)
      die "refusing to write $key: the value contains whitespace or '#', which no .env parser reads identically"
      ;;
  esac

  temporary="$(mktemp "${TMPDIR:-/tmp}/miniclass-env.XXXXXX")" || die "could not create a temporary file"
  awk -v key="$key" -v value="$value" '
    BEGIN { written = 0 }
    index($0, key "=") == 1 {
      if (!written) { print key "=" value; written = 1 }
      next
    }
    { print }
    END { if (!written) print key "=" value }
  ' "$file" >"$temporary" || die "could not rewrite $key in $file"

  # Copy over the original rather than moving the temporary onto it, so the
  # file keeps its inode and permissions instead of inheriting mktemp's 0600.
  cat "$temporary" >"$file" || die "could not write $file"
  rm -f "$temporary"
}

# jwt_expiry prints the exp claim of a JWT as a Unix timestamp, and fails for
# anything it cannot read. No signature is checked: the caller is deciding
# whether to re-mint a token it is about to hand to a verifier that will check
# the signature properly.
jwt_expiry() {
  local token="$1"
  local payload
  local decoded
  local expiry

  [[ "$(printf '%s' "$token" | awk -F. '{ print NF }')" == "3" ]] || return 1
  payload="$(printf '%s' "$token" | cut -d. -f2 | tr '_-' '/+')"
  [[ -n "$payload" ]] || return 1
  while [[ $(( ${#payload} % 4 )) -ne 0 ]]; do
    payload="${payload}="
  done

  decoded="$(printf '%s' "$payload" | openssl base64 -d -A 2>/dev/null)" || return 1
  expiry="$(printf '%s' "$decoded" | sed -n 's/.*"exp"[[:space:]]*:[[:space:]]*\([0-9][0-9]*\).*/\1/p')"
  [[ -n "$expiry" ]] || return 1
  printf '%s\n' "$expiry"
}

# dev_admin_subject prints the local provider subject for DEV_ADMIN_EMAIL.
#
# One derivation, in one place. The seed's invitation email, the token's email
# claim and the token's subject all have to agree or the claim is refused and
# the login silently does not exist; deriving the subject from the address
# rather than configuring it separately is what makes disagreement impossible.
dev_admin_subject() {
  local email="${1:-${DEV_ADMIN_EMAIL:-}}"

  [[ -n "$email" ]] || die "DEV_ADMIN_EMAIL is not set; add it to .env (see .env.example)"
  printf 'local:%s\n' "$email"
}

# env_keys prints the variable names assigned in the given file, in order.
env_keys() {
  local file="$1"

  awk '
    /^[[:space:]]*$/ { next }
    /^[[:space:]]*#/ { next }
    /^[A-Za-z_][A-Za-z0-9_]*=/ { print substr($0, 1, index($0, "=") - 1) }
  ' "$file"
}

# load_env exports every assignment in the given file into the environment.
#
# The file is checked before it is sourced. Sourcing a file that violates the
# invariant produces errors like "sh: EC: command not found" from a PEM header,
# which under `set -e` kills the caller with no indication of the real cause.
load_env() {
  local file="$1"
  local violations

  [[ -f "$file" ]] || die "missing $file; run ./scripts/setup.sh first"

  violations="$(env_violations "$file")"
  if [[ -n "$violations" ]]; then
    echo "$file cannot be loaded; no value may contain whitespace or '#':" >&2
    echo "$violations" >&2
    echo >&2
    echo "Move multi-line or spaced values, such as a PEM key, into a file under" >&2
    echo ".secrets/ and reference the path instead. See docs/adr/0011-local-development-orchestration-and-environment-contract.md." >&2
    exit 1
  fi

  set -a
  # shellcheck disable=SC1090
  . "$file"
  set +a
}

# port_in_use reports whether anything is listening on a local TCP port. bash's
# /dev/tcp is used rather than lsof or nc so that no extra prerequisite appears.
#
# The probe runs in a subshell, whose exit both closes the descriptor and
# supplies this function's status. Do not "tidy up" afterwards with
# `exec 3>&- 2>/dev/null`: an exec with redirections and no command applies them
# to the calling shell permanently, so that line sent stderr to /dev/null for
# the remainder of every script that called this function, and warnings and
# `set -x` traces vanished for no visible reason.
port_in_use() {
  local port="$1"

  (exec 3<>"/dev/tcp/127.0.0.1/$port") >/dev/null 2>&1
}

# require_free_port fails unless the given port is free.
#
# Checked up front because the alternative is a service that exits immediately
# with "address already in use" and then a full health-check timeout, whose
# failure message names the health check rather than the port.
require_free_port() {
  local port="$1"
  local what="$2"

  if port_in_use "$port"; then
    die "port $port is already in use, so the $what cannot start; stop whatever is listening on it and re-run"
  fi
}

# stop_process_tree terminates a background job and everything it spawned.
#
# Signalling the job alone is not enough: `go run` compiles to a temporary binary
# and runs it as a child, and `bun run dev` reaches vite through two more
# processes. Killing only the job leaves the real server holding its port, and
# the next run fails with "address already in use" for no visible reason. The
# caller enables job control so each background job is its own process group,
# which is what makes the whole tree addressable as a negative pid.
stop_process_tree() {
  local pid="$1"

  [[ -n "$pid" ]] || return 0

  kill -TERM "-$pid" 2>/dev/null || kill -TERM "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
  kill -KILL "-$pid" 2>/dev/null || true
}

# wait_for_postgres blocks until the compose postgres service accepts
# connections, or fails after the given number of seconds.
wait_for_postgres() {
  local env_file="$1"
  local user="$2"
  local database="$3"
  local timeout="$4"
  local log_file="$5"
  local _

  for _ in $(seq 1 "$timeout"); do
    if docker compose --env-file "$env_file" exec -T postgres \
      pg_isready -U "$user" -d "$database" >"$log_file" 2>&1; then
      return 0
    fi
    sleep 1
  done

  docker compose --env-file "$env_file" exec -T postgres \
    pg_isready -U "$user" -d "$database" >"$log_file" 2>&1 \
    || die "PostgreSQL did not become ready within ${timeout}s (see $log_file)"
}
