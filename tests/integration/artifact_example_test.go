package integration

import (
	"bytes"
	"strings"
	"testing"
)

// Releases examples/artifact through semverSteps with the flags from its README, against a fake GitHub.
func TestArtifactExample(t *testing.T) {
	gh := newFakeGitHub(t)
	r := newRepo(t, "../../examples/artifact")
	r.env = append(r.env, gh.env()...)

	releaseSteps(t, r, "-prepare", "./build.sh ${version}", "-artifacts", "dist/hello.tar.gz=hello-${version}.tar.gz")
	checkChangelog(t, r)

	var tags []string
	for _, rel := range gh.releases {
		tags = append(tags, rel["tag_name"].(string))
	}
	if strings.Join(tags, " ") != "v1.0.0 v1.0.1 v1.1.0 v2.0.0" {
		t.Fatalf("GitHub releases = %v", tags)
	}
	// The GitHub release body is the same text as the CHANGELOG entry.
	if body := gh.releases[3]["body"].(string); !strings.Contains(r.read("CHANGELOG.md"), body) {
		t.Errorf("release notes not in CHANGELOG.md:\n%s", body)
	}
	for _, tag := range tags {
		name := "hello-" + tag[1:] + ".tar.gz"
		if !bytes.HasPrefix(gh.assets[name], []byte{0x1f, 0x8b}) { // gzip magic
			t.Errorf("asset %s missing or not gzip (%d bytes)", name, len(gh.assets[name]))
		}
	}
	if pkg := r.read("package.json"); !strings.Contains(pkg, `"version": "2.0.0"`) {
		t.Errorf("package.json not bumped:\n%s", pkg)
	}

	// A missing artifact fails before anything is pushed.
	r.commit("fix: oops")
	out, err := r.tryRelease("-artifacts", "dist/nope.zip")
	if err == nil || !strings.Contains(out, "matched no files") {
		t.Errorf("missing artifact: err=%v\n%s", err, out)
	}
	if got := r.remoteTags(); strings.Contains(got, "v2.0.1") {
		t.Errorf("failed release pushed a tag: %q", got)
	}
}
