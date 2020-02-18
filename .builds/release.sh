#!/bin/sh

set -x
basedir=$(dirname "$(realpath "$0")")

# determine current commit
GIT_COMMIT="$(git rev-parse HEAD)"

# shellcheck source=.builds/lib.sh
. "$basedir/lib.sh"

if ! commit_on_branch "$GIT_COMMIT" "master"; then
  GORELEASER_FLAGS="--snapshot --skip-publish"
fi

# we want word splitting here
# shellcheck disable=SC2086
~/go/bin/goreleaser $GORELEASER_FLAGS --rm-dist

# sort the hashes of the built binaries in a reliable (if derpy) way
find dist -executable -type f -exec md5sum '{}' \; | rev | sort | rev
