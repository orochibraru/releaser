// Integration: build the real binary once and release throwaway repos into local bare remotes.
package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var bin string

// coverDir, when set (by .github/scripts/coverage.sh), gets the coverage of every run of the binary.
var coverDir = os.Getenv("RELEASER_COVERDIR")

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "releaser-bin")
	if err != nil {
		panic(err)
	}
	bin = filepath.Join(dir, "releaser")
	args := []string{"build", "-o", bin}
	if coverDir != "" {
		args = append(args, "-cover", "-coverpkg=../../cmd/...,../../internal/...")
	}
	if out, err := exec.Command("go", append(args, "../../cmd/releaser")...).CombinedOutput(); err != nil {
		panic(fmt.Sprintf("build: %v\n%s", err, out))
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// repo is a work tree on main whose origin is a local bare remote.
type repo struct {
	t            *testing.T
	work, remote string
	env          []string
}

// newRepo copies src (an example dir, or "" for empty) into a fresh repo.
// The env is isolated: no user git config, no GitHub context, CI on so it really releases.
func newRepo(t *testing.T, src string) *repo {
	t.Helper()
	tmp := t.TempDir()
	r := &repo{t: t, work: filepath.Join(tmp, "work"), remote: filepath.Join(tmp, "remote.git"),
		env: []string{"PATH=" + os.Getenv("PATH"), "HOME=" + tmp, "GIT_CONFIG_NOSYSTEM=1", "CI=true", "GOCOVERDIR=" + coverDir}}
	r.run(tmp, "git", "init", "-q", "--bare", "-b", "main", r.remote)
	if src != "" {
		if err := os.CopyFS(r.work, os.DirFS(src)); err != nil {
			t.Fatal(err)
		}
	}
	r.run(tmp, "git", "init", "-q", "-b", "main", r.work)
	r.run(r.work, "git", "remote", "add", "origin", r.remote)
	r.run(r.work, "git", "add", ".")
	return r
}

func (r *repo) run(dir, name string, args ...string) string {
	r.t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir, cmd.Env = dir, r.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
	return string(out)
}

func (r *repo) commit(msg string) {
	r.t.Helper()
	r.run(r.work, "git", "-c", "user.name=t", "-c", "user.email=t@t", "commit", "--allow-empty", "-qm", msg)
}

func (r *repo) release(args ...string) string {
	r.t.Helper()
	return r.run(r.work, bin, args...)
}

// tryRelease is release for runs expected to fail.
func (r *repo) tryRelease(args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	cmd.Dir, cmd.Env = r.work, r.env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (r *repo) remoteTags() string {
	r.t.Helper()
	return strings.TrimSpace(r.run(r.remote, "git", "tag", "--list"))
}

func (r *repo) read(name string) string {
	return readFile(filepath.Join(r.work, name))
}

func (r *repo) write(name, data string) {
	r.t.Helper()
	if err := os.WriteFile(filepath.Join(r.work, name), []byte(data), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func readFile(path string) string {
	b, _ := os.ReadFile(path)
	return string(b)
}
