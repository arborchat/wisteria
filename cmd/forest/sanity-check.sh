#!/bin/sh

set -eux

forest_cmd=$(realpath ./forest)
workdir=$(mktemp -d)

cd "$workdir"
identity=$("$forest_cmd" create identity)
community=$("$forest_cmd" create community --as "$identity")
reply1=$("$forest_cmd" create reply --as "$identity" --to "$community" --content test1)
reply2=$("$forest_cmd" create reply --as "$identity" --to "$reply1" --content test2)

"$forest_cmd" show "$identity"
"$forest_cmd" show "$community"
"$forest_cmd" show "$reply1"
"$forest_cmd" show "$reply2"
