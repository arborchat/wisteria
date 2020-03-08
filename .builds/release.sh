#!/bin/sh

set -x
basedir=$(dirname "$(realpath "$0")")

# shellcheck source=.builds/lib.sh
. "$basedir/lib.sh"

# if we're not on master or master isn't tagged
PUBLISH_RELEASE=0
if git describe --tags --exact-match HEAD; then
  PUBLISH_RELEASE=1
fi

if command -v goreleaser ; then
    goreleaser_path=$(command -v goreleaser)
elif find "$HOME/go/bin" -executable -type f -name goreleaser; then
    goreleaser_path="$HOME/go/bin/goreleaser"
fi

if [ "$PUBLISH_RELEASE" -eq 0 ]; then
  # Build but do not publish
  GORELEASER_FLAGS="--snapshot --skip-publish"
fi

# we want word splitting here
# shellcheck disable=SC2086
"$goreleaser_path" $GORELEASER_FLAGS --rm-dist

# sort the hashes of the built binaries in a reliable (if derpy) way
find dist -executable -type f -exec sha256sum '{}' \; | rev | sort | rev

if [ "$PUBLISH_RELEASE" -eq 1 ]; then
    # erase the non-tarred directories from disk
    find dist -type d -exec rm -rf '{}' \;

    tag=$(git describe --exact-match HEAD)

    # ensure that we don't leak this srht token
    set +x
    for artifact in dist/* ; do
        echo curl -H "Authorization: token <token>" -F "file=@$artifact" "https://git.sr.ht/api/repos/wisteria/artifacts/$tag"
        curl -H "Authorization: token $SRHT_TOKEN" -F "file=@$artifact" "https://git.sr.ht/api/repos/wisteria/artifacts/$tag"
    done
    set -x
fi
