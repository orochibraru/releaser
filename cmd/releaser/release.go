package main

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/orochibraru/releaser/internal/artifacts"
	"github.com/orochibraru/releaser/internal/changelog"
	"github.com/orochibraru/releaser/internal/conventional"
	"github.com/orochibraru/releaser/internal/docker"
	"github.com/orochibraru/releaser/internal/git"
	"github.com/orochibraru/releaser/internal/github"
	"github.com/orochibraru/releaser/internal/npm"
	"github.com/orochibraru/releaser/internal/semver"
)

// releaseBranch is the head branch of the release PR.
const releaseBranch = "releaser/release"

var prereleaseRe = regexp.MustCompile(`^[0-9A-Za-z-]+$`)

// mergeRe matches GitHub's merge commit subject for a PR; the group is the head branch's owner.
var mergeRe = regexp.MustCompile(`^Merge pull request #\d+ from ([^/\s]+)/` + regexp.QuoteMeta(releaseBranch) + `$`)

type run struct {
	flags                options
	repo, repoURL, token string
}

// release picks the phase for this push:
//   - with -release-pr, a merged release PR at HEAD → tag its release commit (stable);
//   - otherwise with -prerelease → tag HEAD as the next prerelease, and/or with -release-pr → update the PR;
//   - neither flag → release commit and tag, pushed together (direct).
//
// Every phase runs everything that can fail before its one push, so a failed run leaves the remote untouched.
func release(flags options) error {
	rules, err := conventional.ParseRules(flags.rules)
	if err != nil {
		return err
	}
	if flags.prerelease != "" && !prereleaseRe.MatchString(flags.prerelease) {
		return fmt.Errorf("bad prerelease id %q: letters, digits and hyphens only", flags.prerelease)
	}

	branch := os.Getenv("GITHUB_REF_NAME")
	if branch == "" {
		branch, _ = git.Run("rev-parse", "--abbrev-ref", "HEAD")
	}
	if branch != flags.branch {
		fmt.Printf("on %q, releases only happen on %q: skipping\n", branch, flags.branch)
		return github.SetOutput("released=false")
	}

	releaser := run{flags: flags, token: cmp.Or(os.Getenv("GITHUB_TOKEN"), os.Getenv("GH_TOKEN")), repo: os.Getenv("GITHUB_REPOSITORY")}
	if releaser.repo == "" {
		remote, _ := git.Run("remote", "get-url", "origin")
		releaser.repo = git.Slug(remote)
	}
	if releaser.repo != "" {
		releaser.repoURL = cmp.Or(os.Getenv("GITHUB_SERVER_URL"), "https://github.com") + "/" + releaser.repo
	}
	if !flags.dryRun {
		if len(flags.artifacts) > 0 && releaser.token == "" {
			return fmt.Errorf("artifact mode needs GITHUB_TOKEN")
		}
		if flags.docker && releaser.repo == "" && flags.dockerImage == "" {
			return fmt.Errorf("docker mode needs -docker-image (no GitHub remote found)")
		}
		if flags.releasePR && (releaser.token == "" || releaser.repo == "") {
			return fmt.Errorf("release PR mode needs GITHUB_TOKEN and a GitHub remote")
		}
	}

	// Commits since the last release.
	tags, err := git.MergedTags()
	if err != nil {
		return err
	}
	prevTag, prev, found := semver.Latest(tags)
	since := []string{"HEAD"}
	if found {
		since = append(since, "^"+prevTag)
	}
	raw, err := git.Log(since...)
	if err != nil {
		return err
	}

	if flags.releasePR {
		version, sha, err := releaser.releaseCommit(since, prev, found)
		if err != nil {
			return err
		}
		if version != "" {
			return releaser.stable(version, sha)
		}
	}

	commits := parse(raw)
	level := conventional.Bump(commits, rules)
	if level == conventional.None {
		fmt.Println("no release-worthy commits since", cmp.Or(prevTag, "the beginning"))
		return github.SetOutput("released=false")
	}
	next := semver.First
	if found {
		next = prev.Next(level)
	}
	notes := func(tag string) string {
		return changelog.Notes(releaser.repoURL, prevTag, tag, time.Now().UTC(), commits)
	}

	if flags.prerelease == "" && !flags.releasePR {
		return releaser.direct(next.String(), notes(next.Tag()))
	}
	outputs := []string{"released=false"}
	if flags.dryRun { // what the release PR would release; a pending canary below takes over
		outputs = []string{"released=false", "version=" + next.String(), "tag=" + next.Tag(), "prerelease=false"}
	}
	if flags.prerelease != "" {
		// Only what landed since the last prerelease can make a new one.
		fresh := since
		if last := git.NearestTag("v*-" + flags.prerelease + ".*"); last != "" {
			fresh = append(fresh, "^"+last)
		}
		newer, err := git.Log(fresh...)
		if err != nil {
			return err
		}
		if conventional.Bump(parse(newer), rules) == conventional.None {
			fmt.Println("no release-worthy commits since the last", flags.prerelease, "prerelease")
		} else {
			version := next.Pre(flags.prerelease, tags)
			tag := "v" + version
			if flags.dryRun {
				fmt.Printf("next prerelease: %s\n\n%s\n", tag, notes(tag))
				outputs = []string{"released=false", "version=" + version, "tag=" + tag, "prerelease=true"}
			} else {
				if err := releaser.publish(version, notes(tag), true, func() error { return git.TagPush(tag, "HEAD") }); err != nil {
					return err
				}
				outputs = []string{"released=true", "version=" + version, "tag=" + tag, "prerelease=true"}
			}
		}
	}
	if flags.releasePR {
		if err := releaser.releasePR(next.String(), notes(next.Tag())); err != nil {
			return err
		}
	}
	if flags.dryRun {
		fmt.Println("dry run: nothing written (pass -dry-run=false or set CI to release)")
	}
	return github.SetOutput(outputs...)
}

