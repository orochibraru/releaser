# Inputs and Flags

Every action input maps to a CLI flag of the same name.

```yaml
- uses: orochibraru/releaser@v1
  with:
    branch: main
    rules: ""
    prepare: ""
    commit: ""
    artifacts: ""
    docker: false
    docker-image: ""
    docker-platforms: ""
    dry-run: false
    token: ${{ github.token }}
```

| Input / flag       | Default                  | Description                                                                                                                 |
| ------------------ | ------------------------ | --------------------------------------------------------------------------------------------------------------------------- |
| `branch`           | `main`                   | Branch to release from. Any other branch is a no-op.                                                                        |
| `rules`            | —                        | Bump overrides, e.g. `breaking=patch,feat=patch`. See [Versioning](versioning.md#rules).                                    |
| `prepare`          | —                        | Shell command run before the release commit. `${version}` is replaced with the new version, e.g. `1.2.3`.                   |
| `commit`           | —                        | Extra files for the release commit, comma separated. `CHANGELOG.md` and `package.json` (when present) are always committed. |
| `artifacts`        | —                        | Enables [artifact mode](artifacts.md). `path[=name]`, comma or newline separated.                                           |
| `docker`           | `false`                  | Enables [Docker mode](docker.md).                                                                                           |
| `docker-image`     | `ghcr.io/<owner>/<repo>` | Image to push.                                                                                                              |
| `docker-platforms` | —                        | e.g. `linux/amd64,linux/arm64`.                                                                                             |
| `dry-run`          | `false` in CI            | Only print the next version and notes. The CLI defaults to `true` when `CI` is unset.                                       |
| `token`            | `github.token`           | Action only; the CLI reads `GITHUB_TOKEN` or `GH_TOKEN`. Used for the release, uploads and the ghcr.io login.               |

## Outputs

| Output     | Example  | Description                    |
| ---------- | -------- | ------------------------------ |
| `released` | `true`   | `false` when nothing shipped.  |
| `version`  | `1.2.3`  | Only set when `released=true`. |
| `tag`      | `v1.2.3` | Only set when `released=true`. |

```yaml
- id: release
  uses: orochibraru/releaser@v1
- if: steps.release.outputs.released == 'true'
  run: echo "shipped ${{ steps.release.outputs.tag }}"
```

## Environment

The CLI reads the usual Actions variables, so it behaves the same inside the
action or as a plain `run:` step:

| Variable                              | Used for                                       |
| ------------------------------------- | ---------------------------------------------- |
| `CI`                                  | Unset → dry run by default.                    |
| `GITHUB_TOKEN` / `GH_TOKEN`           | GitHub release, asset uploads, ghcr.io login.  |
| `GITHUB_REF_NAME`                     | Current branch (falls back to `git`).          |
| `GITHUB_REPOSITORY`                   | `owner/repo` (falls back to the `origin` URL). |
| `GITHUB_SERVER_URL`, `GITHUB_API_URL` | GitHub Enterprise Server.                      |
| `GITHUB_ACTOR`                        | ghcr.io login username.                        |
| `GITHUB_OUTPUT`                       | Where the outputs above are written.           |
