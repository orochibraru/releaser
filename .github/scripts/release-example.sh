#!/bin/sh
# Usage: release-example.sh <expected tag|none> <subject> [body]
# Commits, runs releaser like a user's workflow would, and checks which tag it pushed.
set -eu
# The developer's commit; the release commit uses releaser's own fallback identity.
commit() { git -c user.name=dev -c user.email=dev@example.com commit --allow-empty -q "$@"; }
if [ $# -gt 2 ]; then commit -m "$2" -m "$3"; else commit -m "$2"; fi

# Local remote and fake GitHub: nothing reaches the real repository.
GITHUB_API_URL=http://127.0.0.1:8080 GITHUB_REPOSITORY=example/hello GITHUB_REF_NAME=main GITHUB_TOKEN=fake \
  "$RELEASER" -prepare "$PREPARE" -artifacts "$ARTIFACTS" -commit "$COMMIT" $FLAGS

tag=$(git describe --tags --exact-match 2>/dev/null || echo none)
if [ "$tag" != "$1" ]; then
  echo "::error::expected $1, released $tag"
  exit 1
fi
[ "$tag" = none ] || git ls-remote --exit-code --tags origin "$tag" >/dev/null
