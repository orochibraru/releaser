package unit

import (
	"testing"

	"github.com/orochibraru/releaser/internal/git"
)

func TestSlug(t *testing.T) {
	for remote, want := range map[string]string{
		"git@github.com:orochibraru/bercail.git":      "orochibraru/bercail",
		"https://github.com/orochibraru/bercail":      "orochibraru/bercail",
		"https://github.com/orochibraru/bercail.git/": "orochibraru/bercail",
		"": "",
	} {
		if got := git.Slug(remote); got != want {
			t.Errorf("Slug(%q) = %q, want %q", remote, got, want)
		}
	}
}
