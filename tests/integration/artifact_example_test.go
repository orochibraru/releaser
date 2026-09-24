package integration

import (
	"bytes"
	"strings"
	"testing"
)

// Releases examples/artifact through semverSteps with the flags from its README, against a fake GitHub.
func TestArtifactExample(test *testing.T) {
	gh := newFakeGitHub(test)
	repository := newRepo(test, "../../examples/artifact")
	repository.env = append(repository.env, gh.env()...)

	releaseSteps(test, repository, "-prepare", "./build.sh ${version}", "-artifacts", "dist/hello.tar.gz=hello-${version}.tar.gz")
	checkChangelog(test, repository)

	var tags []string
	for _, rel := range gh.releases {
		tags = append(tags, rel["tag_name"].(string))
	}
	if strings.Join(tags, " ") != "v1.0.0 v1.0.1 v1.1.0 v2.0.0" {
		test.Fatalf("GitHub releases = %v", tags)
	}
	// The GitHub release body is the same text as the CHANGELOG entry.
	if body := gh.releases[3]["body"].(string); !strings.Contains(repository.read("CHANGELOG.md"), body) {
		test.Errorf("release notes not in CHANGELOG.md:\n%s", body)
	}
	for _, tag := range tags {
		name := "hello-" + tag[1:] + ".tar.gz"
		if !bytes.HasPrefix(gh.assets[name], []byte{0x1f, 0x8b}) { // gzip magic
			test.Errorf("asset %s missing or not gzip (%d bytes)", name, len(gh.assets[name]))
		}
	}
	if pkg := repository.read("package.json"); !strings.Contains(pkg, `"version": "2.0.0"`) {
		test.Errorf("package.json not bumped:\n%s", pkg)
	}

	// A missing artifact fails before anything is pushed.
	repository.commit("fix: oops")
	out, err := repository.tryRelease("-artifacts", "dist/nope.zip")
	if err == nil || !strings.Contains(out, "matched no files") {
		test.Errorf("missing artifact: err=%v\n%s", err, out)
	}
	if got := repository.remoteTags(); strings.Contains(got, "v2.0.1") {
		test.Errorf("failed release pushed a tag: %q", got)
	}
}
