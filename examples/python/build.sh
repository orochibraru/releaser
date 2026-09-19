#!/bin/sh
# Bump pyproject.toml, then build the wheel.
set -eu
sed -i.bak "s/^version = .*/version = \"$1\"/" pyproject.toml && rm pyproject.toml.bak
python3 -m pip wheel --no-deps --wheel-dir dist .
