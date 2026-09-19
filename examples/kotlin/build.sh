#!/bin/sh
# Compile a self-contained jar, then stamp the version into its manifest.
set -eu
mkdir -p build dist
kotlinc hello.kt -include-runtime -d dist/hello.jar
printf 'Implementation-Version: %s\n' "$1" > build/MANIFEST.MF
jar --update --file dist/hello.jar --manifest build/MANIFEST.MF
