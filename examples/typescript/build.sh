#!/bin/sh
# releaser already bumped package.json: sync the lockfile, compile, pack.
set -eu
npm install --no-audit --no-fund
npm run build
npm pack --pack-destination dist
