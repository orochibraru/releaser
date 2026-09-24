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
func TestReleasePR(test *testing.T) {
	gh := newFakeGitHub(test)
	repository := newRepo(test, "")
	repository.env = append(repository.env, gh.env()...)
	outputs := filepath.Join(repository.work, "..", "output")
	repository.env = append(repository.env, "GITHUB_OUTPUT="+outputs)
	repository.write("package.json", "{\n  \"name\": \"x\",\n  \"version\": \"0.0.0\"\n}\n")
	repository.run(repository.work, "git", "add", "package.json")
	// prepare fails if it can see the token.
	args := []string{"-prerelease", "canary", "-release-pr", "-prepare", `test -z "$GITHUB_TOKEN$GH_TOKEN" && echo ${version} > prepared.txt`, "-commit", "prepared.txt"}

	// push lands msg on remote main, like a merged feature PR, then runs releaser in a fresh checkout.
	push := func(msg string) string {
		test.Helper()
		if msg != "" {
			repository.commit(msg)
		}
		repository.run(repository.work, "git", "push", "-q", "origin", "main")
		os.Remove(outputs)
		out := repository.release(args...)
		repository.run(repository.work, "git", "reset", "-q", "--hard")
		repository.run(repository.work, "git", "clean", "-fdq")
		return out
	}
	// merge merges the open release PR the given way and marks it merged in the fake.
	merge := func(squash bool) {
		test.Helper()
		repository.run(repository.work, "git", "fetch", "-q", "origin", "releaser/release")
		gh.mu.Lock()
		defer gh.mu.Unlock()
		title := gh.pulls[len(gh.pulls)-1]["title"].(string)
		if squash {
			repository.run(repository.work, "git", "merge", "-q", "--squash", "FETCH_HEAD")
			repository.commit(title + " (#" + strconv.Itoa(len(gh.pulls)) + ")")
		} else {
			repository.run(repository.work, "git", "-c", "user.name=t", "-c", "user.email=t@t", "merge", "-q", "--no-ff", "-m", "Merge pull request #"+strconv.Itoa(len(gh.pulls))+" from o/releaser/release", "FETCH_HEAD")
		}
		gh.pulls[len(gh.pulls)-1]["state"] = "merged"
	}
	openPR := func() map[string]any {
		test.Helper()
		gh.mu.Lock() // the fake's handlers write these maps
		defer gh.mu.Unlock()
		var open []map[string]any
		for _, pr := range gh.pulls {
			if pr["state"] == "open" {
				open = append(open, pr)
			}
		}
		if len(open) != 1 {
			test.Fatalf("want 1 open PR, got %v", gh.pulls)
		}
		return open[0]
	}
	lastRelease := func(tag string, prerelease bool) map[string]any {
		test.Helper()
		gh.mu.Lock() // the fake's handlers write these maps
		defer gh.mu.Unlock()
		rel := gh.releases[len(gh.releases)-1]
		if rel["tag_name"] != tag || rel["prerelease"] != prerelease {
			test.Fatalf("last GitHub release = %v %v, want %s prerelease=%v", rel["tag_name"], rel["prerelease"], tag, prerelease)
		}
		return rel
	}

	// Dry run with the release PR alone reports the stable version it would release.
	repository.commit("feat: hello")
	repository.run(repository.work, "git", "push", "-q", "origin", "main")
	repository.release("-release-pr", "-dry-run")
	if got := readFile(outputs); got != "released=false\nversion=1.0.0\ntag=v1.0.0\nprerelease=false\n" {
		test.Errorf("release PR dry run GITHUB_OUTPUT = %q", got)
	}
	// With canaries on, the canary is what would ship.
	os.Remove(outputs)
	repository.release("-prerelease", "canary", "-release-pr", "-dry-run")
	if got := readFile(outputs); got != "released=false\nversion=1.0.0-canary.1\ntag=v1.0.0-canary.1\nprerelease=true\n" {
		test.Errorf("canary dry run GITHUB_OUTPUT = %q", got)
	}
	if got := repository.remoteTags(); got != "" {
		test.Errorf("dry runs pushed %q", got)
	}

	// feat → first canary, and a PR for 1.0.0 whose commit bumps the files and has no [skip ci].
	push("")
	lastRelease("v1.0.0-canary.1", true)
	if got := readFile(outputs); got != "released=true\nversion=1.0.0-canary.1\ntag=v1.0.0-canary.1\nprerelease=true\n" {
		test.Errorf("canary GITHUB_OUTPUT = %q", got)
	}
	if pr := openPR(); pr["title"] != "chore(release): 1.0.0" || pr["head"] != "releaser/release" || pr["base"] != "main" {
		test.Errorf("PR = %v", pr)
	}
	if msg := repository.run(repository.remote, "git", "log", "-1", "--format=%s", "releaser/release"); strings.TrimSpace(msg) != "chore(release): 1.0.0" {
		test.Errorf("release PR commit = %q", msg)
	}
	if pkg := repository.run(repository.remote, "git", "show", "releaser/release:package.json"); !strings.Contains(pkg, `"version": "1.0.0"`) {
		test.Errorf("PR package.json not bumped:\n%s", pkg)
	}
	// prepare ran for the canary first, then again for the PR: the PR commit has the stable version.
	if got := repository.run(repository.remote, "git", "show", "releaser/release:prepared.txt"); got != "1.0.0\n" {
		test.Errorf("PR prepared.txt = %q", got)
	}

	// fix → next canary of the same base, same PR updated.
	push("fix: typo")
	lastRelease("v1.0.0-canary.2", true)
	if body := openPR()["body"].(string); !strings.Contains(body, "* typo (") || !strings.Contains(body, "* hello (") {
		test.Errorf("PR body not updated:\n%s", body)
	}

	// chore → nothing new since the last canary.
	if out := push("chore: tidy"); !strings.Contains(out, "no release-worthy commits since the last canary") {
		test.Errorf("chore made a canary:\n%s", out)
	}

	// Squash-merge the PR → v1.0.0 on the merged commit, notes straight from its CHANGELOG entry, no canary.
	merge(true)
	head := strings.TrimSpace(repository.run(repository.work, "git", "rev-parse", "HEAD"))
	os.Remove(outputs)
	repository.run(repository.work, "git", "push", "-q", "origin", "main")
	if out := repository.release("-prerelease", "canary", "-release-pr", "-dry-run"); !strings.Contains(out, "merged release PR") {
		test.Errorf("stable dry run:\n%s", out)
	}
	if got := readFile(outputs); got != "released=false\nversion=1.0.0\ntag=v1.0.0\nprerelease=false\n" {
		test.Errorf("stable dry run GITHUB_OUTPUT = %q", got)
	}
	push("")
	rel := lastRelease("v1.0.0", false)
	if got := readFile(outputs); got != "released=true\nversion=1.0.0\ntag=v1.0.0\nprerelease=false\n" {
		test.Errorf("stable GITHUB_OUTPUT = %q", got)
	}
	if got := strings.TrimSpace(repository.run(repository.remote, "git", "rev-list", "-n1", "v1.0.0")); got != head {
		test.Errorf("v1.0.0 on %s, want the squash commit %s", got, head)
	}
	if want := changelog.Latest(repository.read("CHANGELOG.md")); rel["body"] != want || !strings.Contains(want, "* typo (") {
		test.Errorf("release body %q, want CHANGELOG entry %q", rel["body"], want)
	}

	// Re-running on the same commit changes nothing.
	if out := push(""); !strings.Contains(out, "no release-worthy commits since v1.0.0") {
		test.Errorf("rerun:\n%s", out)
	}

	// feat → canary of the next base, a new PR; a merge commit tags the release commit behind it.
	push("feat: greet by name")
	lastRelease("v1.1.0-canary.1", true)
	if pr := openPR(); pr["title"] != "chore(release): 1.1.0" {
		test.Errorf("second PR = %v", pr)
	}
	merge(false)
	push("")
	lastRelease("v1.1.0", false)
	if tagged, second := repository.run(repository.remote, "git", "rev-list", "-n1", "v1.1.0"), repository.run(repository.remote, "git", "rev-parse", "main^2"); tagged != second {
		test.Errorf("v1.1.0 on %s, want the release commit %s", tagged, second)
	}

	push("feat!: new greeting format\n\nBREAKING CHANGE: greetings are now uppercase")
	lastRelease("v2.0.0-canary.1", true)
	if pr := openPR(); pr["title"] != "chore(release): 2.0.0" {
		test.Errorf("third PR = %v", pr)
	}

	// A failing prepare pushes nothing.
	repository.commit("fix: oops")
	repository.run(repository.work, "git", "push", "-q", "origin", "main")
	if out, err := repository.tryRelease("-prerelease", "canary", "-release-pr", "-prepare", "false"); err == nil {
		test.Errorf("failing prepare succeeded:\n%s", out)
	}
	if got := strings.Fields(repository.remoteTags()); strings.Join(got, " ") != "v1.0.0 v1.0.0-canary.1 v1.0.0-canary.2 v1.1.0 v1.1.0-canary.1 v2.0.0-canary.1" {
		test.Errorf("remote tags = %v", got)
	}

	// Main's CHANGELOG has the two stable releases and no canary.
	mainChangelog := repository.run(repository.remote, "git", "show", "main:CHANGELOG.md")
	if !strings.HasPrefix(mainChangelog, "# Changelog\n\n## [1.1.0](https://github.com/o/r/compare/v1.0.0...v1.1.0) (") ||
		!strings.Contains(mainChangelog, "\n## 1.0.0 (") || strings.Contains(mainChangelog, "canary") || strings.Count(mainChangelog, "\n## ") != 2 ||
		!strings.HasSuffix(mainChangelog, ")\n") {
		test.Errorf("main CHANGELOG.md:\n%s", mainChangelog)
	}
}

