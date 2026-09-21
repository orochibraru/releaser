# Documentation

The guides in this folder, in reading order:

1. [Getting started](getting-started.md) — add the action, push, get a release.
2. [Inputs and flags](options.md) — every action input and CLI flag.
3. [Migrating from semantic-release](migrating.md) — which plugin maps to which
   input.
4. [Versioning](versioning.md) — commit format, bump rules, tags.
5. [Artifact mode](artifacts.md) — attaching files to the GitHub release.
6. [Docker mode](docker.md) — building and pushing the release image.
7. [Canaries and release PR](trunk.md) — trunk-based development: a canary per
   push, a PR per stable release.
8. [Architecture](architecture.md) — the release pipeline and how the code is
   laid out.

This file is the index when someone reads the repo on GitHub; the docs site
ignores it. The site's categories, order, titles and icons come from
[`config.json`](config.json).
