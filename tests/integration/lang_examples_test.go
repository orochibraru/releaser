package integration

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// langExamples are released with -prepare "./build.sh ${version}" plus the flags below, as in
// examples/README.md. asset is the uploaded name, %s the version; manifest must end at 2.0.0.
var langExamples = []struct {
	dir, tool, artifacts, commit, asset, manifest string
}{
	{"python", "python3", "dist/*.whl", "pyproject.toml", "hello-%s-py3-none-any.whl", "pyproject.toml"},
	{"go", "go", "dist/*", "", "hello-%s-linux-amd64", ""},
	{"typescript", "npm", "dist/*.tgz", "package-lock.json", "hello-ts-%s.tgz", "package-lock.json"},
	{"javascript", "npm", "dist/*.tgz", "", "hello-js-%s.tgz", "package.json"},
	{"rust", "cargo", "dist/hello.tar.gz=hello-${version}.tar.gz", "Cargo.toml,Cargo.lock", "hello-%s.tar.gz", "Cargo.lock"},
	{"java", "javac", "dist/hello.jar=hello-${version}.jar", "", "hello-%s.jar", ""},
	{"kotlin", "kotlinc", "dist/hello.jar=hello-${version}.jar", "", "hello-%s.jar", ""},
	{"c", "cc", "dist/hello.tar.gz=hello-${version}.tar.gz", "", "hello-%s.tar.gz", ""},
	{"bun", "bun", "dist/hello=hello-${version}", "", "hello-%s", "package.json"},
}

// Real toolchains, some fetching packages: skipped with -short, or per example when its tool is
// missing locally. In CI a missing tool fails instead, so no example silently drops out.
func TestLangExamples(t *testing.T) {
	if testing.Short() {
		t.Skip("language examples skipped in -short mode")
	}
	for _, example := range langExamples {
		t.Run(example.dir, func(t *testing.T) {
			t.Parallel()
			if _, err := exec.LookPath(example.tool); err != nil {
				if os.Getenv("CI") != "" {
					t.Fatal(example.tool, "not installed: CI must run every example")
				}
				t.Skip(example.tool, "not installed")
			}
			gh := newFakeGitHub(t)
			r := newRepo(t, "../../examples/"+example.dir)
			r.env = append(r.env, gh.env()...)
			// The repo's HOME is a temp dir; rustup (CI's cargo) needs its real homes to find a toolchain.
			home, _ := os.UserHomeDir()
			r.env = append(r.env, "RUSTUP_HOME="+cmp.Or(os.Getenv("RUSTUP_HOME"), home+"/.rustup"),
				"CARGO_HOME="+cmp.Or(os.Getenv("CARGO_HOME"), home+"/.cargo"))

			releaseSteps(t, r, "-prepare", "./build.sh ${version}", "-artifacts", example.artifacts, "-commit", example.commit)
			checkChangelog(t, r)

			for _, version := range []string{"1.0.0", "1.0.1", "1.1.0", "2.0.0"} {
				if name := fmt.Sprintf(example.asset, version); len(gh.assets[name]) == 0 {
					t.Errorf("asset %s not uploaded; got %v", name, assetNames(gh.assets))
				}
			}
			if example.manifest != "" && !strings.Contains(r.read(example.manifest), `2.0.0"`) {
				t.Errorf("%s not bumped to 2.0.0:\n%s", example.manifest, r.read(example.manifest))
			}
			// Everything the build touched is either committed or gitignored.
			if status := r.run(r.work, "git", "status", "--porcelain"); status != "" {
				t.Errorf("dirty tree after release:\n%s", status)
			}
		})
	}
}

func assetNames(assets map[string][]byte) (names []string) {
	for name := range assets {
		names = append(names, name)
	}
	return names
}
