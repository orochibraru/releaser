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

func TestMain(suite *testing.M) {
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
	code := suite.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// repo is a work tree on main whose origin is a local bare remote.
type repo struct {
	test         *testing.T
	work, remote string
	env          []string
}

// newRepo copies src (an example dir, or "" for empty) into a fresh repo.
// The env is isolated: no user git config, no GitHub context, CI on so it really releases.
func newRepo(test *testing.T, src string) *repo {
	test.Helper()
	tmp := test.TempDir()
	repository := &repo{test: test, work: filepath.Join(tmp, "work"), remote: filepath.Join(tmp, "remote.git"),
		env: []string{"PATH=" + os.Getenv("PATH"), "HOME=" + tmp, "GIT_CONFIG_NOSYSTEM=1", "CI=true", "GOCOVERDIR=" + coverDir}}
	repository.run(tmp, "git", "init", "-q", "--bare", "-b", "main", repository.remote)
	if src != "" {
		if err := os.CopyFS(repository.work, os.DirFS(src)); err != nil {
			test.Fatal(err)
		}
	}
	repository.run(tmp, "git", "init", "-q", "-b", "main", repository.work)
	repository.run(repository.work, "git", "remote", "add", "origin", repository.remote)
	repository.run(repository.work, "git", "add", ".")
	return repository
}

func (repository *repo) run(dir, name string, args ...string) string {
	repository.test.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir, cmd.Env = dir, repository.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		repository.test.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
	return string(out)
}

func (repository *repo) commit(msg string) {
	repository.test.Helper()
	repository.run(repository.work, "git", "-c", "user.name=t", "-c", "user.email=t@t", "commit", "--allow-empty", "-qm", msg)
}

func (repository *repo) release(args ...string) string {
	repository.test.Helper()
	return repository.run(repository.work, bin, args...)
}

// tryRelease is release for runs expected to fail.
func (repository *repo) tryRelease(args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	cmd.Dir, cmd.Env = repository.work, repository.env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (repository *repo) remoteTags() string {
	repository.test.Helper()
	return strings.TrimSpace(repository.run(repository.remote, "git", "tag", "--list"))
}

func (repository *repo) read(name string) string {
	return readFile(filepath.Join(repository.work, name))
}

func (repository *repo) write(name, data string) {
	repository.test.Helper()
	if err := os.WriteFile(filepath.Join(repository.work, name), []byte(data), 0o644); err != nil {
		repository.test.Fatal(err)
	}
}

func readFile(path string) string {
	data, _ := os.ReadFile(path)
	return string(data)
}
