# Architecture

## Repository layout

```text
cmd/                    the CLI
├── main.go             flags → options
└── release.go          the release pipeline
internal/
├── conventional/       commit parsing (commit.go), bump rules (rules.go)
├── semver/             vX.Y.Z tags: latest, next, prerelease N, release PR commit
├── changelog/          release notes (notes.go), CHANGELOG.md (file.go)
├── git/                git CLI wrapper: log, tags, commit + tag + push, tag push,
│                       release PR branch push
├── github/             REST release + uploads (release.go), release PR
│                       (pull_request.go), outputs (output.go)
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

`release()` in `cmd/release.go` runs these steps in order (the default mode;
[canaries and the release PR](#canaries-and-release-pr) reuse them):

1. Skip unless on the release branch.
2. Unshallow if needed, find the latest `vX.Y.Z` tag, read commits since it.
3. Parse commits, compute the bump. No bump → stop.
4. Render the notes. Dry run → print, write the `version` and `tag` outputs, and
   stop.
5. Prepend `CHANGELOG.md`, bump `package.json`, run `prepare`.
6. Resolve artifacts, then build and push the Docker image.
7. Commit, tag, and `git push --atomic` the branch and tag together.
8. Create the GitHub release (a draft with `draft`) and upload artifacts.
9. Write the `released`, `version`, `tag`, `prerelease` outputs.

Everything that can fail on bad input or a broken build (steps 5–6) runs before
step 7, so a failed run leaves the remote untouched. After step 7 a GitHub API
failure leaves a pushed tag without a release; create it manually from the tag.

## Canaries and release PR

With `release-pr`, step 2 also looks for a merged `chore(release): X.Y.Z` commit
since the last tag. Found → the stable phase: steps 5–6 without the file
changes, then push only the tag, on that commit, and create the release from its
`CHANGELOG.md` entry. Nothing else runs on that push.

Otherwise, after step 4:

- `prerelease` (canary phase): skip if nothing release-worthy landed since the
  nearest `v*-<id>.*` tag; else steps 5–6 without the file changes, push only
  the tag on `HEAD`, create a prerelease.
- `release-pr` (PR phase): step 5, then `git commit-tree` on top of `HEAD` (the
  branch doesn't move), force-push it to `releaser/release`, open or update the
  PR.

Each phase has one push, after everything that can fail. [trunk.md](trunk.md)
has the user-facing side.

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
