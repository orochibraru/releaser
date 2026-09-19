package integration

import (
	"strings"
	"testing"
)

// The core flow, no modes: versions, CHANGELOG, package.json, prepare, rules, branches.
func TestRelease(t *testing.T) {
	r := newRepo(t, "../../examples/artifact") // just for its package.json
	r.commit("feat: first")

	// First release is 1.0.0, prepare sees the version, package.json bumped, commit + tag pushed.
	r.release("-prepare", "echo ${version} > prepared.txt")
	if got := r.remoteTags(); got != "v1.0.0" {
		t.Fatalf("tags after first release = %q", got)
	}
	if pkg, prepared, changelog := r.read("package.json"), r.read("prepared.txt"), r.read("CHANGELOG.md"); !strings.Contains(pkg, `"version": "1.0.0"`) ||
		strings.TrimSpace(prepared) != "1.0.0" || !strings.Contains(changelog, "* first (") {
		t.Errorf("package.json=%s prepared=%s changelog=%s", pkg, prepared, changelog)
	}
	if msg := r.run(r.remote, "git", "log", "-1", "--format=%s", "main"); strings.TrimSpace(msg) != "chore(release): 1.0.0 [skip ci]" {
		t.Errorf("release commit = %q", msg)
	}

	// The release commit itself (chore) never triggers another release.
	if out := r.release(); !strings.Contains(out, "no release-worthy commits") {
		t.Errorf("chore-only run released:\n%s", out)
	}

	// Dry run touches nothing.
	r.commit("fix: bug")
	r.release("-dry-run")
	if got := r.remoteTags(); got != "v1.0.0" {
		t.Errorf("dry run pushed: %q", got)
	}

	// feat bumps minor by default, patch with custom rules.
	r.commit("feat: more")
	r.release("-rules", "feat=patch")
	if got := r.remoteTags(); !strings.HasSuffix(got, "v1.0.1") {
		t.Errorf("custom rules tags = %q", got)
	}
	r.commit("feat: again")
	r.release()
	if got := r.remoteTags(); !strings.HasSuffix(got, "v1.1.0") {
		t.Errorf("default rules tags = %q", got)
	}

	// Wrong branch is a no-op.
	r.run(r.work, "git", "checkout", "-qb", "feature")
	r.commit("feat!: nope")
	if out := r.release(); !strings.Contains(out, "skipping") {
		t.Errorf("feature branch released:\n%s", out)
	}
}