// direct makes the release commit and pushes it with its tag.
func (releaser run) direct(version, notes string) error {
	tag := "v" + version
	fmt.Printf("next release: %s\n\n%s\n", tag, notes)
	if releaser.flags.dryRun {
		fmt.Println("dry run: nothing written (pass -dry-run=false or set CI to release)")
		return github.SetOutput("released=false", "version="+version, "tag="+tag, "prerelease=false")
	}
	files, err := bumpFiles(version, notes)
	if err != nil {
		return err
	}
	push := func() error {
		return git.CommitTagPush(append(files, releaser.flags.commit...), "chore(release): "+version+" [skip ci]\n\n"+notes, tag, releaser.flags.branch)
	}
	if err := releaser.publish(version, notes, false, push); err != nil {
		return err
	}
	return github.SetOutput("released=true", "version="+version, "tag="+tag, "prerelease=false")
}

// releaseCommit finds a merged release PR at HEAD: HEAD is a chore(release): X.Y.Z commit (squash or
// rebase merge), or a merge commit of this repo's releaseBranch with that commit behind it. Only HEAD
// counts: a release commit deeper down was never tagged (its run failed or was refused), and taking it
// again on every push would block canaries and the release PR for good. Commits that came in with any
// other PR never count, nor versions not above the last release: either would let a contributor pick
// the version and the tagged tree.
func (releaser run) releaseCommit(since []string, prev semver.Version, found bool) (version, sha string, err error) {
	raw, err := git.Log(append([]string{"-1", "--first-parent"}, since...)...)
	if err != nil || len(raw) == 0 {
		return "", "", err
	}
	commit := raw[0]
	owner, _, _ := strings.Cut(releaser.repo, "/")
	subject, _, _ := strings.Cut(commit.Message, "\n")
	if match := mergeRe.FindStringSubmatch(strings.TrimSpace(subject)); match != nil && match[1] == owner {
		if commit.Hash, err = git.Run("rev-parse", commit.Hash+"^2"); err != nil {
			return "", "", err
		}
		if commit.Message, err = git.Run("log", "-1", "--format=%B", commit.Hash); err != nil {
			return "", "", err
		}
	}
	candidate, ok := semver.FromReleaseCommit(commit.Message)
	if !ok {
		return "", "", nil
	}
	if _, next, _ := semver.Latest([]string{"v" + candidate}); found && slices.Compare(next[:], prev[:]) <= 0 {
		fmt.Printf("ignoring release commit %s: %s is not above the last release %s\n", commit.Hash[:min(7, len(commit.Hash))], candidate, prev)
		return "", "", nil
	}
	return candidate, commit.Hash, nil
}

// stable tags a merged release PR commit. The notes are its CHANGELOG.md entry.
func (releaser run) stable(version, sha string) error {
	tag := "v" + version
	log, err := git.Run("show", sha+":CHANGELOG.md")
	if err != nil {
		return fmt.Errorf("release commit %s has no CHANGELOG.md: %w", sha[:min(7, len(sha))], err)
	}
	notes := changelog.Latest(log + "\n")
	fmt.Printf("merged release PR %s: releasing %s\n\n%s\n", sha[:min(7, len(sha))], tag, notes)
	if releaser.flags.dryRun {
		fmt.Println("dry run: nothing written (pass -dry-run=false or set CI to release)")
		return github.SetOutput("released=false", "version="+version, "tag="+tag, "prerelease=false")
	}
	if err := releaser.publish(version, notes, false, func() error { return git.TagPush(tag, sha) }); err != nil {
		return err
	}
	return github.SetOutput("released=true", "version="+version, "tag="+tag, "prerelease=false")
}

