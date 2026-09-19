---
name: fix-ci
description:
  Diagnose and fix a failed GitHub Actions run of this repo (CI workflow
  prek/test/release jobs, Examples workflow). Use when the user says CI failed,
  is red, or a run broke.
---

# Fix CI

1. `gh run list -L 5`, then
   `gh run view <id> --log-failed | cut -c1-250 | tail -80`.
2. Reproduce locally the way CI runs it:
   - `test` job: `go test ./... -count=1`.
   - `prek` job: `prek run --all-files`.
   - Examples workflow: run the failing job's steps by hand (see the add-example
     skill, step 4), for that one example.
   - `release` job: rejected pushes to `main` come from the repository ruleset
     (PR required, status checks), not from code. Report it; the fix is a repo
     setting the user owns.
3. Differences between this Mac and `ubuntu-latest` that have bitten before:
   - Integration test repos run with `HOME` set to a temp dir. Tools that keep
     state under `$HOME` (docker config, rustup) must get it passed back
     explicitly in the test env.
   - A background server started in a step must redirect its output
     (`> log 2>&1 &`), or the step can hang on the open pipe.
   - `run:` values containing `:` must be quoted, or the workflow is invalid
     YAML.
   - GNU vs BSD `sed -i`: use `sed -i.bak` then delete the backup.
   - Toolchains differ (Homebrew cargo vs rustup). Check actions/runner-images
     before assuming a tool is there.
4. Fix the root cause in the shared place (test harness, `build.sh`), not by
   skipping the test. Run the failing test locally, then report. The user
   commits and pushes; watch the next run with `gh run watch <id>` if asked.
