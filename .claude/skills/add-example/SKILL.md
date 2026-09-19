---
name: add-example
description:
  Add a runnable release example for a new language or toolchain under
  examples/, wired into the Examples workflow and examples/README.md. Use when
  asked for a new example, language, or build tool.
---

# Add an example

An example is a minimal project that releases with artifact mode. Copy the shape
of an existing one (`examples/rust` for a manifest with a lockfile, `examples/c`
for a version stamped at build time).

1. `examples/<name>/`: smallest real source that prints `hello from <version>`,
   plus:
   - `build.sh` (executable, `#!/bin/sh`, `set -eu`, one-line header comment).
     It takes the version as `$1`, bumps the manifest if the language has one
     (`sed -i.bak ... && rm *.bak`, portable across GNU and BSD), builds, and
     leaves the artifact in `dist/`. releaser bumps `package.json` itself.
   - `.gitignore` for every build output (`dist/`, `target/`, `build/`, ...).
     The workflow fails on a dirty tree.
   - Any lockfile, generated locally and committed.
2. Add a matrix entry to `.github/workflows/examples.yml`: `artifacts`, `commit`
   (manifest and lockfile), `asset` (uploaded name with `VERSION`), `manifest`
   (file that must end at `2.0.0`). If the toolchain isn't preinstalled on
   `ubuntu-latest` (check actions/runner-images `Ubuntu2404-Readme.md`), add a
   setup step with `if: matrix.example == '<name>'` at the action's latest tag
   (`git ls-remote --tags --refs`).
3. Add the same row to the table in `examples/README.md`.
4. Verify locally by running the job's steps by hand: build `./cmd` and
   `./tests/fakegithub`, copy the folder to a fresh repo with a bare remote, and
   run `.github/scripts/release-example.sh` for each commit with `RELEASER`,
   `PREPARE`, `ARTIFACTS`, `COMMIT` and `FLAGS` set. Lint with
   `go run github.com/rhysd/actionlint/cmd/actionlint@latest`. If the tool isn't
   installed locally, run `build.sh` once in a container and say the full run
   only happens in CI.