// releasePR force-pushes the release commit to releaseBranch and opens or updates its PR.
// No [skip ci]: merging the PR must run releaser, which then finds the commit and tags it.
func (releaser run) releasePR(version, notes string) error {
	title := "chore(release): " + version
	fmt.Printf("release PR: %s\n\n%s\n", title, notes)
	if releaser.flags.dryRun {
		return nil
	}
	files, err := bumpFiles(version, notes)
	if err != nil {
		return err
	}
	if err := releaser.prepare(version); err != nil {
		return err
	}
	if err := git.CommitPushBranch(append(files, releaser.flags.commit...), title+"\n\n"+notes, releaseBranch); err != nil {
		return err
	}
	url, err := github.UpsertPR(releaser.repo, releaser.token, releaseBranch, releaser.flags.branch, title, notes)
	if err != nil {
		return fmt.Errorf("release PR: %w", err)
	}
	fmt.Println("release PR", url)
	return nil
}

// publish runs prepare, resolves artifacts and pushes the Docker image as :version, then push (the git
// push), then creates the GitHub release and moves the image's alias (:latest or the prerelease id),
// so a rejected push never moves the alias.
func (releaser run) publish(version, notes string, prerelease bool, push func() error) error {
	tag := "v" + version
	if err := releaser.prepare(version); err != nil {
		return err
	}
	assets, err := artifacts.Resolve(releaser.flags.artifacts, version)
	if err != nil {
		return err
	}
	body := notes // GitHub release body; CHANGELOG.md stays the plain notes
	image := cmp.Or(releaser.flags.dockerImage, "ghcr.io/"+strings.ToLower(releaser.repo))
	if releaser.flags.docker {
		if err := docker.Push(image, releaser.flags.dockerPlatforms, version, releaser.repoURL, releaser.token); err != nil {
			return fmt.Errorf("docker: %w", err)
		}
		body += "\n### Docker\n\n```sh\ndocker pull " + image + ":" + version + "\n```\n"
	}

	if err := push(); err != nil {
		return fmt.Errorf("%w\nnothing released: if %s moved since this run started, the next run releases it", err, releaser.flags.branch)
	}
	fmt.Println("pushed", tag)
	if releaser.token == "" || releaser.repo == "" {
		fmt.Println("no GITHUB_TOKEN or GitHub remote: skipping GitHub release")
	} else if err := github.CreateRelease(releaser.repo, releaser.token, tag, body, assets, releaser.flags.draft, prerelease); err != nil {
		return fmt.Errorf("github release: %w\n%s is pushed but has no GitHub release, and rerunning won't retry it: create it from the tag (gh release create %s)", err, tag, tag)
	}
	if releaser.flags.docker {
		alias := "latest"
		if prerelease {
			alias = releaser.flags.prerelease
		}
		if err := docker.Alias(image, version, alias); err != nil {
			return fmt.Errorf("docker: %w\n%s is released but :%s didn't move: docker buildx imagetools create -t %s:%s %s:%s", err, tag, alias, image, alias, image, version)
		}
	}
	return nil
}

func (releaser run) prepare(version string) error {
	if releaser.flags.prepare == "" {
		return nil
	}
	cmd := exec.Command("sh", "-c", strings.ReplaceAll(releaser.flags.prepare, "${version}", version))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	// prepare runs build tools and their dependencies: keep the release token away from them.
	cmd.Env = slices.DeleteFunc(os.Environ(), func(kv string) bool {
		return strings.HasPrefix(kv, "GITHUB_TOKEN=") || strings.HasPrefix(kv, "GH_TOKEN=")
	})
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	return nil
}

// bumpFiles writes the release into CHANGELOG.md and package.json and returns the files to commit.
func bumpFiles(version, notes string) ([]string, error) {
	if err := changelog.PrependFile("CHANGELOG.md", notes); err != nil {
		return nil, err
	}
	files := []string{"CHANGELOG.md"}
	if bumped, err := npm.SetVersionFile("package.json", version); err != nil {
		return nil, err
	} else if bumped {
		files = append(files, "package.json")
	}
	return files, nil
}

func parse(raw []git.RawCommit) []conventional.Commit {
	var commits []conventional.Commit
	for _, rawCommit := range raw {
		if _, ok := semver.FromReleaseCommit(rawCommit.Message); ok { // an untagged release PR merge, even under a chore rule
			continue
		}
		if commit, ok := conventional.Parse(rawCommit.Hash, rawCommit.Message); ok {
			commits = append(commits, commit)
		}
	}
	return commits
}
