# Examples

Each folder is a minimal project that releases with artifact mode. `build.sh` is
the whole recipe: bump the manifest (if the language has one), build, and leave
the artifact in `dist/`. `package.json` is bumped by releaser itself.

```yaml
- uses: orochibraru/releaser@v1
  with:
    prepare: ./build.sh ${version}
    artifacts: <artifacts>
    commit: <commit>
```

| Example                  | Toolchain setup        | `artifacts`                                 | `commit`                | Release asset                        |
| ------------------------ | ---------------------- | ------------------------------------------- | ----------------------- | ------------------------------------ |
| [Python](python)         | preinstalled           | `dist/*.whl`                                | `pyproject.toml`        | `hello-X.Y.Z-py3-none-any.whl`       |
| [Go](go)                 | `actions/setup-go`     | `dist/*`                                    | none                    | `hello-X.Y.Z-<os>-<arch>`, 4 targets |
| [TypeScript](typescript) | preinstalled           | `dist/*.tgz`                                | `package-lock.json`     | `hello-ts-X.Y.Z.tgz` (`npm pack`)    |
| [JavaScript](javascript) | preinstalled           | `dist/*.tgz`                                | none                    | `hello-js-X.Y.Z.tgz` (`npm pack`)    |
| [Rust](rust)             | preinstalled           | `dist/hello.tar.gz=hello-${version}.tar.gz` | `Cargo.toml,Cargo.lock` | `hello-X.Y.Z.tar.gz`                 |
| [Java](java)             | preinstalled (JDK 17)  | `dist/hello.jar=hello-${version}.jar`       | none                    | `hello-X.Y.Z.jar`                    |
| [Kotlin](kotlin)         | preinstalled           | `dist/hello.jar=hello-${version}.jar`       | none                    | `hello-X.Y.Z.jar`                    |
| [C](c)                   | preinstalled           | `dist/hello.tar.gz=hello-${version}.tar.gz` | none                    | `hello-X.Y.Z.tar.gz`                 |
| [Bun](bun)               | `oven-sh/setup-bun@v2` | `dist/hello=hello-${version}`               | none                    | `hello-X.Y.Z` (standalone binary)    |
| [Generic](artifact)      | none                   | `dist/hello.tar.gz=hello-${version}.tar.gz` | none                    | `hello-X.Y.Z.tar.gz`                 |
| [Docker](docker)         | none                   | `docker: true`                              | none                    | `ghcr.io/<owner>/<repo>:X.Y.Z`       |

"Preinstalled" means on GitHub's `ubuntu-latest` runner. Where the version lives
in a file other than `package.json`, `build.sh` bumps it and `commit` adds it
(and its lockfile) to the release commit. Where it doesn't (Go, Java, Kotlin,
C), the version is stamped in at build time and the tag is the only record.

The [Examples workflow](../.github/workflows/examples.yml) releases every folder
above with exactly these settings, one job per example. It builds releaser from
the current commit, makes the folder a fresh repo with a local git remote and a
[fake GitHub API](../tests/fakegithub/main.go), then plays the same history as
the integration tests, one step per commit:

| Commit                        | Release    |
| ----------------------------- | ---------- |
| `feat: hello`                 | `v1.0.0`   |
| `fix(build): smaller archive` | `v1.0.1`   |
| `feat: greet by name`         | `v1.1.0`   |
| `chore: tidy`                 | no release |
| `feat!: new greeting format`  | `v2.0.0`   |

It then checks the four tags, every uploaded asset (or, for Docker, the image
tags in a throwaway registry), the manifest ending at `2.0.0`, and that the
build left nothing uncommitted outside `.gitignore`. Each job's summary shows
the resulting `CHANGELOG.md`.
