package integration

import (
	"path/filepath"
	"strings"
	"testing"
)

// actions/checkout clones with depth 1 by default: releaser unshallows to see the previous tag,
// and commits as the configured identity when there is one.
func TestShallowCheckout(test *testing.T) {
	repository := newRepo(test, "")
	repository.commit("feat: first")
	repository.release()
	repository.commit("fix: second")
	repository.run(repository.work, "git", "push", "-q", "origin", "main")

	ci := filepath.Join(repository.work, "..", "ci")
	repository.run(repository.work, "git", "clone", "-q", "--depth", "1", "--no-tags", "file://"+repository.remote, ci)
	repository.run(ci, "git", "config", "user.name", "Release Bot")
	repository.run(ci, "git", "config", "user.email", "bot@example.com")
	out := repository.run(ci, bin)
	if !strings.Contains(out, "next release: v1.0.1") {
		test.Fatalf("shallow clone lost the previous tag:\n%s", out)
	}
	if got := repository.remoteTags(); got != "v1.0.0\nv1.0.1" {
		test.Errorf("remote tags = %q", got)
	}
	if got := strings.TrimSpace(repository.run(repository.remote, "git", "log", "-1", "--format=%an <%ae>", "main")); got != "Release Bot <bot@example.com>" {
		test.Errorf("release commit author = %q", got)
	}
}
