#!/bin/sh
# releaser already bumped package.json: just pack it.
set -eu
mkdir -p dist
npm pack --pack-destination dist
