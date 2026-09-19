# releaser

semantic-release without the plugins, the config, or `node_modules`. One static Go binary, zero dependencies.

Out of the box, on every push to `main`:

1. Reads [Conventional Commits](https://www.conventionalcommits.org) since the last `vX.Y.Z` tag.
2. Picks the bump: `breaking` → major, `feat` → minor, `fix`/`perf`/`revert` → patch.
3. Writes release notes (conventionalcommits style) to `CHANGELOG.md` and bumps `package.json` if there is one.
4. Runs your `prepare` command.
5. Commits `chore(release): X.Y.Z [skip ci]`, tags `vX.Y.Z`, and pushes both atomically.
6. Creates the GitHub release.

Two optional modes, which you can enable together:

- **artifact**: uploads files to the GitHub release (`artifacts`).
- **docker**: builds `./Dockerfile` and pushes `:X.Y.Z` and `:latest` to `ghcr.io/<owner>/<repo>` (`docker: true`).

Anything that can fail (prepare, artifact lookup, Docker push) runs before the git push, so a failed run leaves the remote untouched.

## GitHub Action

```yaml
on:
  push:
    branches: [main]

jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      packages: write # docker mode only
    steps:
      - uses: actions/checkout@v5
      - uses: orochibraru/releaser@v1
        with:
          artifacts: dist/app.zip=app-${version}.zip
          docker: true
```

This is the equivalent of a full `.releaserc.json` with 6 plugins, such as [bercail's](https://github.com/orochibraru/bercail/blob/main/.releaserc.json):

```yaml
      - uses: orochibraru/releaser@v1
        with:
          rules: breaking=patch,feat=patch,docs=patch,refactor=patch
          prepare: bun scripts/bump-version.ts ${version} && bun run prettier --write CHANGELOG.md && bun run package:extension
          artifacts: dist/bercail-extension.zip=bercail-extension-${version}.zip
```

| Input              | Default                  | Notes                                                    |
| ------------------ | ------------------------ | -------------------------------------------------------- |
| `branch`           | `main`                   | Other branches are a no-op.                              |
| `rules`            |                          | `type=none\|patch\|minor\|major`, `breaking` for `!`/footer. |
| `prepare`          |                          | Shell command, `${version}` replaced.                    |
| `commit`           |                          | Extra files for the release commit.                      |
| `artifacts`        |                          | `path[=name]`, comma/newline separated, globs ok.        |
| `docker`           | `false`                  | Multi-arch via `docker-platforms` + setup-qemu/buildx.   |
| `docker-image`     | `ghcr.io/<owner>/<repo>` | Other registries: log in first (docker/login-action).    |
| `dry-run`          | `false`                  |                                                          |
| `token`            | `github.token`           | Use a PAT if `main` is protected.                        |

Outputs: `released`, `version`, `tag`.

Shallow clones are fetched in full automatically, so `fetch-depth: 0` is optional.

## CLI

```sh
go install github.com/orochibraru/releaser/cmd@latest   # installs as "cmd"; rename or build with -o releaser
releaser                                # dry run outside CI: prints the next version and notes
releaser -dry-run=false                 # really release
releaser -h
```

## Development

```sh
prek install            # gofmt, go mod tidy, go vet, go test on commit
go test ./...           # tests/unit and tests/integration (real binary against a local bare remote)
```
