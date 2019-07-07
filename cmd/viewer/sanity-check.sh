#!/bin/sh

set -eux

base_dir=$(dirname "$(realpath "$0")")
workdir=$(mktemp -d)
forest_cmd="$workdir/forest"
viewer_cmd="$workdir/viewer"

env --chdir="$base_dir/../forest" go build -o "$forest_cmd"
env --chdir="$base_dir" go build -o "$viewer_cmd"

cd "$workdir"
identity=$("$forest_cmd" create identity)
community=$("$forest_cmd" create community --as "$identity")
replyA=$("$forest_cmd" create reply --as "$identity" --to "$community" --content "A root")
replyB=$("$forest_cmd" create reply --as "$identity" --to "$community" --content "B root")
replyA2=$("$forest_cmd" create reply --as "$identity" --to "$replyA" --content "A2")
replyA3=$("$forest_cmd" create reply --as "$identity" --to "$replyA2" --content "A3")
replyB2=$("$forest_cmd" create reply --as "$identity" --to "$replyB" --content "B2")
replyAA2=$("$forest_cmd" create reply --as "$identity" --to "$replyA" --content "AA2")
replyA4=$("$forest_cmd" create reply --as "$identity" --to "$replyA3" --content "A4")
replyBB2=$("$forest_cmd" create reply --as "$identity" --to "$replyB" --content "BB2")
replyB3=$("$forest_cmd" create reply --as "$identity" --to "$replyB2" --content "B3")

"$viewer_cmd"
