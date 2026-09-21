package integration

import (
	"os"
	"path/filepath"
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

	// Dry run touches nothing but still writes what it would release to the outputs.
	outputs := filepath.Join(r.work, "..", "output")
	r.env = append(r.env, "GITHUB_OUTPUT="+outputs)
	head := r.run(r.remote, "git", "rev-parse", "main")
	r.commit("fix: bug")
	r.release("-dry-run")
	if got := r.remoteTags(); got != "v1.0.0" {
		t.Errorf("dry run pushed: %q", got)
	}
	if got := r.run(r.remote, "git", "rev-parse", "main"); got != head {
		t.Errorf("dry run moved remote main: %s -> %s", head, got)
	}
	if got := readFile(outputs); got != "released=false\nversion=1.0.1\ntag=v1.0.1\n" {
		t.Errorf("dry run GITHUB_OUTPUT = %q", got)
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

	// Nothing to release: a dry run leaves version unset, so consumers can test version != ''.
	os.Remove(outputs)
	r.release("-dry-run")
	if got := readFile(outputs); got != "released=false\n" {
		t.Errorf("no-op dry run GITHUB_OUTPUT = %q", got)
	}

	// Wrong branch is a no-op.
	r.run(r.work, "git", "checkout", "-qb", "feature")
	r.commit("feat!: nope")
	if out := r.release(); !strings.Contains(out, "skipping") {
		t.Errorf("feature branch released:\n%s", out)
	}
}

// -draft creates the GitHub release as a draft (a JSON boolean) and still uploads assets and writes outputs.
func TestDraftRelease(t *testing.T) {
	gh := newFakeGitHub(t)
	r := newRepo(t, "")
	r.env = append(r.env, gh.env()...)
	outputs := filepath.Join(r.work, "..", "output")
	r.env = append(r.env, "GITHUB_OUTPUT="+outputs)
	r.write("a.txt", "a")

	r.commit("feat: first")
	if log := r.release("-artifacts", "a.txt"); !strings.Contains(log, "released ") {
		t.Errorf("non-draft log:\n%s", log)
	}
	r.commit("fix: second")
	if log := r.release("-draft", "-artifacts", "a.txt=a-${version}.txt"); !strings.Contains(log, "created draft ") {
		t.Errorf("draft log:\n%s", log)
	}

	if len(gh.releases) != 2 || gh.releases[0]["draft"] != false || gh.releases[1]["draft"] != true {
		t.Fatalf("draft fields = %v", gh.releases)
	}
	if string(gh.assets["a-1.0.1.txt"]) != "a" {
		t.Errorf("draft asset not uploaded: %v", gh.assets)
	}
	if got := readFile(outputs); !strings.HasSuffix(got, "released=true\nversion=1.0.1\ntag=v1.0.1\n") {
		t.Errorf("GITHUB_OUTPUT = %q", got)
	}
}
