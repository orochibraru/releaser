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

## 3. Protected branches

The default `github.token` can't push to a protected `main`. Check out with a
token that can, and pass it to the action:

```yaml
- uses: actions/checkout@v7
  with:
    token: ${{ secrets.RELEASE_TOKEN }}
- uses: orochibraru/releaser@v1
  with:
    token: ${{ secrets.RELEASE_TOKEN }}
```

## 4. Try it locally

```bash
go install github.com/orochibraru/releaser/cmd@latest
cmd            # the binary is named after its folder; rename it to releaser
```

Outside CI (`CI` unset) it's a dry run: it prints the next version and its notes
and touches nothing. Pass `-dry-run=false` to really release.

Next: [Inputs and flags](options.md).
