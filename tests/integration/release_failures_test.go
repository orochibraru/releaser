package integration

import (
	"path/filepath"
	"strings"
	"testing"
)

// Bad input fails before anything is pushed.
func TestBadInput(t *testing.T) {
	for _, c := range []struct {
		name string
		args []string
		gh   bool // run against the fake GitHub (token and repo set)
		want string
	}{
		{"unknown bump level", []string{"-rules", "feat=huge"}, true, `bad rule "feat=huge"`},
		{"prerelease id with a space", []string{"-prerelease", "rc 1"}, true, `bad prerelease id "rc 1"`},
		{"prerelease id with a dot", []string{"-prerelease", "rc.1"}, true, `bad prerelease id "rc.1"`},
		{"artifacts without a token", []string{"-artifacts", "a.txt"}, false, "artifact mode needs GITHUB_TOKEN"},
		{"docker without a repo or image", []string{"-docker"}, false, "docker mode needs -docker-image"},
		{"release PR without a token", []string{"-release-pr"}, false, "release PR mode needs GITHUB_TOKEN"},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newRepo(t, "")
			if c.gh {
				r.env = append(r.env, newFakeGitHub(t).env()...)
			}
			r.commit("feat: first")
			r.run(r.work, "git", "push", "-q", "origin", "main")
			out, err := r.tryRelease(c.args...)
			if err == nil || !strings.Contains(out, c.want) {
				t.Errorf("err=%v, want %q in:\n%s", err, c.want, out)
			}
			if tags := r.remoteTags(); tags != "" {
				t.Errorf("pushed tags %q", tags)
			}
		})
	}
}

// Someone else released first: the atomic push is rejected, so no tag and no GitHub release.
func TestPushRejected(t *testing.T) {
	gh := newFakeGitHub(t)
	r := newRepo(t, "")
	r.env = append(r.env, gh.env()...)
	r.commit("feat: first")
	r.release()

	other := filepath.Join(r.work, "..", "other")
	r.run(r.work, "git", "clone", "-q", r.remote, other)
	r.run(other, "git", "-c", "user.name=t", "-c", "user.email=t@t", "commit", "--allow-empty", "-qm", "fix: elsewhere")
	r.run(other, "git", "push", "-q", "origin", "main")

	r.commit("fix: here")
	out, err := r.tryRelease()
	if err == nil || !strings.Contains(out, "git push") || !strings.Contains(out, "the next run releases it") {
		t.Errorf("rejected push: err=%v\n%s", err, out)
	}
	if got := r.remoteTags(); got != "v1.0.0" {
		t.Errorf("remote tags = %q", got)
	}
	if len(gh.releases) != 1 {
		t.Errorf("GitHub releases after a rejected push: %d", len(gh.releases))
	}
}

// GitHub API errors surface with their status. The release API runs after the push (documented:
// the tag stays, create the release from it); the release PR API after the branch push.
func TestGitHubFailures(t *testing.T) {
	for _, c := range []struct {
		failing string
		args    []string
		want    string
		tags    string
	}{
		{"POST /repos/o/r/releases", nil, "github release: ", "v1.0.0"},
		{"POST /repos/o/r/releases", nil, "create it from the tag (gh release create v1.0.0)", "v1.0.0"},
		{"POST /uploads", []string{"-artifacts", "a.txt"}, "500 Internal Server Error: boom", "v1.0.0"},
		{"GET /repos/o/r/pulls", []string{"-release-pr"}, "release PR: ", ""},
		{"POST /repos/o/r/pulls", []string{"-release-pr"}, "release PR: ", ""},
	} {
		t.Run(c.failing, func(t *testing.T) {
			gh := newFakeGitHub(t)
			gh.failing = c.failing
			r := newRepo(t, "")
			r.env = append(r.env, gh.env()...)
			r.write("a.txt", "a")
			r.commit("feat: first")
			r.run(r.work, "git", "push", "-q", "origin", "main")
			out, err := r.tryRelease(c.args...)
			if err == nil || !strings.Contains(out, c.want) {
				t.Errorf("err=%v, want %q in:\n%s", err, c.want, out)
			}
			if got := r.remoteTags(); got != c.tags {
				t.Errorf("remote tags = %q, want %q", got, c.tags)
			}
		})
	}
}

