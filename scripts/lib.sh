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
port_in_use() {
  local port="$1"

  (exec 3<>"/dev/tcp/127.0.0.1/$port") >/dev/null 2>&1 || return 1
  exec 3>&- 2>/dev/null || true
  return 0
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
