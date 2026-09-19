# Artifact Mode

Attach build outputs to the GitHub release.

```yaml
- uses: orochibraru/releaser@v1
  with:
    prepare: bun run build
    artifacts: |
      dist/extension.zip=extension-${version}.zip
      dist/*.tar.gz
```

Each entry is `path[=name]`, separated by newlines or commas:

- `path` is a file or a glob, relative to the repository root.
- `name` is the asset name on the release, with `${version}` replaced. Without
  it, the file's base name is used.

Artifacts are resolved after `prepare`, so they can be built there. Every entry
must match at least one file, or the run fails **before** anything is pushed.

Uploads need a token (the action passes `github.token` by default) and the job
needs `contents: write`.

Artifact mode and [Docker mode](docker.md) can be enabled together.

A runnable example lives in [`examples/artifact`](../examples/artifact).
