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
