#!/bin/sh
# Stamp the version in and cross-compile one binary per platform.
set -eu
for platform in linux/amd64 linux/arm64 darwin/arm64 windows/amd64; do
  os=${platform%/*} arch=${platform#*/} suffix=
  [ "$os" = windows ] && suffix=.exe
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags "-s -w -X main.version=$1" \
    -o "dist/hello-$1-$os-$arch$suffix" .
done
