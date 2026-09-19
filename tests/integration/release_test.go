// Integration: build the real binary and release a throwaway repo into a local bare remote.
package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRelease(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "releaser")
	if out, err := exec.Command("go", "build", "-o", bin, "../../cmd").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	remote, work := filepath.Join(tmp, "remote.git"), filepath.Join(tmp, "work")
	// Isolated env: no user git config, no GitHub context, CI on so it really releases.
	env := []string{"PATH=" + os.Getenv("PATH"), "HOME=" + tmp, "GIT_CONFIG_NOSYSTEM=1", "CI=true"}
	sh := func(dir, name string, args ...string) string {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir, cmd.Env = dir, env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, out)
		}
		return string(out)
	}
	commit := func(msg string) {
		t.Helper()
		sh(work, "git", "-c", "user.name=t", "-c", "user.email=t@t", "commit", "--allow-empty", "-qm", msg)
	}
	remoteTags := func() string { return strings.TrimSpace(sh(remote, "git", "tag", "--list")) }

	sh(tmp, "git", "init", "-q", "--bare", "-b", "main", remote)
	sh(tmp, "git", "init", "-q", "-b", "main", work)
	sh(work, "git", "remote", "add", "origin", remote)
	if err := os.WriteFile(filepath.Join(work, "package.json"), []byte("{\n  \"version\": \"0.0.0\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sh(work, "git", "add", ".")
	commit("feat: first")

	// First release is 1.0.0, prepare sees the version, package.json bumped, commit + tag pushed.
	sh(work, bin, "-prepare", "echo ${version} > prepared.txt")
	if got := remoteTags(); got != "v1.0.0" {
		t.Fatalf("tags after first release = %q", got)
	}
	pkg, _ := os.ReadFile(filepath.Join(work, "package.json"))
	prepared, _ := os.ReadFile(filepath.Join(work, "prepared.txt"))
	changelog, _ := os.ReadFile(filepath.Join(work, "CHANGELOG.md"))
	if !strings.Contains(string(pkg), `"version": "1.0.0"`) || strings.TrimSpace(string(prepared)) != "1.0.0" ||
		!strings.Contains(string(changelog), "* first (") {
		t.Errorf("package.json=%s prepared=%s changelog=%s", pkg, prepared, changelog)
	}
	if msg := sh(remote, "git", "log", "-1", "--format=%s", "main"); strings.TrimSpace(msg) != "chore(release): 1.0.0 [skip ci]" {
		t.Errorf("release commit = %q", msg)
	}

	// The release commit itself (chore) never triggers another release.
	if out := sh(work, bin); !strings.Contains(out, "no release-worthy commits") {
		t.Errorf("chore-only run released:\n%s", out)
	}

	// Dry run touches nothing.
	commit("fix: bug")
	sh(work, bin, "-dry-run")
	if got := remoteTags(); got != "v1.0.0" {
		t.Errorf("dry run pushed: %q", got)
	}

	// feat bumps minor by default, patch with custom rules.
	commit("feat: more")
	sh(work, bin, "-rules", "feat=patch")
	if got := remoteTags(); !strings.HasSuffix(got, "v1.0.1") {
		t.Errorf("custom rules tags = %q", got)
	}
	commit("feat: again")
	sh(work, bin)
	if got := remoteTags(); !strings.HasSuffix(got, "v1.1.0") {
		t.Errorf("default rules tags = %q", got)
	}

	// Wrong branch is a no-op.
	sh(work, "git", "checkout", "-qb", "feature")
	commit("feat!: nope")
	if out := sh(work, bin); !strings.Contains(out, "skipping") {
		t.Errorf("feature branch released:\n%s", out)
	}
}
