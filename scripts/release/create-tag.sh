#!/usr/bin/env sh
set -eu

version=${1:?usage: create-tag.sh <vX.Y.Z-rc.N|vX.Y.Z>}

if ! printf '%s\n' "$version" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+)?$'; then
    echo "release version must be vX.Y.Z-rc.N or vX.Y.Z: $version" >&2
    exit 1
fi

if [ "$(git branch --show-current)" != "main" ]; then
    echo "release tags must be created from main" >&2
    exit 1
fi

if [ -n "$(git status --porcelain)" ]; then
    echo "refusing to tag a dirty worktree" >&2
    exit 1
fi

git fetch --quiet origin main
if [ "$(git rev-parse HEAD)" != "$(git rev-parse origin/main)" ]; then
    echo "local main is not current with origin/main; run 'git pull --ff-only'" >&2
    exit 1
fi

if git rev-parse --verify --quiet "refs/tags/$version" >/dev/null; then
    echo "release tag already exists: $version" >&2
    exit 1
fi

git tag -a "$version" -m "Release $version"
git push origin "refs/tags/$version"

echo "Created and pushed annotated release tag $version."
echo "GitHub Actions validates CI and starts the matching staging or production release."
