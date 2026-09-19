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

// Log returns commits in rev range rng, newest first.
func Log(rng string) ([]RawCommit, error) {
	out, err := Run("log", "--format=%H%x1f%B%x1e", rng)
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
	args := []string{"commit", "-m", message}
	if email, _ := Run("config", "user.email"); email == "" {
		args = append([]string{"-c", "user.name=github-actions[bot]", "-c", "user.email=41898282+github-actions[bot]@users.noreply.github.com"}, args...)
	}
	if _, err := Run(args...); err != nil {
		return err
	}
	if _, err := Run("tag", tag); err != nil {
		return err
	}
	_, err := Run("push", "--atomic", "origin", "HEAD:refs/heads/"+branch, "refs/tags/"+tag)
	return err
}
