package integration

import (
	"cmp"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// Releases examples/docker into a throwaway local registry. Needs a Docker daemon with buildx;
// skipped with -short (the pre-commit hook) or when Docker isn't there.
func TestDockerExample(test *testing.T) {
	if testing.Short() {
		test.Skip("docker example skipped in -short mode")
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		test.Skip("docker not available")
	}

	id := dockerOut(test, "run", "-d", "--rm", "-p", "127.0.0.1::5000", "registry:3")
	test.Cleanup(func() { exec.Command("docker", "rm", "-f", id).Run() })
	_, port, _ := strings.Cut(dockerOut(test, "port", id, "5000"), ":")
	port = strings.Fields(port)[0] // first line if the daemon lists IPv4 and IPv6
	registry := "http://127.0.0.1:" + port
	waitFor(test, registry+"/v2/")

	gh := newFakeGitHub(test)
	repository := newRepo(test, "../../examples/docker")
	repository.env = append(repository.env, gh.env()...)
	// Keep the real docker config: buildx and the daemon context live there.
	repository.env = append(repository.env, "DOCKER_CONFIG="+cmp.Or(os.Getenv("DOCKER_CONFIG"), filepath.Join(os.Getenv("HOME"), ".docker")))
	releaseSteps(test, repository, "-docker", "-docker-image", "localhost:"+port+"/example")
	checkChangelog(test, repository)

	// The GitHub release tells you how to pull the image; the CHANGELOG doesn't.
	pull := "docker pull localhost:" + port + "/example:2.0.0"
	if body := gh.releases[len(gh.releases)-1]["body"].(string); !strings.Contains(body, pull) {
		test.Errorf("release body has no %q:\n%s", pull, body)
	}
	if strings.Contains(repository.read("CHANGELOG.md"), "docker pull") {
		test.Error("docker pull leaked into CHANGELOG.md")
	}

	res, err := http.Get(registry + "/v2/example/tags/list")
	if err != nil {
		test.Fatal(err)
	}
	defer res.Body.Close()
	var list struct{ Tags []string }
	json.NewDecoder(res.Body).Decode(&list)
	slices.Sort(list.Tags)
	if got := strings.Join(list.Tags, " "); got != "1.0.0 1.0.1 1.1.0 2.0.0 latest" {
		test.Errorf("registry tags = %s", got)
	}
}

func dockerOut(test *testing.T, args ...string) string {
	test.Helper()
	var stderr strings.Builder
	cmd := exec.Command("docker", args...)
	cmd.Stderr = &stderr // pull progress, not part of the answer
	out, err := cmd.Output()
	if err != nil {
		test.Fatalf("docker %v: %v\n%s", args, err, stderr.String())
	}
	return strings.TrimSpace(string(out))
}

func waitFor(test *testing.T, url string) {
	test.Helper()
	for range 50 {
		if res, err := http.Get(url); err == nil {
			res.Body.Close()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	test.Fatalf("%s never came up", url)
}
