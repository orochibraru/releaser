#!/bin/sh
# Bump Cargo.toml, build (which syncs Cargo.lock), pack the binary.
set -eu
# ponytail: first `version =` line is [package]'s; use `cargo set-version` (cargo-edit) for workspaces.
sed -i.bak "1,/^version = /s/^version = .*/version = \"$1\"/" Cargo.toml && rm Cargo.toml.bak
cargo build --release
mkdir -p dist
tar -czf dist/hello.tar.gz -C target/release hello
