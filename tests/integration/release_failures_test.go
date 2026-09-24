package integration

import (
	"path/filepath"
	"strings"
	"testing"
)

// Bad input fails before anything is pushed.
func TestBadInput(test *testing.T) {
	for _, testCase := range []struct {
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
		test.Run(testCase.name, func(test *testing.T) {
			repository := newRepo(test, "")
			if testCase.gh {
				repository.env = append(repository.env, newFakeGitHub(test).env()...)
			}
			repository.commit("feat: first")
			repository.run(repository.work, "git", "push", "-q", "origin", "main")
			out, err := repository.tryRelease(testCase.args...)
			if err == nil || !strings.Contains(out, testCase.want) {
				test.Errorf("err=%v, want %q in:\n%s", err, testCase.want, out)
			}
			if tags := repository.remoteTags(); tags != "" {
				test.Errorf("pushed tags %q", tags)
			}
		})
	}
}

// Someone else released first: the atomic push is rejected, so no tag and no GitHub release.
func TestPushRejected(test *testing.T) {
	gh := newFakeGitHub(test)
	repository := newRepo(test, "")
	repository.env = append(repository.env, gh.env()...)
	repository.commit("feat: first")
	repository.release()

	other := filepath.Join(repository.work, "..", "other")
	repository.run(repository.work, "git", "clone", "-q", repository.remote, other)
	repository.run(other, "git", "-c", "user.name=t", "-c", "user.email=t@t", "commit", "--allow-empty", "-qm", "fix: elsewhere")
	repository.run(other, "git", "push", "-q", "origin", "main")

	repository.commit("fix: here")
	out, err := repository.tryRelease()
	if err == nil || !strings.Contains(out, "git push") || !strings.Contains(out, "the next run releases it") {
		test.Errorf("rejected push: err=%v\n%s", err, out)
	}
	if got := repository.remoteTags(); got != "v1.0.0" {
		test.Errorf("remote tags = %q", got)
	}
	if len(gh.releases) != 1 {
		test.Errorf("GitHub releases after a rejected push: %d", len(gh.releases))
	}
}

// GitHub API errors surface with their status. The release API runs after the push (documented:
// the tag stays, create the release from it); the release PR API after the branch push.
func TestGitHubFailures(test *testing.T) {
	for _, testCase := range []struct {
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
		test.Run(testCase.failing, func(test *testing.T) {
			gh := newFakeGitHub(test)
			gh.failing = testCase.failing
			repository := newRepo(test, "")
			repository.env = append(repository.env, gh.env()...)
			repository.write("a.txt", "a")
			repository.commit("feat: first")
			repository.run(repository.work, "git", "push", "-q", "origin", "main")
			out, err := repository.tryRelease(testCase.args...)
			if err == nil || !strings.Contains(out, testCase.want) {
				test.Errorf("err=%v, want %q in:\n%s", err, testCase.want, out)
			}
			if got := repository.remoteTags(); got != testCase.tags {
				test.Errorf("remote tags = %q, want %q", got, testCase.tags)
			}
		})
	}
}

// Failures before the push leave the remote untouched in every mode: no tag, no release branch.
func TestFailuresPushNothing(test *testing.T) {
	for _, testCase := range []struct {
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
		test.Run(testCase.name, func(test *testing.T) {
			gh := newFakeGitHub(test)
			repository := newRepo(test, "")
			repository.env = append(repository.env, gh.env()...)
			repository.commit("feat: first")
			repository.run(repository.work, "git", "push", "-q", "origin", "main")
			out, err := repository.tryRelease(testCase.args...)
			if err == nil || !strings.Contains(out, testCase.want) {
				test.Errorf("err=%v, want %q in:\n%s", err, testCase.want, out)
			}
			if got := repository.remoteTags(); got != "" {
				test.Errorf("remote tags = %q", got)
			}
			if got := repository.run(repository.remote, "git", "branch", "--list", "releaser/release"); got != "" {
				test.Errorf("release branch pushed: %q", got)
			}
			if len(gh.releases)+len(gh.pulls) != 0 {
				test.Errorf("GitHub calls: releases=%v pulls=%v", gh.releases, gh.pulls)
			}
		})
	}
}

// A hand-written "chore(release): X.Y.Z" commit without a CHANGELOG.md can't be a release PR:
// fail loudly rather than tag it with empty notes.
func TestReleaseCommitWithoutChangelog(test *testing.T) {
	repository := newRepo(test, "")
	repository.env = append(repository.env, newFakeGitHub(test).env()...)
	repository.commit("chore(release): 1.0.0")
	repository.run(repository.work, "git", "push", "-q", "origin", "main")
	out, err := repository.tryRelease("-release-pr")
	if err == nil || !strings.Contains(out, "has no CHANGELOG.md") {
		test.Errorf("err=%v\n%s", err, out)
	}
	if got := repository.remoteTags(); got != "" {
		test.Errorf("remote tags = %q", got)
	}
}

// Only a release PR merged into the branch counts as one: not a chore(release) commit brought in by
// another PR's merge commit, and not a version that isn't above the last release.
func TestForgedReleaseCommit(test *testing.T) {
	for _, testCase := range []struct {
		name, merge string // merge: merge commit subject of a side branch holding the release commit; "" commits it on main
		tag         string // existing release
		want        string
	}{
		{"merged from a contributor's branch", "Merge pull request #5 from o/feature", "", "release PR: chore(release): 1.0.0"},
		{"merged from a fork's releaser/release", "Merge pull request #5 from evil/releaser/release", "", "release PR: chore(release): 1.0.0"},
		{"not above the last release", "", "v2.0.0", "ignoring release commit"},
	} {
		test.Run(testCase.name, func(test *testing.T) {
			repository := newRepo(test, "")
			repository.env = append(repository.env, "GITHUB_REPOSITORY=o/r")
			repository.commit("feat: first")
			if testCase.tag != "" {
				repository.run(repository.work, "git", "tag", testCase.tag)
				repository.commit("feat: second")
			}
			if testCase.merge != "" {
				repository.run(repository.work, "git", "checkout", "-qb", "side")
			}
			repository.write("CHANGELOG.md", "# Changelog\n\n## 1.0.0 (2026-01-01)\n\n* forged\n")
			repository.run(repository.work, "git", "add", "CHANGELOG.md")
			repository.commit("chore(release): 1.0.0")
			if testCase.merge != "" {
				repository.commit("feat: side")
				repository.run(repository.work, "git", "checkout", "-q", "main")
				repository.run(repository.work, "git", "-c", "user.name=t", "-c", "user.email=t@t", "merge", "-q", "--no-ff", "-m", testCase.merge, "side")
			}
			out := repository.release("-release-pr", "-dry-run")
			if strings.Contains(out, "merged release PR") || !strings.Contains(out, testCase.want) {
				test.Errorf("want %q, no stable release:\n%s", testCase.want, out)
			}
		})
	}
}
