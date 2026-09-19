#!/bin/sh
# releaser already bumped package.json: compile a standalone binary (version inlined).
set -eu
bun build hello.ts --compile --outfile dist/hello
