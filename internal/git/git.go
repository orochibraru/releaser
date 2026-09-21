// Package git shells out to the git CLI.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var remoteRe = regexp.MustCompile(`[:/]([^/:]+/[^/]+?)(?:\.git)?/?$`)

// Run executes git and returns trimmed stdout; errors carry stderr.
func Run(args ...string) (string, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

type RawCommit struct{ Hash, Message string }

// Log returns the commits git log selects from revs (e.g. "v1.0.0..HEAD", or "HEAD", "^v1.0.0"), newest first.
func Log(revs ...string) ([]RawCommit, error) {
	out, err := Run(append([]string{"log", "--format=%H%x1f%B%x1e"}, revs...)...)
	if err != nil {
		return nil, err
	}
	var commits []RawCommit
	for _, rec := range strings.Split(out, "\x1e") {
		if hash, msg, ok := strings.Cut(strings.TrimSpace(rec), "\x1f"); ok {
			commits = append(commits, RawCommit{hash, msg})
		}
	}
	return commits, nil
}

// MergedTags lists v* tags reachable from HEAD, unshallowing first if needed (CI clones).
func MergedTags() ([]string, error) {
	if shallow, _ := Run("rev-parse", "--is-shallow-repository"); shallow == "true" {
		if _, err := Run("fetch", "--unshallow", "--tags", "--quiet"); err != nil {
			return nil, err
		}
	}
	out, err := Run("tag", "--merged", "HEAD", "--list", "v*")
	return strings.Fields(out), err
}

// NearestTag returns the closest tag matching glob reachable from HEAD, "" if none.
func NearestTag(glob string) string {
	tag, _ := Run("describe", "--tags", "--abbrev=0", "--match", glob)
	return tag
}

// Slug extracts "owner/repo" from a remote URL (ssh or https).
func Slug(remote string) string {
	if m := remoteRe.FindStringSubmatch(remote); m != nil {
		return m[1]
	}
	return ""
}

// CommitTagPush makes the release commit and pushes it with its tag atomically.
func CommitTagPush(files []string, message, tag, branch string) error {
	if _, err := Run(append([]string{"add", "--"}, files...)...); err != nil {
		return err
	}
	if _, err := Run(asBot("commit", "-m", message)...); err != nil {
		return err
	}
	if _, err := Run("tag", tag); err != nil {
		return err
	}
	_, err := Run("push", "--atomic", "origin", "HEAD:refs/heads/"+branch, "refs/tags/"+tag)
	return err
}

// TagPush tags rev and pushes that tag alone.
func TagPush(tag, rev string) error {
	if _, err := Run("tag", tag, rev); err != nil {
		return err
	}
	_, err := Run("push", "origin", "refs/tags/"+tag)
	return err
}

// CommitPushBranch commits files on top of HEAD without moving the current branch, and force-pushes
// that commit to branch. The working tree keeps the changes, staged.
func CommitPushBranch(files []string, message, branch string) error {
	if _, err := Run(append([]string{"add", "--"}, files...)...); err != nil {
		return err
	}
	tree, err := Run("write-tree")
	if err != nil {
		return err
	}
	sha, err := Run(asBot("commit-tree", tree, "-p", "HEAD", "-m", message)...)
	if err != nil {
		return err
	}
	_, err = Run("push", "-f", "origin", sha+":refs/heads/"+branch)
	return err
}

// asBot commits as github-actions[bot] when no identity is configured.
func asBot(args ...string) []string {
	if email, _ := Run("config", "user.email"); email == "" {
		return append([]string{"-c", "user.name=github-actions[bot]", "-c", "user.email=41898282+github-actions[bot]@users.noreply.github.com"}, args...)
	}
	return args
}
