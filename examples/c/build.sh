#!/bin/sh
# Compile with the version baked in, pack the binary.
set -eu
mkdir -p dist
${CC:-cc} -O2 -DVERSION="\"$1\"" -o dist/hello hello.c
tar -czf dist/hello.tar.gz -C dist hello
