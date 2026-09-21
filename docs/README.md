# Documentation

The guides in this folder, in reading order:

1. [Getting started](getting-started.md) — add the action, push, get a release.
2. [Inputs and flags](options.md) — every action input and CLI flag.
3. [Migrating from semantic-release](migrating.md) — which plugin maps to which
   input.
4. [Release flows](flows.md) — straight to main, canaries, release PR, with
   diagrams.
5. [Versioning](versioning.md) — commit format, bump rules, tags.
6. [Artifact mode](artifacts.md) — attaching files to the GitHub release.
7. [Docker mode](docker.md) — building and pushing the release image.
8. [Canaries and release PR](trunk.md) — trunk-based development: a canary per
   push, a PR per stable release.
9. [Architecture](architecture.md) — the release pipeline and the action.

This file is the index when someone reads the repo on GitHub; the docs site
ignores it. The site's categories, order, titles and icons come from
[`config.json`](config.json).
