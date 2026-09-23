# Release Flows

Two inputs pick how releases reach `main`. Neither is the default.

| Setup                   | Inputs                                   | Every push to `main`                 | Stable release                        |
| ----------------------- | ---------------------------------------- | ------------------------------------ | ------------------------------------- |
| Straight to main        | —                                        | commit + tag `vX.Y.Z`                | that push                             |
| Canary + release PR     | `prerelease: canary`, `release-pr: true` | tag `vX.Y.Z-canary.N`, update the PR | merging the PR                        |
| Canary, manual stable   | `prerelease: canary`                     | tag `vX.Y.Z-canary.N`                | a run without `prerelease` (dispatch) |
| Release PR, no canaries | `release-pr: true`                       | update the PR                        | merging the PR                        |

## Straight to main

Each push with release-worthy commits gets a release commit (`CHANGELOG.md`,
`package.json`, `commit` files) and a tag, pushed together. The commit carries
`[skip ci]`, so it doesn't trigger another run.

```mermaid
gitGraph TB:
  commit id: "feat: login"
  commit id: "chore(release): 1.0.0" tag: "v1.0.0"
  commit id: "fix: typo"
  commit id: "chore(release): 1.0.1" tag: "v1.0.1"
  commit id: "docs: readme"
  commit id: "feat: logout"
  commit id: "chore(release): 1.1.0" tag: "v1.1.0"
```

`docs: readme` alone releases nothing: `docs` doesn't bump by default (see
[rules](versioning.md#rules)).

## Canary + release PR

`main` never gets a commit from releaser. Each push tags a canary on `HEAD` and
rebuilds the release PR on top of it; merging the PR tags the stable version on
the release commit.

```mermaid
gitGraph TB:
  commit id: "feat: login" tag: "v1.1.0-canary.1"
  commit id: "fix: typo" tag: "v1.1.0-canary.2"
  commit id: "chore: tidy"
  branch releaser/release
  commit id: "chore(release): 1.1.0" tag: "v1.1.0"
  checkout main
  merge releaser/release id: "Merge PR #12"
  commit id: "feat: search" tag: "v1.2.0-canary.1"
```

- `chore: tidy` makes no canary: nothing release-worthy since `canary.2`. The PR
  is still rebuilt.
- The graph shows a merge commit. With a squash merge, `v1.1.0` goes on the
  squashed commit on `main` instead; with a rebase merge, on the replayed one.
- `releaser/release` is force-pushed on every push, so the PR always sits on top
  of `main`: it never conflicts and its notes are current.

```mermaid
sequenceDiagram
  actor Dev
  participant Main as main
  participant CI as releaser (CI)
  participant GH as GitHub
  Dev->>Main: merge feat: login
  Main->>CI: push
  CI->>GH: tag v1.1.0-canary.1, prerelease
  CI->>GH: force-push releaser/release, open PR "chore(release): 1.1.0"
  Dev->>Main: merge fix: typo
  Main->>CI: push
  CI->>GH: tag v1.1.0-canary.2, prerelease
  CI->>GH: force-push releaser/release, update the PR
  Dev->>Main: merge the release PR
  Main->>CI: push
  CI->>GH: tag v1.1.0 on the release commit, release (Latest)
```

Setup, token and check details: [Canaries and release PR](trunk.md).

## Canary, manual stable

Canaries as above, no PR. A run without `prerelease` (for example a
`workflow_dispatch` job with the same other inputs) releases straight to main.

```mermaid
gitGraph TB:
  commit id: "feat: login" tag: "v1.1.0-canary.1"
  commit id: "fix: typo" tag: "v1.1.0-canary.2"
  commit id: "chore(release): 1.1.0" tag: "v1.1.0"
  commit id: "feat: search" tag: "v1.2.0-canary.1"
```

That run pushes to `main`, so it needs the same branch protection bypass as
[straight to main](#straight-to-main).

## What one push does

```mermaid
flowchart TD
  push([push to branch]) --> merged{"release-pr, and a merged<br/>chore(release): X.Y.Z<br/>since the last tag?"}
  merged -- yes --> stable["tag vX.Y.Z on that commit<br/>GitHub release, Docker :latest"]
  merged -- no --> worthy{"release-worthy commits<br/>since the last tag?"}
  worthy -- no --> nothing([nothing to release])
  worthy -- yes --> flags{"prerelease or<br/>release-pr set?"}
  flags -- neither --> direct["commit CHANGELOG.md and package.json<br/>tag vX.Y.Z, push both atomically<br/>GitHub release, Docker :latest"]
  flags -- yes --> fresh{"prerelease, and release-worthy<br/>commits since the last canary?"}
  fresh -- yes --> canary["tag vX.Y.Z-canary.N on HEAD<br/>GitHub prerelease, Docker :canary"]
  fresh -- no --> pr
  canary --> pr{"release-pr?"}
  pr -- yes --> rebuild["rebuild chore(release): X.Y.Z<br/>force-push releaser/release<br/>open or update the PR"]
  pr -- no --> done([done])
  rebuild --> done
```

Every push comes last in its box: releases and canaries run `prepare`, artifacts
and Docker `:X.Y.Z` first, the PR runs `prepare` first. A failure leaves the
remote untouched. Docker `:latest` or `:canary` moves after the push.
