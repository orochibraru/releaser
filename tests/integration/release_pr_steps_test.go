package integration

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/orochibraru/releaser/internal/changelog"
)

// Trunk-based: every push to main ships a canary and refreshes the release PR; merging the PR
// (squash, then merge commit) tags the stable release on the release commit.
func TestReleasePR(t *testing.T) {
	gh := newFakeGitHub(t)
	r := newRepo(t, "")
	r.env = append(r.env, gh.env()...)
	outputs := filepath.Join(r.work, "..", "output")
	r.env = append(r.env, "GITHUB_OUTPUT="+outputs)
	r.write("package.json", "{\n  \"name\": \"x\",\n  \"version\": \"0.0.0\"\n}\n")
	r.run(r.work, "git", "add", "package.json")
	args := []string{"-prerelease", "canary", "-release-pr", "-prepare", "echo ${version} > prepared.txt", "-commit", "prepared.txt"}

	// push lands msg on remote main, like a merged feature PR, then runs releaser in a fresh checkout.
	push := func(msg string) string {
		t.Helper()
		if msg != "" {
			r.commit(msg)
		}
		r.run(r.work, "git", "push", "-q", "origin", "main")
		os.Remove(outputs)
		out := r.release(args...)
		r.run(r.work, "git", "reset", "-q", "--hard")
		r.run(r.work, "git", "clean", "-fdq")
		return out
	}
	// merge merges the open release PR the given way and marks it merged in the fake.
	merge := func(squash bool) {
		t.Helper()
		r.run(r.work, "git", "fetch", "-q", "origin", "releaser/release")
		title := gh.pulls[len(gh.pulls)-1]["title"].(string)
		if squash {
			r.run(r.work, "git", "merge", "-q", "--squash", "FETCH_HEAD")
			r.commit(title + " (#" + strconv.Itoa(len(gh.pulls)) + ")")
		} else {
			r.run(r.work, "git", "-c", "user.name=t", "-c", "user.email=t@t", "merge", "-q", "--no-ff", "-m", "Merge pull request #"+strconv.Itoa(len(gh.pulls))+" from o/releaser/release", "FETCH_HEAD")
		}
		gh.pulls[len(gh.pulls)-1]["state"] = "merged"
	}
	openPR := func() map[string]any {
		t.Helper()
		var open []map[string]any
		for _, pr := range gh.pulls {
			if pr["state"] == "open" {
				open = append(open, pr)
			}
		}
		if len(open) != 1 {
			t.Fatalf("want 1 open PR, got %v", gh.pulls)
		}
		return open[0]
	}
	lastRelease := func(tag string, prerelease bool) map[string]any {
		t.Helper()
		rel := gh.releases[len(gh.releases)-1]
		if rel["tag_name"] != tag || rel["prerelease"] != prerelease {
			t.Fatalf("last GitHub release = %v %v, want %s prerelease=%v", rel["tag_name"], rel["prerelease"], tag, prerelease)
		}
		return rel
	}

	// Dry run with the release PR alone reports the stable version it would release.
	r.commit("feat: hello")
	r.run(r.work, "git", "push", "-q", "origin", "main")
	r.release("-release-pr", "-dry-run")
	if got := readFile(outputs); got != "released=false\nversion=1.0.0\ntag=v1.0.0\nprerelease=false\n" {
		t.Errorf("release PR dry run GITHUB_OUTPUT = %q", got)
	}

	// feat → first canary, and a PR for 1.0.0 whose commit bumps the files and has no [skip ci].
	push("")
	lastRelease("v1.0.0-canary.1", true)
	if got := readFile(outputs); got != "released=true\nversion=1.0.0-canary.1\ntag=v1.0.0-canary.1\nprerelease=true\n" {
		t.Errorf("canary GITHUB_OUTPUT = %q", got)
	}
	if pr := openPR(); pr["title"] != "chore(release): 1.0.0" || pr["head"] != "releaser/release" || pr["base"] != "main" {
		t.Errorf("PR = %v", pr)
	}
	if msg := r.run(r.remote, "git", "log", "-1", "--format=%s", "releaser/release"); strings.TrimSpace(msg) != "chore(release): 1.0.0" {
		t.Errorf("release PR commit = %q", msg)
	}
	if pkg := r.run(r.remote, "git", "show", "releaser/release:package.json"); !strings.Contains(pkg, `"version": "1.0.0"`) {
		t.Errorf("PR package.json not bumped:\n%s", pkg)
	}
	// prepare ran for the canary first, then again for the PR: the PR commit has the stable version.
	if got := r.run(r.remote, "git", "show", "releaser/release:prepared.txt"); got != "1.0.0\n" {
		t.Errorf("PR prepared.txt = %q", got)
	}

	// fix → next canary of the same base, same PR updated.
	push("fix: typo")
	lastRelease("v1.0.0-canary.2", true)
	if body := openPR()["body"].(string); !strings.Contains(body, "* typo (") || !strings.Contains(body, "* hello (") {
		t.Errorf("PR body not updated:\n%s", body)
	}

	// chore → nothing new since the last canary.
	if out := push("chore: tidy"); !strings.Contains(out, "no release-worthy commits since the last canary") {
		t.Errorf("chore made a canary:\n%s", out)
	}

	// Squash-merge the PR → v1.0.0 on the merged commit, notes straight from its CHANGELOG entry, no canary.
	merge(true)
	head := strings.TrimSpace(r.run(r.work, "git", "rev-parse", "HEAD"))
	push("")
	rel := lastRelease("v1.0.0", false)
	if got := readFile(outputs); got != "released=true\nversion=1.0.0\ntag=v1.0.0\nprerelease=false\n" {
		t.Errorf("stable GITHUB_OUTPUT = %q", got)
	}
	if got := strings.TrimSpace(r.run(r.remote, "git", "rev-list", "-n1", "v1.0.0")); got != head {
		t.Errorf("v1.0.0 on %s, want the squash commit %s", got, head)
	}
	if want := changelog.Latest(r.read("CHANGELOG.md")); rel["body"] != want || !strings.Contains(want, "* typo (") {
		t.Errorf("release body %q, want CHANGELOG entry %q", rel["body"], want)
	}

	// Re-running on the same commit changes nothing.
	if out := push(""); !strings.Contains(out, "no release-worthy commits since v1.0.0") {
		t.Errorf("rerun:\n%s", out)
	}

	// feat → canary of the next base, a new PR; a merge commit tags the release commit behind it.
	push("feat: greet by name")
	lastRelease("v1.1.0-canary.1", true)
	if pr := openPR(); pr["title"] != "chore(release): 1.1.0" {
		t.Errorf("second PR = %v", pr)
	}
	merge(false)
	push("")
	lastRelease("v1.1.0", false)
	if tagged, second := r.run(r.remote, "git", "rev-list", "-n1", "v1.1.0"), r.run(r.remote, "git", "rev-parse", "main^2"); tagged != second {
		t.Errorf("v1.1.0 on %s, want the release commit %s", tagged, second)
	}

	push("feat!: new greeting format\n\nBREAKING CHANGE: greetings are now uppercase")
	lastRelease("v2.0.0-canary.1", true)
	if pr := openPR(); pr["title"] != "chore(release): 2.0.0" {
		t.Errorf("third PR = %v", pr)
	}

	// A failing prepare pushes nothing.
	r.commit("fix: oops")
	r.run(r.work, "git", "push", "-q", "origin", "main")
	if out, err := r.tryRelease("-prerelease", "canary", "-release-pr", "-prepare", "false"); err == nil {
		t.Errorf("failing prepare succeeded:\n%s", out)
	}
	if got := strings.Fields(r.remoteTags()); strings.Join(got, " ") != "v1.0.0 v1.0.0-canary.1 v1.0.0-canary.2 v1.1.0 v1.1.0-canary.1 v2.0.0-canary.1" {
		t.Errorf("remote tags = %v", got)
	}

	// Main's CHANGELOG has the two stable releases and no canary.
	c := r.run(r.remote, "git", "show", "main:CHANGELOG.md")
	if !strings.HasPrefix(c, "# Changelog\n\n## [1.1.0](https://github.com/o/r/compare/v1.0.0...v1.1.0) (") ||
		!strings.Contains(c, "\n## 1.0.0 (") || strings.Contains(c, "canary") || strings.Count(c, "\n## ") != 2 ||
		!strings.HasSuffix(c, ")\n") {
		t.Errorf("main CHANGELOG.md:\n%s", c)
	}
}
