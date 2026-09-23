# Canaries and Release PR

For trunk-based development: every push to `main` ships a canary, and a release
PR carries the next stable version. Merging the PR releases it.

```yaml
on:
  push:
    branches: [main]

concurrency: release

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
  `package.json` stay untouched, and so does anything `prepare` writes.
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

### Pruning old canaries

A canary per push piles up GitHub prereleases. Delete the releases, **keep the
tags**: releaser numbers `N` and finds the last canary from them.

```yaml
- env:
    GH_TOKEN: ${{ github.token }}
  run: |
    gh release list --limit 100 --json tagName,isPrerelease \
      --jq '[.[] | select(.isPrerelease)] | .[5:] | .[].tagName' \
      | xargs -r -n1 gh release delete --yes
```

No `--cleanup-tag`: without the tags, the next push would ship a canary even
with nothing new, reusing a number already pushed to your registry.

## Release PR

With `release-pr: true`, releaser never commits to `main`. On each push with
something to release since the last stable tag, it:

1. builds the release commit on top of `HEAD`, titled `chore(release): X.Y.Z`:
   `CHANGELOG.md`, `package.json` and `commit` files, after `prepare` ran;
2. force-pushes it to the `releaser/release` branch;
3. opens a PR from that branch into `branch`, or updates the open one.

The PR is rebuilt on every such push, even one that makes no canary, so it never
conflicts and always shows the current version and notes. The release commit has
no `[skip ci]`, because merging it has to run the workflow.

### Merging it

Merge it any way: squash, merge commit or rebase. The push that lands it finds
the `chore(release): X.Y.Z` commit (with or without GitHub's `(#N)` squash
suffix) and:

- tags `vX.Y.Z` on that commit, which isn't `HEAD` after a merge commit;
- creates the GitHub release, with the commit's `CHANGELOG.md` entry as notes;
- runs `prepare` again (nothing committed) and uploads artifacts;
- pushes Docker `:X.Y.Z` and moves `:latest`.

That push makes no canary and no new PR. Commits merged in the same push wait
for the next one.

### Which commits count

Only the release PR's own commit: one on `branch`'s first-parent chain, or
behind a merge of this repo's `releaser/release`. Ignored, so a contributor
can't pick the version or the tagged tree:

- a `chore(release): X.Y.Z` commit brought in by another PR's merge commit;
- one whose version isn't above the last release.

A rebase merge puts every commit of a PR on `branch`, so don't rebase-merge a
contributor PR carrying such a commit.

## Checks on the PR

Nothing done with `github.token` triggers workflows: a release PR opened with it
gets no `pull_request` run, so a ruleset that requires status checks blocks the
merge. Push the branch with a deploy key instead:

1. Generate a key pair (`ssh-keygen -t ed25519 -N "" -f release`). Add
   `release.pub` as a deploy key with write access, and `release` as the
   `RELEASE_DEPLOY_KEY` secret. If rulesets guard tags or branches, add **Deploy
   keys** to their bypass list.
2. Check out with `ssh-key: ${{ secrets.RELEASE_DEPLOY_KEY }}`, as above.
   releaser pushes through `origin`, so the branch and tags go over the key.
3. Run your CI on pushes to the branch, not only on `pull_request`. releaser
   pushes the branch before it opens the PR, so the first release commit only
   gets a `push` run:

   ```yaml
   on:
     pull_request:
     push:
       branches: [releaser/release]
   ```

The PR itself is still opened with `token` (`github.token` is fine). A PAT or
GitHub App token passed as `token` and to `actions/checkout` works too, but it's
a credential tied to a person or an app, with a wider reach than one repo's key.

## Reading the phase in a dry run

A `dry-run: true` step with the same inputs tells the jobs after it what the
real run will do, e.g. to build and test the image under its final tag first:

| Push                                             | `version`        | `prerelease` |
| ------------------------------------------------ | ---------------- | ------------ |
| lands the release PR                             | `X.Y.Z`          | `false`      |
| has release-worthy commits since the last canary | `X.Y.Z-canary.N` | `true`       |
| has none since the last canary                   | `X.Y.Z` (the PR) | `false`      |
| has nothing since the last stable tag            | empty            | empty        |

Rows 1 and 3 look the same. Tell them apart by the subject of the pushed commit:

```yaml
- id: next
  uses: orochibraru/releaser@v1
  with:
    prerelease: canary
    release-pr: true
    dry-run: true
- id: mode
  env:
    PRERELEASE: ${{ steps.next.outputs.prerelease }}
  run: |
    if git log -1 --format=%s | grep -Eq '^chore\(release\): [0-9]+\.[0-9]+\.[0-9]+( \(#[0-9]+\))?$'; then
      echo mode=stable >> "$GITHUB_OUTPUT"
    elif [ "$PRERELEASE" = true ]; then
      echo mode=canary >> "$GITHUB_OUTPUT"
    else
      echo mode=none >> "$GITHUB_OUTPUT"
    fi
```

A later job then runs releaser for real, and can fail if its `version` output
differs from the dry run's.

### Shipping the last canary as stable

To promote the last canary's image to `:X.Y.Z` instead of rebuilding, every
commit in the release PR must have shipped a canary. With default
[rules](versioning.md#rules), a `chore` or `docs` push makes no canary but still
rebuilds the PR, which then carries code no canary was built from. Make every
type bump:

```yaml
rules: breaking=major,feat=patch,docs=patch,refactor=patch,style=patch,test=patch,build=patch,ci=patch,chore=patch
```

## Major tag for actions

A GitHub Action that moves its major tag (`v1`) must skip canaries, and point it
at the release tag (after a merge commit, that's not `HEAD`). Canary tags are
for binaries and images: a canary makes no commit, so an action pinning a binary
version in `action.yml` (as this one does) still pins the last stable one at a
canary tag.

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