// Only a release PR merge at HEAD is tagged. One deeper down (its stable run was refused, or other
// commits landed on top) is ignored: the push cuts the next canary and rebuilds the release PR instead.
func TestReleaseCommitAtHead(test *testing.T) {
	for _, testCase := range []struct {
		name  string
		merge bool   // the release commit comes in with a merge commit of releaser/release
		tip   string // commit on top of the release PR merge; "" leaves the merge at HEAD
		args  []string
		want  string // GITHUB_OUTPUT
	}{
		{"squash merge at HEAD", false, "", nil, "released=false\nversion=1.0.1\ntag=v1.0.1\nprerelease=false\n"},
		{"merge commit at HEAD", true, "", nil, "released=false\nversion=1.0.1\ntag=v1.0.1\nprerelease=false\n"},
		{"squash merge below HEAD", false, "fix: tests", nil, "released=false\nversion=1.0.1-canary.2\ntag=v1.0.1-canary.2\nprerelease=true\n"},
		{"merge commit below HEAD", true, "fix: tests", nil, "released=false\nversion=1.0.1-canary.2\ntag=v1.0.1-canary.2\nprerelease=true\n"},
		// The stale release commit alone never bumps, even under a chore rule.
		{"stale release commit, chore rule", false, "docs: readme", []string{"-rules", "chore=patch"}, "released=false\n"},
	} {
		test.Run(testCase.name, func(test *testing.T) {
			repository := newRepo(test, "")
			outputs := filepath.Join(repository.work, "..", "output")
			repository.env = append(repository.env, "GITHUB_REPOSITORY=o/r", "GITHUB_OUTPUT="+outputs)
			repository.commit("feat: first")
			repository.run(repository.work, "git", "tag", "v1.0.0")
			if testCase.args == nil {
				repository.commit("fix: a")
				repository.run(repository.work, "git", "tag", "v1.0.1-canary.1")
				repository.commit("fix: b")
			}
			if testCase.merge {
				repository.run(repository.work, "git", "checkout", "-qb", "releaser/release")
			}
			repository.write("CHANGELOG.md", "# Changelog\n\n## 1.0.1 (2026-01-01)\n\n* b\n")
			repository.run(repository.work, "git", "add", "CHANGELOG.md")
			if testCase.merge {
				repository.commit("chore(release): 1.0.1")
				repository.run(repository.work, "git", "checkout", "-q", "main")
				repository.run(repository.work, "git", "-c", "user.name=t", "-c", "user.email=t@t", "merge", "-q", "--no-ff", "-m", "Merge pull request #1 from o/releaser/release", "releaser/release")
			} else {
				repository.commit("chore(release): 1.0.1 (#1)")
			}
			if testCase.tip != "" {
				repository.commit(testCase.tip)
			}
			out := repository.release(append([]string{"-prerelease", "canary", "-release-pr", "-dry-run"}, testCase.args...)...)
			if got := readFile(outputs); got != testCase.want {
				test.Errorf("GITHUB_OUTPUT = %q, want %q\n%s", got, testCase.want, out)
			}
			if stale := testCase.tip != "" && testCase.args == nil; stale && (strings.Contains(out, "merged release PR") || !strings.Contains(out, "release PR: chore(release): 1.0.1")) {
				test.Errorf("want a rebuilt release PR, no stable release:\n%s", out)
			}
			if strings.Contains(out, "(#1)") {
				test.Errorf("release commit in the notes:\n%s", out)
			}
		})
	}
}
