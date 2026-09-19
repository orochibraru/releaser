#!/bin/sh
# Stand-in for a real build: stamp the version into a file and pack it.
set -eu
mkdir -p dist
echo "hello from $1" > dist/hello.txt
tar -czf dist/hello.tar.gz -C dist hello.txt
