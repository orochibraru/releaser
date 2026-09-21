package integration

import (
	"path/filepath"
	"strings"
	"testing"
)

// actions/checkout clones with depth 1 by default: releaser unshallows to see the previous tag,
// and commits as the configured identity when there is one.
func TestShallowCheckout(t *testing.T) {
	r := newRepo(t, "")
	r.commit("feat: first")
	r.release()
	r.commit("fix: second")
	r.run(r.work, "git", "push", "-q", "origin", "main")

	ci := filepath.Join(r.work, "..", "ci")
	r.run(r.work, "git", "clone", "-q", "--depth", "1", "--no-tags", "file://"+r.remote, ci)
	r.run(ci, "git", "config", "user.name", "Release Bot")
	r.run(ci, "git", "config", "user.email", "bot@example.com")
	out := r.run(ci, bin)
	if !strings.Contains(out, "next release: v1.0.1") {
		t.Fatalf("shallow clone lost the previous tag:\n%s", out)
	}
	if got := r.remoteTags(); got != "v1.0.0\nv1.0.1" {
		t.Errorf("remote tags = %q", got)
	}
	if got := strings.TrimSpace(r.run(r.remote, "git", "log", "-1", "--format=%an <%ae>", "main")); got != "Release Bot <bot@example.com>" {
		t.Errorf("release commit author = %q", got)
	}
}
