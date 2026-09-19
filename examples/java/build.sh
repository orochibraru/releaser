#!/bin/sh
# Compile, then stamp the version into the jar manifest.
set -eu
javac -d build src/hello/Hello.java
printf 'Implementation-Version: %s\n' "$1" > build/MANIFEST.MF
mkdir -p dist
jar --create --file dist/hello.jar --manifest build/MANIFEST.MF --main-class hello.Hello -C build hello
