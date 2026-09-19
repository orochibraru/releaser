# releaser

semantic-release without the plugins, the config, or `node_modules`. One static
Go binary, zero dependencies.

On every push to `main`, it reads your
[Conventional Commits](https://www.conventionalcommits.org), picks the next
version, writes `CHANGELOG.md`, bumps `package.json`, commits, tags, and creates
the GitHub release. Two optional modes, usable together: **artifact** (attach
files to the release) and **docker** (build and push the image).

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
      - uses: actions/checkout@v7
      - uses: orochibraru/releaser@v1
        with:
          artifacts: dist/app.zip=app-${version}.zip
          docker: true
```

## Documentation

- [Getting started](docs/getting-started.md)
- [Inputs and flags](docs/options.md)
- [Migrating from semantic-release](docs/migrating.md)
- [Versioning](docs/versioning.md)
- [Artifact mode](docs/artifacts.md)
- [Docker mode](docs/docker.md)
- [Architecture](docs/architecture.md)

Runnable examples: [artifact mode](examples/artifact),
[docker mode](examples/docker).

## Development

```bash
prek install    # gofmt, go mod tidy, go vet, go test, prettier, markdownlint
go test ./...   # unit + integration, incl. the examples (Docker one needs Docker)
```
