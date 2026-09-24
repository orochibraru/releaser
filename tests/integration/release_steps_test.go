package integration

import (
	"strings"
	"testing"
)

// semverSteps is the commit history every example is released through: one commit per step,
// then a release, which must produce tag ("" = no release).
var semverSteps = []struct{ msg, tag string }{
	{"feat: hello", "v1.0.0"},                 // first release is always 1.0.0
	{"fix(build): smaller archive", "v1.0.1"}, // fix → patch
	{"feat: greet by name", "v1.1.0"},         // feat → minor
	{"chore: tidy", ""},                       // chore → no release
	{"feat!: new greeting format\n\nBREAKING CHANGE: greetings are now uppercase", "v2.0.0"}, // breaking → major
}

// releaseSteps plays semverSteps, releasing with args after each commit.
func releaseSteps(test *testing.T, repository *repo, args ...string) {
	test.Helper()
	for _, step := range semverSteps {
		repository.commit(step.msg)
		out := repository.release(args...)
		if step.tag == "" {
			if !strings.Contains(out, "no release-worthy commits") {
				test.Fatalf("%q released:\n%s", step.msg, out)
			}
			continue
		}
		if tags := strings.Fields(repository.remoteTags()); tags[len(tags)-1] != step.tag {
			test.Fatalf("%q: tags = %v, want last %s", step.msg, tags, step.tag)
		}
	}
}

// checkChangelog asserts CHANGELOG.md holds every release from semverSteps, newest first,
// with the right sections and nothing from non-release commits.
func checkChangelog(test *testing.T, repository *repo) {
	test.Helper()
	changelogFile := repository.read("CHANGELOG.md")
	if !strings.HasPrefix(changelogFile, "# Changelog\n\n## [2.0.0](") || strings.Count(changelogFile, "# Changelog") != 1 {
		test.Errorf("CHANGELOG.md must start with one title then the newest release:\n%s", changelogFile)
	}
	if !strings.HasSuffix(changelogFile, "\n") || strings.HasSuffix(changelogFile, "\n\n") {
		test.Errorf("CHANGELOG.md must end with exactly one newline: %q", changelogFile[max(0, len(changelogFile)-20):])
	}

	// Newest first; every heading but the first release links to its compare view.
	last := -1
	for index := len(semverSteps) - 1; index >= 0; index-- {
		tag := semverSteps[index].tag
		if tag == "" {
			continue
		}
		heading := "## [" + tag[1:] + "]("
		if tag == "v1.0.0" {
			heading = "## 1.0.0 ("
		}
		idx := strings.Index(changelogFile, heading)
		if idx <= last {
			test.Errorf("heading %q missing or out of order in:\n%s", heading, changelogFile)
		}
		last = idx
	}

	for _, want := range []string{
		"### ⚠ BREAKING CHANGES\n\n* greetings are now uppercase (",
		"### Features\n\n* new greeting format (",
		"### Features\n\n* greet by name (",
		"### Bug Fixes\n\n* **build:** smaller archive (",
		"### Features\n\n* hello (",
	} {
		if !strings.Contains(changelogFile, want) {
			test.Errorf("CHANGELOG.md missing %q:\n%s", want, changelogFile)
		}
	}
	if strings.Contains(changelogFile, "tidy") {
		test.Errorf("chore leaked into CHANGELOG.md:\n%s", changelogFile)
	}
}
