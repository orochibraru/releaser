# Versioning

## Commit format

Commits follow [Conventional Commits](https://www.conventionalcommits.org):

```text
<type>[(scope)][!]: <subject>

[body]

[BREAKING CHANGE: <note>]
```

A commit is breaking when it has `!` after the type/scope or a
`BREAKING CHANGE:` (or `BREAKING-CHANGE:`) footer. The footer text becomes the
entry in the notes' breaking changes section; with only `!`, the subject is
used. Commits that don't match the format are ignored.

## Rules

Each commit asks for a bump; the strongest one wins.

| Key        | Default |
| ---------- | ------- |
| `breaking` | `major` |
| `feat`     | `minor` |
| `fix`      | `patch` |
| `perf`     | `patch` |
| `revert`   | `patch` |
| any other  | `none`  |

`rules` overrides or adds keys with `none`, `patch`, `minor` or `major`:

```yaml
with:
  rules: breaking=patch,feat=patch,docs=patch,refactor=patch
```

Overrides are merged over the defaults, so `fix` and `perf` still bump patch
above. Unknown levels fail the run.

## Tags

- Tags are `vX.Y.Z`. The previous release is the highest such tag reachable from
  `HEAD`; prereleases (`v2.0.0-rc.1`) and other tags are ignored.
- No tag yet → the first release is `1.0.0`, whatever the bump.
- Only commits after that tag count. The release commit is a `chore`, so it
  never triggers another release.

## Release notes

Sections appear in this order, only when they have entries:

1. ⚠ BREAKING CHANGES
2. Features — `feat`
3. Bug Fixes — `fix`
4. Performance Improvements — `perf`
5. Reverts — `revert`
6. Documentation — `docs`
7. Code Refactoring — `refactor`

Other types (`chore`, `ci`, `test`…) can trigger a release through `rules` but
don't appear in the notes. Scopes are shown in bold, and every entry links to
its commit. The heading links to the compare view against the previous tag.
