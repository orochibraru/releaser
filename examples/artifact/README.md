# Artifact mode example

`build.sh` stands in for a real build: it stamps the version into a file and
packs it as `dist/hello.tar.gz`. Each release attaches it as
`hello-X.Y.Z.tar.gz`, and bumps `package.json`.

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
        with:
          prepare: ./build.sh ${version}
          artifacts: dist/hello.tar.gz=hello-${version}.tar.gz
```

[`tests/integration/artifact_example_test.go`](../../tests/integration/artifact_example_test.go)
releases this folder with the same settings, against a local git remote and a
fake GitHub API. It checks the four GitHub releases, the uploaded
`hello-X.Y.Z.tar.gz` for each, and `package.json` ending at `2.0.0`.

## What the test proves

Both examples are released through the same commit history
([`release_steps_test.go`](../../tests/integration/release_steps_test.go)):

| Commit                        | Release    | Rule             |
| ----------------------------- | ---------- | ---------------- |
| `feat: hello`                 | `v1.0.0`   | first release    |
| `fix(build): smaller archive` | `v1.0.1`   | `fix` → patch    |
| `feat: greet by name`         | `v1.1.0`   | `feat` → minor   |
| `chore: tidy`                 | no release | `chore` → none   |
| `feat!: new greeting format`  | `v2.0.0`   | breaking → major |

After the run, `CHANGELOG.md` must hold the four releases newest first, with
compare links, a `⚠ BREAKING CHANGES` section for `2.0.0`, the `build` scope,
nothing from the `chore`, and a single trailing newline.

See [Artifact mode](../../docs/artifacts.md).
