package main

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
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

// release runs the whole pipeline. Everything that can fail runs before the git push,
// so a failed release leaves the remote untouched.
func release(o options) error {
	rules, err := conventional.ParseRules(o.rules)
	if err != nil {
		return err
	}

	branch := os.Getenv("GITHUB_REF_NAME")
	if branch == "" {
		branch, _ = git.Run("rev-parse", "--abbrev-ref", "HEAD")
	}
	if branch != o.branch {
		fmt.Printf("on %q, releases only happen on %q: skipping\n", branch, o.branch)
		return github.SetOutput("released=false")
	}

	// Work out the next version.
	tags, err := git.MergedTags()
	if err != nil {
		return err
	}
	prevTag, prev, found := semver.Latest(tags)
	rng := "HEAD"
	if found {
		rng = prevTag + "..HEAD"
	}
	raw, err := git.Log(rng)
	if err != nil {
		return err
	}
	var commits []conventional.Commit
	for _, r := range raw {
		if c, ok := conventional.Parse(r.Hash, r.Message); ok {
			commits = append(commits, c)
		}
	}
	level := conventional.Bump(commits, rules)
	if level == conventional.None {
		fmt.Println("no release-worthy commits since", cmp.Or(prevTag, "the beginning"))
		return github.SetOutput("released=false")
	}
	next := semver.First
	if found {
		next = prev.Next(level)
	}
	version, tag := next.String(), next.Tag()

	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		remote, _ := git.Run("remote", "get-url", "origin")
		repo = git.Slug(remote)
	}
	repoURL := ""
	if repo != "" {
		repoURL = cmp.Or(os.Getenv("GITHUB_SERVER_URL"), "https://github.com") + "/" + repo
	}
	notes := changelog.Notes(repoURL, prevTag, tag, time.Now().UTC(), commits)
	fmt.Printf("next release: %s\n\n%s\n", tag, notes)
	if o.dryRun {
		fmt.Println("dry run: nothing written (pass -dry-run=false or set CI to release)")
		return nil
	}

	token := cmp.Or(os.Getenv("GITHUB_TOKEN"), os.Getenv("GH_TOKEN"))
	if len(o.artifacts) > 0 && token == "" {
		return fmt.Errorf("artifact mode needs GITHUB_TOKEN")
	}
	if o.docker && repo == "" && o.dockerImage == "" {
		return fmt.Errorf("docker mode needs -docker-image (no GitHub remote found)")
	}

	// Prepare the release commit.
	if err := changelog.PrependFile("CHANGELOG.md", notes); err != nil {
		return err
	}
	files := []string{"CHANGELOG.md"}
	if bumped, err := npm.SetVersionFile("package.json", version); err != nil {
		return err
	} else if bumped {
		files = append(files, "package.json")
	}
	if o.prepare != "" {
		cmd := exec.Command("sh", "-c", strings.ReplaceAll(o.prepare, "${version}", version))
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("prepare: %w", err)
		}
	}
	assets, err := artifacts.Resolve(o.artifacts, version)
	if err != nil {
		return err
	}
	if o.docker {
		image := cmp.Or(o.dockerImage, "ghcr.io/"+strings.ToLower(repo))
		if err := docker.Push(image, o.dockerPlatforms, version, repoURL, token); err != nil {
			return fmt.Errorf("docker: %w", err)
		}
	}

	// Publish.
	message := "chore(release): " + version + " [skip ci]\n\n" + notes
	if err := git.CommitTagPush(append(files, o.commit...), message, tag, o.branch); err != nil {
		return err
	}
	fmt.Println("pushed", tag)
	if token == "" || repo == "" {
		fmt.Println("no GITHUB_TOKEN or GitHub remote: skipping GitHub release")
	} else if err := github.CreateRelease(repo, token, tag, notes, assets); err != nil {
		return fmt.Errorf("github release: %w", err)
	}
	return github.SetOutput("released=true", "version="+version, "tag="+tag)
}
