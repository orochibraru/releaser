package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The core flow, no modes: versions, CHANGELOG, package.json, prepare, rules, branches.
func TestRelease(test *testing.T) {
	repository := newRepo(test, "../../examples/artifact") // just for its package.json
	repository.commit("feat: first")

	// First release is 1.0.0, prepare sees the version, package.json bumped, commit + tag pushed.
	repository.release("-prepare", "echo ${version} > prepared.txt")
	if got := repository.remoteTags(); got != "v1.0.0" {
		test.Fatalf("tags after first release = %q", got)
	}
	if pkg, prepared, changelog := repository.read("package.json"), repository.read("prepared.txt"), repository.read("CHANGELOG.md"); !strings.Contains(pkg, `"version": "1.0.0"`) ||
		strings.TrimSpace(prepared) != "1.0.0" || !strings.Contains(changelog, "* first (") {
		test.Errorf("package.json=%s prepared=%s changelog=%s", pkg, prepared, changelog)
	}
	if msg := repository.run(repository.remote, "git", "log", "-1", "--format=%s", "main"); strings.TrimSpace(msg) != "chore(release): 1.0.0 [skip ci]" {
		test.Errorf("release commit = %q", msg)
	}

	// The release commit itself (chore) never triggers another release.
	if out := repository.release(); !strings.Contains(out, "no release-worthy commits") {
		test.Errorf("chore-only run released:\n%s", out)
	}

	// Dry run touches nothing but still writes what it would release to the outputs.
	outputs := filepath.Join(repository.work, "..", "output")
	repository.env = append(repository.env, "GITHUB_OUTPUT="+outputs)
	head := repository.run(repository.remote, "git", "rev-parse", "main")
	repository.commit("fix: bug")
	repository.release("-dry-run")
	if got := repository.remoteTags(); got != "v1.0.0" {
		test.Errorf("dry run pushed: %q", got)
	}
	if got := repository.run(repository.remote, "git", "rev-parse", "main"); got != head {
		test.Errorf("dry run moved remote main: %s -> %s", head, got)
	}
	if got := readFile(outputs); got != "released=false\nversion=1.0.1\ntag=v1.0.1\nprerelease=false\n" {
		test.Errorf("dry run GITHUB_OUTPUT = %q", got)
	}

	// feat bumps minor by default, patch with custom rules.
	repository.commit("feat: more")
	repository.release("-rules", "feat=patch")
	if got := repository.remoteTags(); !strings.HasSuffix(got, "v1.0.1") {
		test.Errorf("custom rules tags = %q", got)
	}
	repository.commit("feat: again")
	repository.release()
	if got := repository.remoteTags(); !strings.HasSuffix(got, "v1.1.0") {
		test.Errorf("default rules tags = %q", got)
	}

	// Nothing to release: a dry run leaves version unset, so consumers can test version != ''.
	os.Remove(outputs)
	repository.release("-dry-run")
	if got := readFile(outputs); got != "released=false\n" {
		test.Errorf("no-op dry run GITHUB_OUTPUT = %q", got)
	}

	// Wrong branch is a no-op.
	repository.run(repository.work, "git", "checkout", "-qb", "feature")
	repository.commit("feat!: nope")
	if out := repository.release(); !strings.Contains(out, "skipping") {
		test.Errorf("feature branch released:\n%s", out)
	}
}

// -draft creates the GitHub release as a draft (a JSON boolean) and still uploads assets and writes outputs.
func TestDraftRelease(test *testing.T) {
	gh := newFakeGitHub(test)
	repository := newRepo(test, "")
	repository.env = append(repository.env, gh.env()...)
	outputs := filepath.Join(repository.work, "..", "output")
	repository.env = append(repository.env, "GITHUB_OUTPUT="+outputs)
	repository.write("a.txt", "a")

	repository.commit("feat: first")
	if log := repository.release("-artifacts", "a.txt"); !strings.Contains(log, "released ") {
		test.Errorf("non-draft log:\n%s", log)
	}
	repository.commit("fix: second")
	if log := repository.release("-draft", "-artifacts", "a.txt=a-${version}.txt"); !strings.Contains(log, "created draft ") {
		test.Errorf("draft log:\n%s", log)
	}

	if len(gh.releases) != 2 || gh.releases[0]["draft"] != false || gh.releases[1]["draft"] != true {
		test.Fatalf("draft fields = %v", gh.releases)
	}
	if string(gh.assets["a-1.0.1.txt"]) != "a" {
		test.Errorf("draft asset not uploaded: %v", gh.assets)
	}
	if got := readFile(outputs); !strings.HasSuffix(got, "released=true\nversion=1.0.1\ntag=v1.0.1\nprerelease=false\n") {
		test.Errorf("GITHUB_OUTPUT = %q", got)
	}
}
