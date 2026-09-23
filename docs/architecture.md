# Architecture

Only the Go standard library is used. `git` and, in Docker mode, `docker` must
be on `PATH`.

## The pipeline

`release()` in `cmd/releaser/release.go` runs these steps in order (the default
mode; [canaries and the release PR](#canaries-and-release-pr) reuse them):

1. Skip unless on the release branch.
2. Unshallow if needed, find the latest `vX.Y.Z` tag, read commits since it.
3. Parse commits, compute the bump. No bump → stop.
4. Render the notes. Dry run → print, write the `version` and `tag` outputs, and
   stop.
5. Prepend `CHANGELOG.md`, bump `package.json`, run `prepare`.
6. Resolve artifacts, then build and push the Docker image as `:X.Y.Z`.
7. Commit, tag, and `git push --atomic` the branch and tag together.
8. Create the GitHub release (a draft with `draft`) and upload artifacts.
9. Point the image's `:latest` (or prerelease id) at `:X.Y.Z`.
10. Write the `released`, `version`, `tag`, `prerelease` outputs.

Everything that can fail on bad input or a broken build (steps 5–6) runs before
step 7, so a failed run leaves the remote untouched (a rejected push leaves at
most an orphan `:X.Y.Z` image, which the next release of that version
overwrites). After step 7 a rerun finds nothing to release, so a failure in
steps 8–9 says what to finish by hand: `gh release create` from the tag, or the
`imagetools` command.

`prepare` runs with the environment minus `GITHUB_TOKEN` and `GH_TOKEN`: it runs
build tools and their dependencies, which have no business with the release
token.

## Canaries and release PR

With `release-pr`, step 2 also looks for a merged `chore(release): X.Y.Z` commit
since the last tag: on the branch's first-parent chain (squash, rebase), or as
the second parent of a `Merge pull request #N from <owner>/releaser/release`
commit, with `<owner>` the repo's. A release commit that came in with any other
PR, or whose version isn't above the last tag, is ignored: otherwise a
contributor could pick the version and the tagged tree. Found → the stable
phase: steps 5–6 without the file changes, then push only the tag, on that
commit, and create the release from its `CHANGELOG.md` entry. Nothing else runs
on that push.

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
`RELEASER_VERSION`, checks it against the `RELEASER_SHA256_<OS>_<ARCH>` pinned
next to it, and falls back to `go build` from the action's checkout when the
download fails or doesn't match. A release asset can be replaced after the fact;
`action.yml` at the ref a user pins can't. Inputs go through environment
variables, never through `${{ }}` inside the script, so they can't inject shell.

This repo releases itself with its own binary (`.github/workflows/ci.yml`):
`prepare` writes the new version into `action.yml`, cross-compiles
`dist/releaser-<os>-<arch>` with `-buildvcs=false` (the release PR and its merge
must build the same bytes) and writes their SHA-256 next to it, artifact mode
uploads them, and a last step moves the major tag (`v1`) that Marketplace users
pin.
