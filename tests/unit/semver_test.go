package unit

import (
	"testing"

	"github.com/orochibraru/releaser/internal/conventional"
	"github.com/orochibraru/releaser/internal/semver"
)

func TestLatest(t *testing.T) {
	tag, v, ok := semver.Latest([]string{"v1.2.3", "v1.10.0", "v2.0.0-rc.1", "v1.9.9", "vnext", "v01.2.3"})
	if !ok || tag != "v1.10.0" || v != (semver.Version{1, 10, 0}) {
		t.Errorf("Latest = %s %v %v", tag, v, ok)
	}
	if _, _, ok := semver.Latest([]string{"release-1"}); ok {
		t.Error("found a tag in junk")
	}
}

func TestNext(t *testing.T) {
	v := semver.Version{1, 2, 3}
	for level, want := range map[int]string{
		conventional.Patch: "1.2.4",
		conventional.Minor: "1.3.0",
		conventional.Major: "2.0.0",
	} {
		if got := v.Next(level).String(); got != want {
			t.Errorf("Next(%d) = %s, want %s", level, got, want)
		}
	}
}

func TestPre(t *testing.T) {
	v := semver.Version{1, 2, 0}
	tags := []string{"v1.2.0-canary.1", "v1.2.0-canary.10", "v1.2.0-canary.9", "v1.1.0-canary.40", "v1.2.0-beta.7", "v1.2.0-canary.x"}
	if got := v.Pre("canary", tags); got != "1.2.0-canary.11" {
		t.Errorf("Pre = %s, want 1.2.0-canary.11", got)
	}
	if got := v.Pre("beta", nil); got != "1.2.0-beta.1" {
		t.Errorf("first Pre = %s", got)
	}
}

func TestFromReleaseCommit(t *testing.T) {
	for msg, want := range map[string]string{
		"chore(release): 1.4.0":                         "1.4.0",
		"chore(release): 1.4.0 (#12)\n\n## notes":       "1.4.0", // squash merge
		"chore(release): 1.4.0 [skip ci]":               "",      // direct mode commit
		"chore(release): 1.4.0-canary.1":                "",
		"fix: chore(release): 1.4.0":                    "",
		"Merge pull request #3 from o/releaser/release": "",
	} {
		if got, ok := semver.FromReleaseCommit(msg); got != want || ok != (want != "") {
			t.Errorf("FromReleaseCommit(%q) = %q %v, want %q", msg, got, ok, want)
		}
	}
}
