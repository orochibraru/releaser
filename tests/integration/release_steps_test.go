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
func releaseSteps(t *testing.T, r *repo, args ...string) {
	t.Helper()
	for _, s := range semverSteps {
		r.commit(s.msg)
		out := r.release(args...)
		if s.tag == "" {
			if !strings.Contains(out, "no release-worthy commits") {
				t.Fatalf("%q released:\n%s", s.msg, out)
			}
			continue
		}
		if tags := strings.Fields(r.remoteTags()); tags[len(tags)-1] != s.tag {
			t.Fatalf("%q: tags = %v, want last %s", s.msg, tags, s.tag)
		}
	}
}

// checkChangelog asserts CHANGELOG.md holds every release from semverSteps, newest first,
// with the right sections and nothing from non-release commits.
func checkChangelog(t *testing.T, r *repo) {
	t.Helper()
	c := r.read("CHANGELOG.md")
	if !strings.HasPrefix(c, "# Changelog\n\n## [2.0.0](") || strings.Count(c, "# Changelog") != 1 {
		t.Errorf("CHANGELOG.md must start with one title then the newest release:\n%s", c)
	}
	if !strings.HasSuffix(c, "\n") || strings.HasSuffix(c, "\n\n") {
		t.Errorf("CHANGELOG.md must end with exactly one newline: %q", c[max(0, len(c)-20):])
	}

	// Newest first; every heading but the first release links to its compare view.
	last := -1
	for i := len(semverSteps) - 1; i >= 0; i-- {
		tag := semverSteps[i].tag
		if tag == "" {
			continue
		}
		heading := "## [" + tag[1:] + "]("
		if tag == "v1.0.0" {
			heading = "## 1.0.0 ("
		}
		idx := strings.Index(c, heading)
		if idx <= last {
			t.Errorf("heading %q missing or out of order in:\n%s", heading, c)
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
		if !strings.Contains(c, want) {
			t.Errorf("CHANGELOG.md missing %q:\n%s", want, c)
		}
	}
	if strings.Contains(c, "tidy") {
		t.Errorf("chore leaked into CHANGELOG.md:\n%s", c)
	}
}
