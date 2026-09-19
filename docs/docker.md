# Docker Mode

Build `./Dockerfile` and push it tagged with the release version.

```yaml
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

This runs `docker buildx build --push` on the repository root and pushes two
tags:

- `ghcr.io/<owner>/<repo>:X.Y.Z`
- `ghcr.io/<owner>/<repo>:latest`

The image gets the `org.opencontainers.image.version` and
`org.opencontainers.image.source` labels, so GHCR links the package to the repo.

The push happens **before** the release commit and tag, so a failed build leaves
the repository untouched.

## Registry and login

For `ghcr.io` images, the action logs in with its token as `GITHUB_ACTOR`. For
any other registry, set `docker-image` and log in first:

```yaml
- uses: docker/login-action@v4
  with:
    username: ${{ vars.DOCKERHUB_USERNAME }}
    password: ${{ secrets.DOCKERHUB_TOKEN }}
- uses: orochibraru/releaser@v1
  with:
    docker: true
    docker-image: docker.io/me/app
```

## Multi-arch

Multi-platform builds need QEMU and a buildx builder:

```yaml
- uses: docker/setup-qemu-action@v4
- uses: docker/setup-buildx-action@v4
- uses: orochibraru/releaser@v1
  with:
    docker: true
    docker-platforms: linux/amd64,linux/arm64
```

Docker mode and [artifact mode](artifacts.md) can be enabled together.
