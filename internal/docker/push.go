// Package docker builds and pushes the release image with buildx.
package docker

import (
	"cmp"
	"os"
	"os/exec"
	"strings"
)

// Push tags image as :version and :latest. Logs into ghcr.io itself when given a token;
// other registries must be logged in beforehand (e.g. docker/login-action).
func Push(image, platforms, version, repoURL, token string) error {
	if strings.HasPrefix(image, "ghcr.io/") && token != "" {
		login := exec.Command("docker", "login", "ghcr.io", "-u", cmp.Or(os.Getenv("GITHUB_ACTOR"), "x-access-token"), "--password-stdin")
		login.Stdin, login.Stdout, login.Stderr = strings.NewReader(token), os.Stdout, os.Stderr
		if err := login.Run(); err != nil {
			return err
		}
	}
	args := []string{"buildx", "build", "--push", "-t", image + ":" + version, "-t", image + ":latest",
		"--label", "org.opencontainers.image.version=" + version}
	if repoURL != "" {
		args = append(args, "--label", "org.opencontainers.image.source="+repoURL)
	}
	if platforms != "" {
		args = append(args, "--platform", platforms)
	}
	cmd := exec.Command("docker", append(args, ".")...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}
