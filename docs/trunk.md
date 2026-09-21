# Canaries and Release PR

For trunk-based development: every push to `main` ships a canary, and a release
PR carries the next stable version. Merging the PR releases it.

```yaml
on:
  push:
    branches: [main]

jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
    steps:
      - uses: actions/checkout@v7
        with:
          ssh-key: ${{ secrets.RELEASE_DEPLOY_KEY }} # see "Checks on the PR"
      - id: release
        uses: orochibraru/releaser@v1
        with:
          prerelease: canary
          release-pr: true
```

The two inputs are independent; use either one alone. [Release flows](flows.md)
draws each setup.

## Canaries

With `prerelease: canary`, a push with release-worthy commits since the last
canary ships `X.Y.Z-canary.N`:

- `X.Y.Z` is the next stable version, from the commits since the last stable
  tag. `N` counts up per `X.Y.Z` and restarts when it moves (`1.3.1-canary.2`,
  then a `feat`, then `1.4.0-canary.1`).
- Only a tag on `HEAD` is pushed. No release commit: `CHANGELOG.md` and
  `package.json` stay untouched.
- The GitHub release is marked as a prerelease, so it never becomes "Latest".
  Its notes cover everything since the last stable release.
- `prepare`, artifacts and Docker run as usual with the canary version. The
  image is tagged `:X.Y.Z-canary.N` and `:canary`, never `:latest`.
- Outputs: `released=true`, `prerelease=true`.

Any id made of letters, digits and hyphens works (`beta`, `next`). Stable
versions still come from stable tags only: canary tags never count as the
previous release.

Without `release-pr`, cut stable releases with a run that doesn't set
`prerelease`, e.g. from `workflow_dispatch`.

## Release PR

With `release-pr: true`, releaser never commits to `main`. On each push with
something to release, it:

1. builds the release commit on top of `HEAD` (`CHANGELOG.md`, `package.json`,
   `prepare`, `commit` files), titled `chore(release): X.Y.Z`;
2. force-pushes it to the `releaser/release` branch;
3. opens a PR from that branch into `branch`, or updates the open one.

The PR is rebuilt on every push, so it never conflicts and always shows the
current version and notes.

Merge it any way (squash, merge commit, rebase). The push that lands it finds
the `chore(release): X.Y.Z` commit (with or without GitHub's `(#N)` squash
suffix) and releases it: tag `vX.Y.Z` on that commit, GitHub release with its
`CHANGELOG.md` entry as notes, `prepare` again (not committed) to build
artifacts, and Docker `:X.Y.Z` and `:latest`. That push makes no canary and no
new PR; commits merged in the same push wait for the next one.

The release commit has no `[skip ci]`, because merging it has to run the
workflow.

## Checks on the PR

Nothing done with `github.token` triggers workflows: a release PR opened with it
gets no `pull_request` run, so a ruleset that requires status checks blocks the
merge. Push the branch with a deploy key instead, and run your checks on pushes
to it:

1. Generate a key pair (`ssh-keygen -t ed25519 -N "" -f release`). Add
   `release.pub` as a deploy key with write access, and `release` as the
   `RELEASE_DEPLOY_KEY` secret.
2. Check out with `ssh-key: ${{ secrets.RELEASE_DEPLOY_KEY }}`, as above.
   releaser pushes through `origin`, so the branch and tags go over the key.
3. Trigger your CI on that branch:

   ```yaml
   on:
     push:
       branches: [main, releaser/release]
     pull_request:
   ```

Pushes over a deploy key trigger workflows, so each update of the PR runs CI on
its head commit, and those checks count for the PR. The PR itself is still
opened with `token` (`github.token` is fine).

A PAT or GitHub App token passed as `token` works too, but it's a credential
tied to a person or an app, with a wider reach than one repo's key.

## Major tag for actions

A GitHub Action that moves its major tag (`v1`) must skip canaries, and point it
at the release tag (after a merge commit, that's not `HEAD`):

```yaml
- if:
    steps.release.outputs.released == 'true' && steps.release.outputs.prerelease
    == 'false'
  env:
    TAG: ${{ steps.release.outputs.tag }}
  run: |
    git tag -f "${TAG%%.*}" "$TAG"
    git push -f origin "${TAG%%.*}"
```
