// Package docker builds and pushes the release image with buildx.
package docker

import (
	"cmp"
	"os"
	"os/exec"
	"strings"
)

// Push builds and pushes image:version. Logs into ghcr.io itself when given a token; other registries
// must be logged in beforehand (e.g. docker/login-action).
func Push(image, platforms, version, repoURL, token string) error {
	if strings.HasPrefix(image, "ghcr.io/") && token != "" {
		login := exec.Command("docker", "login", "ghcr.io", "-u", cmp.Or(os.Getenv("GITHUB_ACTOR"), "x-access-token"), "--password-stdin")
		login.Stdin, login.Stdout, login.Stderr = strings.NewReader(token), os.Stdout, os.Stderr
		if err := login.Run(); err != nil {
			return err
		}
	}
	args := []string{"buildx", "build", "--push", "-t", image + ":" + version, "--label", "org.opencontainers.image.version=" + version}
	if repoURL != "" {
		args = append(args, "--label", "org.opencontainers.image.source="+repoURL)
	}
	if platforms != "" {
		args = append(args, "--platform", platforms)
	}
	return run(append(args, ".")...)
}

// Alias points image:alias (latest, or the prerelease id) at the pushed image:version, in the registry.
func Alias(image, version, alias string) error {
	return run("buildx", "imagetools", "create", "-t", image+":"+alias, image+":"+version)
}

func run(args ...string) error {
	cmd := exec.Command("docker", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}
