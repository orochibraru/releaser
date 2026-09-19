package integration

import (
	"fmt"
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

// Real toolchains, some fetching packages: skipped with -short, or per example when its tool is missing.
func TestLangExamples(t *testing.T) {
	if testing.Short() {
		t.Skip("language examples skipped in -short mode")
	}
	for _, ex := range langExamples {
		t.Run(ex.dir, func(t *testing.T) {
			t.Parallel()
			if _, err := exec.LookPath(ex.tool); err != nil {
				t.Skip(ex.tool, "not installed")
			}
			gh := newFakeGitHub(t)
			r := newRepo(t, "../../examples/"+ex.dir)
			r.env = append(r.env, gh.env()...)

			releaseSteps(t, r, "-prepare", "./build.sh ${version}", "-artifacts", ex.artifacts, "-commit", ex.commit)
			checkChangelog(t, r)

			for _, v := range []string{"1.0.0", "1.0.1", "1.1.0", "2.0.0"} {
				if name := fmt.Sprintf(ex.asset, v); len(gh.assets[name]) == 0 {
					t.Errorf("asset %s not uploaded; got %v", name, keys(gh.assets))
				}
			}
			if ex.manifest != "" && !strings.Contains(r.read(ex.manifest), `2.0.0"`) {
				t.Errorf("%s not bumped to 2.0.0:\n%s", ex.manifest, r.read(ex.manifest))
			}
			// Everything the build touched is either committed or gitignored.
			if st := r.run(r.work, "git", "status", "--porcelain"); st != "" {
				t.Errorf("dirty tree after release:\n%s", st)
			}
		})
	}
}

func keys(m map[string][]byte) (ks []string) {
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
