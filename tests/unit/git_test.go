package unit

import (
	"testing"

	"github.com/orochibraru/releaser/internal/git"
)

func TestSlug(test *testing.T) {
	for remote, want := range map[string]string{
		"git@github.com:orochibraru/bercail.git":       "orochibraru/bercail",
		"https://github.com/orochibraru/bercail":       "orochibraru/bercail",
		"https://github.com/orochibraru/bercail.git/":  "orochibraru/bercail",
		"ssh://git@github.com/orochibraru/bercail.git": "orochibraru/bercail",
		"https://ghe.example.com/team/app.git":         "team/app",
		"":                                             "",
		"/tmp/x/remote.git":                            "", // local remotes (the integration tests) have no GitHub repo
		"file:///tmp/x/remote.git":                     "",
		"../remote.git":                                "",
		"https://github.com/a/b/c.git":                 "",
	} {
		if got := git.Slug(remote); got != want {
			test.Errorf("Slug(%q) = %q, want %q", remote, got, want)
		}
	}
}
