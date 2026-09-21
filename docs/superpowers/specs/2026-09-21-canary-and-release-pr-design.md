# Canary prereleases and release PR

Trunk-based development: every push to `main` ships a canary, and a
release-please style PR carries the next stable release. Merging that PR tags
the stable version.

## Interface

Two new flags, orthogonal, each mapped 1:1 to an action input of the same name.
Without either one, behavior is unchanged (direct commit, tag, push).

| Flag / input        | Effect                                                          |
| ------------------- | --------------------------------------------------------------- |
| `-prerelease <id>`  | Push to the release branch ships `X.Y.Z-<id>.N`: tag only       |
| `-release-pr`       | Release commit goes to a PR; merging it cuts the stable tag     |
| output `prerelease` | `"true"` when the release was a prerelease, `"false"` otherwise |

- Canary without release PR: stable releases come from a run without
  `-prerelease` (e.g. `workflow_dispatch`).
- Release PR without canary: plain release-please.
- PR head branch is `releaser/release`, hardcoded. PR base is `-branch`.

## Flow

```text
push to -branch
 ├─ release commit in lastStable..HEAD?            (only with -release-pr)
 │    yes → prepare (not committed), resolve artifacts, docker :X.Y.Z + :latest,
 │          push tag vX.Y.Z on the release commit, GitHub release (not prerelease).
 │          Stop: no canary, no PR on this push.
 └─ no  → 1. canary (with -prerelease): prepare (not committed), artifacts,
             docker :X.Y.Z-<id>.N + :<id>, push tag on HEAD, GitHub release
             with prerelease: true.
          2. release PR (with -release-pr): build the release commit on a
             detached HEAD (CHANGELOG, package.json, prepare, -commit files),
             force-push it to releaser/release, create or update the PR
             (title "chore(release): X.Y.Z", body = notes).
          Neither flag → today's direct mode.
```

Each phase keeps the invariant: everything that can fail runs before its one
push (tag only for stable and canary, one branch for the PR). A failed phase
leaves the remote untouched.

## Rules

- **Stable base:** `semver.Latest` keeps ignoring prerelease tags. Notes and
  bump are always computed from the last stable tag.
- **Canary N:** highest merged `vX.Y.Z-<id>.N` for the computed base, plus 1;
  none → 1. The base can move between canaries (`1.3.1-canary.2`, then a `feat`,
  then `1.4.0-canary.1`).
- **Canary skip:** no release-worthy commit since the latest canary tag of any
  base → no canary.
- **Release commit detection:** `git log lastStable..HEAD` over all parents,
  subject matching `^chore\(release\): (\d+\.\d+\.\d+)( \(#\d+\))?$` (squash
  merges append a `(#N)` suffix; merge commits reach it through the second
  parent; rebase merges replay it). Several matches → the newest. The tag goes
  on the matched commit, not HEAD. Tag already exists → nothing to do.
- **No `[skip ci]` in PR mode.** It would stop the merge push from running
  releaser and the PR's own checks. The detection above is the loop guard.
  Direct mode keeps `[skip ci]`.
- **Prepare runs twice in PR mode:** when building the PR commit (output
  committed) and at stable tag time (to build artifacts; output discarded). It
  must be idempotent. The dogfood `prepare` already is.
- **Docker:** `docker.Push` takes its tag list. Stable: `:X.Y.Z`, `:latest`.
  Canary: `:X.Y.Z-<id>.N`, `:<id>`.
- **Dry run:** prints which phase would run and its version; writes nothing.

Deliberate cut: commits merged in the same push as the release PR wait for the
next push before they get a canary or show in the PR. Add a second pass if merge
queues make it matter.

## Tokens

PRs opened with `github.token` don't trigger `pull_request` workflows, so
required status checks never run on them. Recommended fix: check out with a
deploy key (`ssh-key`) and run CI on `push` to `releaser/release`; deploy key
pushes trigger workflows and their checks count on the PR head. A PAT or App
token also works. Documented, not worked around in code.

## Code

- `internal/semver`: `Version` stays `[3]int`; `v.Pre(id, tags)` returns
  `X.Y.Z-id.N`, `FromReleaseCommit(message)` the merged PR's version.
- `internal/git`: `ReleaseCommit(rng) (sha, version, found)`,
  `TagPush(tag, sha)`, `CommitPushBranch(files, message, branch)` (force push of
  a commit made on a detached HEAD, then back to the original ref).
- `internal/github`: `CreateRelease` takes `prerelease`. `UpsertPR` (list by
  `head=owner:releaser/release&state=open`, create or `PATCH`) over a generic
  `send` next to `post`. No `ClosePR`: in trunk history, release-worthy commits
  since the last tag never disappear, so an open PR never goes stale.
- `internal/docker`: `Push(image, platforms, tags, ...)`.
- `cmd/release.go`: split into the stable-from-PR, canary and PR phases, with
  direct mode as today. `cmd/main.go`, `action.yml`, `docs/options.md` get both
  flags and the output together.
- Docs: `docs/versioning.md` (prereleases no longer ignored in output, still in
  base), a trunk-based section with the `ci.yml` snippet, `docs/migrating.md`
  (release-please mapping), CLAUDE.md "Deferred" list.

## Dogfood

`ci.yml` `release` job: `-prerelease canary -release-pr`, `github.token`, pushes
over the existing `RELEASE_DEPLOY_KEY`; CI also runs on `push` to
`releaser/release`. The major tag step gets:

```yaml
if:
  steps.release.outputs.released == 'true' && steps.release.outputs.prerelease
  == 'false'
```

## Tests

- Unit: `semver` (prerelease parse/format, `NextPre`, `Latest` still ignores
  them), release commit subject matching (plain, `(#12)` suffix, lookalikes
  rejected).
- Fake GitHub (`fake_github_test.go`): PR list, create, patch; releases store
  `prerelease`. `tests/fakegithub` unchanged: no example uses PR mode.
- New `release_pr_steps_test.go` history: `feat` → `1.1.0-canary.1` + PR
  `1.1.0`; `fix` → `1.1.0-canary.2` + PR updated; merge the PR (squash and
  merge-commit variants) → `v1.1.0` stable, no canary, PR none open; `feat` →
  `1.2.0-canary.1` + new PR. Final `CHANGELOG.md` asserted in full (stable
  entries only).
- Failure: a failing `prepare` in each phase leaves the remote untouched.
- The existing direct-mode history in `release_steps_test.go` stays green,
  unchanged.
