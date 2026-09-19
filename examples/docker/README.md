# Docker mode example

The `Dockerfile` stands in for a real image: `FROM scratch` plus one file, so it
builds in a second with nothing to pull. Each release pushes
`ghcr.io/<owner>/<repo>:X.Y.Z` and `:latest`.

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
      packages: write
    steps:
      - uses: actions/checkout@v7
      - uses: orochibraru/releaser@v1
        with:
          docker: true
```

[`tests/integration/docker_example_test.go`](../../tests/integration/docker_example_test.go)
releases this folder into a throwaway local registry (`registry:3`) and checks
it ends up with `1.0.0`, `1.0.1`, `1.1.0`, `2.0.0` and `latest`. It needs Docker
with buildx and is skipped in `-short` mode.

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

See [Docker mode](../../docs/docker.md).
