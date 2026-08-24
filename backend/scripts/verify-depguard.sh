#!/usr/bin/env sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
backend_root="$repository_root/backend"
temp_root=${TMPDIR:-${TMP:-${TEMP:-/tmp}}}
fixture_root=$(mktemp -d "$temp_root/miniclass-depguard.XXXXXX")
cleanup() {
    rm -rf "$fixture_root"
}
trap cleanup EXIT

module_root="$fixture_root/backend"
mkdir -p "$module_root/internal/db/gen" "$module_root/internal/data/identity" \
    "$module_root/internal/data" "$module_root/internal/identity" "$module_root/internal/illegal"

echo 'module github.com/chrismott/miniclass' > "$module_root/go.mod"
echo 'package gen; var Value = 1' > "$module_root/internal/db/gen/gen.go"
echo 'package identity; var Value = 1' > "$module_root/internal/data/identity/identity.go"
cp "$backend_root/internal/testing/depguardfixtures/allowed_generated.go.txt" \
    "$module_root/internal/data/allowed_generated.go"
cp "$backend_root/internal/testing/depguardfixtures/allowed_identity.go.txt" \
    "$module_root/internal/identity/allowed_identity.go"

run_lint() {
    golangci-lint run --config "$backend_root/.golangci.yml" --disable-all --enable depguard ./... 2>&1
}

cp "$backend_root/internal/testing/depguardfixtures/generated_violation.go.txt" \
    "$module_root/internal/illegal/generated_violation.go"
if output=$(cd "$module_root" && run_lint); then
    echo "depguard accepted a generated-code violation"
    exit 1
fi
echo "$output" | grep -F 'generated sqlc access belongs only in internal/data' >/dev/null
rm -f "$module_root/internal/illegal/generated_violation.go"

cp "$backend_root/internal/testing/depguardfixtures/identity_violation.go.txt" \
    "$module_root/internal/illegal/identity_violation.go"
if output=$(cd "$module_root" && run_lint); then
    echo "depguard accepted an identity-accessor violation"
    exit 1
fi
echo "$output" | grep -F 'the unscoped identity accessor belongs only in internal/identity' >/dev/null

echo "depguard import-boundary proofs passed"
