#!/bin/bash

set -euo pipefail

# go to the directory in which this script resides
basedir=$(dirname "$(realpath "$0")")
cd "$basedir"

# make sure we can start wisteria and choose all of the default options
# without crashing
echo -e "0\n0\n0\n" | go run . \
    -grove /tmp/this/should/not/exist \
    -config /tmp/here/is/another/path/that/should/not/exist \
    -test-startup
