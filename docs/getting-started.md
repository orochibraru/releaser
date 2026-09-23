# Getting Started

Release on every push to `main`, straight from your commit messages. No config
file, no plugins.

## 1. Write conventional commits

```text
feat(ui): dark mode        → minor
fix: crash on empty input  → patch
feat!: drop Node 18        → major
chore: bump deps           → no release
```

See [Versioning](versioning.md) for the full rules and how to change them.

## 2. Add the workflow

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    branches: [main]

concurrency: release # one release at a time; queued pushes release after

jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v7
      - uses: orochibraru/releaser@v1
```

That's it. On the next push with a `feat` or `fix` commit you get:

- a `CHANGELOG.md` entry,
- the `version` in `package.json` bumped (if the file exists),
- a `chore(release): X.Y.Z [skip ci]` commit and a `vX.Y.Z` tag on `main`,
- a GitHub release with the same notes.

The first release is always `1.0.0`. Shallow clones are fetched in full
automatically, so `fetch-depth: 0` is optional.

`@v1` follows the latest 1.x. To pin, use a release commit's SHA
(`orochibraru/releaser@<sha> # vX.Y.Z`): `action.yml` at that commit pins the
binary's version and checksum too.

## 3. Protected branches

The default `github.token` can't push to a protected `main`: the push fails with
`GH013: Repository rule violations`. releaser pushes through `origin`, so check
out with a credential the ruleset lets through. A deploy key is the narrowest:

1. `ssh-keygen -t ed25519 -N "" -f release`. Add `release.pub` as a deploy key
   with write access, and `release` as the `RELEASE_DEPLOY_KEY` secret.
2. In the ruleset on `main`, add **Deploy keys** to the bypass list.
3. Check out with it. The action keeps `github.token` for the API.

```yaml
- uses: actions/checkout@v7
  with:
    ssh-key: ${{ secrets.RELEASE_DEPLOY_KEY }}
- uses: orochibraru/releaser@v1
```

A PAT or GitHub App token (`token:` on both steps) works too, but only if its
user or app is on the bypass list; admins aren't by default. It also reaches
further than one repo's key. Pushes over either one trigger workflows, unlike
`github.token`'s.

## 4. Try it locally

```bash
go install github.com/orochibraru/releaser/cmd/releaser@latest
releaser
```

Or download `releaser-<os>-<arch>` from the
[latest release](https://github.com/orochibraru/releaser/releases/latest).

Outside CI (`CI` unset) it's a dry run: it prints the next version and its notes
and touches nothing. Pass `-dry-run=false` to really release. Only `branch`
(`main`) releases; to preview from a feature branch, pass
`-branch "$(git branch --show-current)"`.

Next: [Inputs and flags](options.md).
