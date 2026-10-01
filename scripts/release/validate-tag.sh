#!/usr/bin/env sh
set -eu

release_tag=${1:?usage: validate-tag.sh <tag>}

if [ "$(git cat-file -t "refs/tags/$release_tag")" != "tag" ]; then
    echo "release tags must be annotated: $release_tag" >&2
    exit 1
fi

release_sha=$(git rev-parse "refs/tags/$release_tag^{commit}")
git fetch --no-tags origin main
if ! git merge-base --is-ancestor "$release_sha" origin/main; then
    echo "release tag $release_tag ($release_sha) is not reachable from main" >&2
    exit 1
fi

if printf '%s\n' "$release_tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+-rc\.[0-9]+$'; then
    environment=staging
elif printf '%s\n' "$release_tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
    environment=production
else
    echo "release tag must be vX.Y.Z-rc.N or vX.Y.Z: $release_tag" >&2
    exit 1
fi

: "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required to verify CI checks}"
checks=$(gh api "/repos/$GITHUB_REPOSITORY/commits/$release_sha/check-runs?per_page=100")
required_checks='Backend tests
Solver contract tests
Solver image
Deployment image smoke tests
Backend lint
Backend format
Generated code drift
Migration round-trip
Frontend tests
Frontend build
Frontend lint
Repository formatting
Developer tooling'

printf '%s\n' "$required_checks" | while IFS= read -r required_check; do
    conclusion=$(printf '%s' "$checks" | jq -r --arg name "$required_check" '
        [.check_runs[] | select(.name == $name) | .conclusion]
        | if index("success") then "success" else "missing or unsuccessful" end
    ')
    if [ "$conclusion" != "success" ]; then
        echo "CI check is $conclusion for $release_sha: $required_check" >&2
        exit 1
    fi
done

if [ -n "${GITHUB_OUTPUT:-}" ]; then
    {
        printf 'environment=%s\n' "$environment"
        printf 'sha=%s\n' "$release_sha"
        printf 'tag=%s\n' "$release_tag"
    } >> "$GITHUB_OUTPUT"
else
    printf 'environment=%s\nsha=%s\ntag=%s\n' "$environment" "$release_sha" "$release_tag"
fi