// Failures before the push leave the remote untouched in every mode: no tag, no release branch.
func TestFailuresPushNothing(t *testing.T) {
	for _, c := range []struct {
		name string
		args []string
		want string
	}{
		// No Dockerfile: the build fails whether or not Docker is installed.
		{"docker build of a canary", []string{"-prerelease", "canary", "-docker", "-docker-image", "localhost:1/x", "-docker-platforms", "linux/amd64"}, "docker: "},
		{"prepare of a canary", []string{"-prerelease", "canary", "-prepare", "exit 3"}, "prepare: exit status 3"},
		{"prepare of the release PR", []string{"-release-pr", "-prepare", "exit 3"}, "prepare: exit status 3"},
		{"missing artifact of a canary", []string{"-prerelease", "canary", "-artifacts", "nope.zip"}, `artifact "nope.zip" matched no files`},
	} {
		t.Run(c.name, func(t *testing.T) {
			gh := newFakeGitHub(t)
			r := newRepo(t, "")
			r.env = append(r.env, gh.env()...)
			r.commit("feat: first")
			r.run(r.work, "git", "push", "-q", "origin", "main")
			out, err := r.tryRelease(c.args...)
			if err == nil || !strings.Contains(out, c.want) {
				t.Errorf("err=%v, want %q in:\n%s", err, c.want, out)
			}
			if got := r.remoteTags(); got != "" {
				t.Errorf("remote tags = %q", got)
			}
			if got := r.run(r.remote, "git", "branch", "--list", "releaser/release"); got != "" {
				t.Errorf("release branch pushed: %q", got)
			}
			if len(gh.releases)+len(gh.pulls) != 0 {
				t.Errorf("GitHub calls: releases=%v pulls=%v", gh.releases, gh.pulls)
			}
		})
	}
}

// A hand-written "chore(release): X.Y.Z" commit without a CHANGELOG.md can't be a release PR:
// fail loudly rather than tag it with empty notes.
func TestReleaseCommitWithoutChangelog(t *testing.T) {
	r := newRepo(t, "")
	r.env = append(r.env, newFakeGitHub(t).env()...)
	r.commit("chore(release): 1.0.0")
	r.run(r.work, "git", "push", "-q", "origin", "main")
	out, err := r.tryRelease("-release-pr")
	if err == nil || !strings.Contains(out, "has no CHANGELOG.md") {
		t.Errorf("err=%v\n%s", err, out)
	}
	if got := r.remoteTags(); got != "" {
		t.Errorf("remote tags = %q", got)
	}
}

// Only a release PR merged into the branch counts as one: not a chore(release) commit brought in by
// another PR's merge commit, and not a version that isn't above the last release.
func TestForgedReleaseCommit(t *testing.T) {
	for _, c := range []struct {
		name, merge string // merge: merge commit subject of a side branch holding the release commit; "" commits it on main
		tag         string // existing release
		want        string
	}{
		{"merged from a contributor's branch", "Merge pull request #5 from o/feature", "", "release PR: chore(release): 1.0.0"},
		{"merged from a fork's releaser/release", "Merge pull request #5 from evil/releaser/release", "", "release PR: chore(release): 1.0.0"},
		{"not above the last release", "", "v2.0.0", "ignoring release commit"},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newRepo(t, "")
			r.env = append(r.env, "GITHUB_REPOSITORY=o/r")
			r.commit("feat: first")
			if c.tag != "" {
				r.run(r.work, "git", "tag", c.tag)
				r.commit("feat: second")
			}
			if c.merge != "" {
				r.run(r.work, "git", "checkout", "-qb", "side")
			}
			r.write("CHANGELOG.md", "# Changelog\n\n## 1.0.0 (2026-01-01)\n\n* forged\n")
			r.run(r.work, "git", "add", "CHANGELOG.md")
			r.commit("chore(release): 1.0.0")
			if c.merge != "" {
				r.commit("feat: side")
				r.run(r.work, "git", "checkout", "-q", "main")
				r.run(r.work, "git", "-c", "user.name=t", "-c", "user.email=t@t", "merge", "-q", "--no-ff", "-m", c.merge, "side")
			}
			out := r.release("-release-pr", "-dry-run")
			if strings.Contains(out, "merged release PR") || !strings.Contains(out, c.want) {
				t.Errorf("want %q, no stable release:\n%s", c.want, out)
			}
		})
	}
}
