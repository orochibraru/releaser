# Architecture

## Repository layout

```text
cmd/                    the CLI
├── main.go             flags → options
└── release.go          the release pipeline
internal/
├── conventional/       commit parsing (commit.go), bump rules (rules.go)
├── semver/             vX.Y.Z tags: latest, next
├── changelog/          release notes (notes.go), CHANGELOG.md (file.go)
├── git/                git CLI wrapper: log, tags, commit + tag + push
├── github/             REST release + uploads (release.go), outputs (output.go)
├── docker/             buildx build --push
├── npm/                package.json version bump
└── artifacts/          path[=name] specs → files
examples/
├── artifact/           artifact mode: build.sh → release asset
└── docker/             docker mode: Dockerfile → image
tests/
├── unit/               one file per internal package
└── integration/        real binary against a local bare remote, a fake
                        GitHub API and, for docker, a local registry
action.yml              composite action: download binary, map inputs to flags
```

Only the Go standard library is used. `git` and, in Docker mode, `docker` must
be on `PATH`.

## The pipeline

`release()` in `cmd/release.go` runs these steps in order:

1. Skip unless on the release branch.
2. Unshallow if needed, find the latest `vX.Y.Z` tag, read commits since it.
3. Parse commits, compute the bump. No bump → stop.
4. Render the notes. Dry run → print and stop.
5. Prepend `CHANGELOG.md`, bump `package.json`, run `prepare`.
6. Resolve artifacts, then build and push the Docker image.
7. Commit, tag, and `git push --atomic` the branch and tag together.
8. Create the GitHub release and upload artifacts.
9. Write the `released`, `version`, `tag` outputs.

Everything that can fail on bad input or a broken build (steps 5–6) runs before
step 7, so a failed run leaves the remote untouched. After step 7 a GitHub API
failure leaves a pushed tag without a release; create it manually from the tag.

## The action

`action.yml` is a composite action. It downloads the prebuilt binary for the
runner's OS and architecture from this repo's release matching
`RELEASER_VERSION`, and falls back to `go build` from the action's checkout.
Inputs go through environment variables, never through `${{ }}` inside the
script, so they can't inject shell.

This repo releases itself with its own binary (`.github/workflows/ci.yml`):
`prepare` writes the new version into `action.yml` and cross-compiles
`dist/releaser-<os>-<arch>`, artifact mode uploads them, and a last step moves
the major tag (`v1`) that Marketplace users pin.
