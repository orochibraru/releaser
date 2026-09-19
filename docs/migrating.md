# Migrating from semantic-release

Delete `.releaserc.json`, the plugins in `package.json` and the
`npx semantic-release` step. Each plugin maps to built-in behavior or one input.

| semantic-release plugin                         | releaser                                                            |
| ----------------------------------------------- | ------------------------------------------------------------------- |
| `commit-analyzer` (conventionalcommits)         | Built in. `releaseRules` → [`rules`](versioning.md#rules).          |
| `release-notes-generator` (conventionalcommits) | Built in, same section layout.                                      |
| `changelog`                                     | Built in, always `CHANGELOG.md`.                                    |
| `npm` (version bump only)                       | Built in: `package.json` `version` is bumped. Nothing is published. |
| `exec` `prepareCmd`                             | `prepare`, with `${nextRelease.version}` → `${version}`.            |
| `git` `assets`                                  | `CHANGELOG.md` and `package.json` built in, the rest via `commit`.  |
| `github` `assets`                               | [`artifacts`](artifacts.md).                                        |
| `tagFormat: "v${version}"`                      | Always `v${version}`.                                               |
| `branches: ["main"]`                            | `branch`, default `main`.                                           |

## Example

This `.releaserc.json`:

```json
{
  "branches": ["main"],
  "plugins": [
    [
      "@semantic-release/commit-analyzer",
      {
        "preset": "conventionalcommits",
        "releaseRules": [
          { "breaking": true, "release": "patch" },
          { "type": "feat", "release": "patch" },
          { "type": "docs", "release": "patch" },
          { "type": "refactor", "release": "patch" }
        ]
      }
    ],
    [
      "@semantic-release/release-notes-generator",
      { "preset": "conventionalcommits" }
    ],
    ["@semantic-release/changelog", { "changelogFile": "CHANGELOG.md" }],
    [
      "@semantic-release/exec",
      {
        "prepareCmd": "bun scripts/bump-version.ts ${nextRelease.version} && bun run package:extension"
      }
    ],
    ["@semantic-release/git", { "assets": ["CHANGELOG.md", "package.json"] }],
    [
      "@semantic-release/github",
      {
        "assets": [
          {
            "path": "dist/extension.zip",
            "name": "extension-${nextRelease.version}.zip"
          }
        ]
      }
    ]
  ],
  "tagFormat": "v${version}"
}
```

becomes:

```yaml
- uses: orochibraru/releaser@v1
  with:
    rules: breaking=patch,feat=patch,docs=patch,refactor=patch
    prepare: bun scripts/bump-version.ts ${version} && bun run package:extension
    artifacts: dist/extension.zip=extension-${version}.zip
```

## Differences

- Overrides merge with the defaults per key. semantic-release instead ignores
  the defaults for a commit once any custom rule matches it.
- One release branch; no prerelease or maintenance channels.
- No npm publish. Run it in `prepare`, or in a later step gated on
  `steps.<id>.outputs.released`.
- Outside CI the CLI is a dry run, like semantic-release's `--no-ci` guard.
