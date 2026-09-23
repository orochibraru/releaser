# releaser

semantic-release without the plugins, the config, or `node_modules`: one static
Go binary and a GitHub Action (composite, meant for the Marketplace). How it
works is in [docs/architecture.md](docs/architecture.md); inputs and flags in
[docs/options.md](docs/options.md).

## Goals and non-goals

- Out of the box what semantic-release needs a config and a stack of plugins
  for: conventional commits in; version, `CHANGELOG.md`, `package.json` bump,
  release commit, tag, GitHub release out. Plus two optional modes, usable
  together: artifact (attach files) and docker (build and push the image). For
  trunk-based repos: `prerelease` (a tag-only canary per push) and `release-pr`
  (release-please style PR; merging it tags the stable release).
- Fast, zero dependencies: Go stdlib only. Never add a module.
- No config file. Flags only, mapped 1:1 to action inputs of the same name.
- Deferred on purpose: prerelease channels per branch, multi-branch releases,
  Windows binaries of releaser itself.
- Positioning: not GoReleaser (that starts from an existing tag and does
  packaging; the two chain). The nearest competitor is go-semantic-release,
  which downloads plugins at runtime.

## Invariants

- Everything that can fail (prepare, artifact resolution, docker push) runs
  before one atomic push of commit and tag. A failed run leaves the remote
  untouched; the tests check it. Only the Docker alias (`:latest`) moves after
  the push, so a rejected push never moves it.
- `CHANGELOG.md` ends with exactly one newline (CI runs `end-of-file-fixer` on
  all files, including the release commit's output).
- Default bumps are plain semver. Custom schemes are one `rules` input (e.g.
  `breaking=patch,feat=patch,docs=patch,refactor=patch`), not code.
- `action.yml` downloads the prebuilt binary for its pinned `RELEASER_VERSION`
  and falls back to `go build`. The dogfood `release` job in `ci.yml` rewrites
  that pin in `prepare`, cross-compiles 4 binaries, and force-moves the major
  tag (`v1`), because Marketplace users pin `@v1`.
- A flag change touches `cmd/releaser/main.go`, `action.yml` and
  `docs/options.md` together. The `docs-sync-reviewer` agent checks this.

## Layout (the user's structure; keep it)

`cmd/releaser/` is the CLI (so `go install .../cmd/releaser` names the binary
`releaser`), `internal/<concern>/` the libraries, `tests/unit` and
`tests/integration` the tests, one file per concern and file names that say
what's inside. Don't collapse into fewer files.

## Testing

- `mise install` (go and prek, pinned in `mise.toml`), `prek install`, then
  hooks run on commit: gofmt, `go mod tidy -diff`, go vet,
  `go test -short ./...`, prettier and markdownlint on Markdown, pinact on
  workflows.
- `go test ./...` is the full suite. Integration tests build the binary and
  release throwaway repos into local bare remotes, with a fake GitHub API
  (`fake_github_test.go`) and, for Docker, a `registry:3` container. Every
  example goes through the history in `release_steps_test.go` (1.0.0, 1.0.1,
  1.1.0, no release, 2.0.0) and the final `CHANGELOG.md` is asserted in full.
- Integration repos get a temp `HOME`: pass through anything a tool keeps there
  (`DOCKER_CONFIG` already is).
- The Examples workflow (`.github/workflows/examples.yml`) releases every
  `examples/<name>` for real: releaser from the current commit, local remote,
  `tests/fakegithub` for the API, one step per commit via
  `.github/scripts/release-example.sh`. Adding an example: the `add-example`
  skill.
- When adding a check, prove it can fail by breaking the code once.
- `mise run cover` (`.github/scripts/coverage.sh`, gotestsum) measures `cmd/`
  and `internal/` across all tests, including every run of the integration
  binary (built with `-cover`, `GOCOVERDIR` via `RELEASER_COVERDIR`). CI's
  `test` job runs it and fails under 90%.

## Conventions

- Conventional commits (`feat: ...`, `fix: ci`). The user commits and pushes. CI
  ships a canary per push to `main` and keeps a release PR open; merging it
  lands `chore(release): X.Y.Z (#N)` and tags the stable release.
- Docs follow the svelte-smol convention: `docs/README.md` is the reading-order
  index, `docs/config.json` holds categories, titles and icons. The root README
  stays short and links to `docs/`; each table lives in one place. Check every
  claim against the code.
- Prettier rewraps Markdown (`proseWrap: always`, 80 columns) and realigns
  tables: keep its output. `CHANGELOG.md` is excluded from both Markdown hooks.
- Actions in workflows and docs are pinned to their latest tag, checked with
  `git ls-remote --tags --refs`; a floating major only if that tag exists.
  Third-party actions in this repo's workflows (not `actions/*`, not docs) are
  pinned by commit SHA with the tag in a comment; the pinact hook enforces it
  (`.pinact.yaml` exempts `actions/*`). Workflows default to
  `permissions: contents: read`.
- `action.yml` pins a SHA-256 per prebuilt binary; the dogfood `prepare`
  rewrites them. Binaries are built with `-buildvcs=false` so the release PR and
  its merge produce the same bytes.
- Shell in `build.sh` and scripts is POSIX `sh`: `sed -i.bak` then delete the
  backup (BSD sed on the user's Mac, GNU in CI; no `\b` in BSD sed).

## Open items

- The action is named "Releaser", probably taken on the Marketplace; rename
  before submitting.
- Kotlin isn't installed locally; its example only runs in CI.
