#!/bin/sh

set -x
basedir=$(dirname "$(realpath "$0")")

# determine current commit
GIT_COMMIT="$(git rev-parse HEAD)"

# shellcheck source=.builds/lib.sh
. "$basedir/lib.sh"

# if we're not on master or master isn't tagged
if ! commit_on_branch "$GIT_COMMIT" "master" || ! git describe --tags --exact-match HEAD; then
  GORELEASER_FLAGS="--snapshot --skip-publish"
fi

if command -v goreleaser ; then
    goreleaser_path=$(command -v goreleaser)
elif find "$HOME/go/bin" -executable -type f -name goreleaser; then
    goreleaser_path="$HOME/go/bin/goreleaser"
fi

# we want word splitting here
# shellcheck disable=SC2086
"$goreleaser_path" $GORELEASER_FLAGS --rm-dist

# sort the hashes of the built binaries in a reliable (if derpy) way
find dist -executable -type f -exec md5sum '{}' \; | rev | sort | rev
