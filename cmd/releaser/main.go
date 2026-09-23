// Command releaser: semantic-release without the plugins.
// Conventional commits in; version, CHANGELOG, tag, GitHub release, artifacts and Docker image out.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type options struct {
	branch, prepare, dockerImage, dockerPlatforms, prerelease string
	rules, commit, artifacts                                  []string
	docker, dryRun, draft, releasePR                          bool
}

func main() {
	var o options
	var rules, commit, artifacts string
	flag.StringVar(&o.branch, "branch", "main", "branch to release from")
	flag.StringVar(&rules, "rules", "", "bump overrides, e.g. breaking=patch,feat=patch,docs=patch")
	flag.StringVar(&o.prepare, "prepare", "", "shell command run before the release commit (${version} is replaced)")
	flag.StringVar(&commit, "commit", "", "extra files for the release commit, comma separated")
	flag.StringVar(&artifacts, "artifacts", "", "artifact mode: files to attach to the GitHub release, path[=name], comma/newline separated, globs ok")
	flag.BoolVar(&o.docker, "docker", false, "docker mode: build and push an image tagged <version> and latest (the prerelease id for prereleases)")
	flag.StringVar(&o.dockerImage, "docker-image", "", "image name (default ghcr.io/<owner>/<repo>)")
	flag.StringVar(&o.dockerPlatforms, "docker-platforms", "", "e.g. linux/amd64,linux/arm64")
	flag.BoolVar(&o.draft, "draft", false, "create the GitHub release as a draft, to publish it yourself once assets are attached")
	flag.StringVar(&o.prerelease, "prerelease", "", "ship every push as a prerelease X.Y.Z-<id>.N (e.g. canary): tag and GitHub prerelease, no commit")
	flag.BoolVar(&o.releasePR, "release-pr", false, "open a release PR instead of committing; merging it tags the release")
	flag.BoolVar(&o.dryRun, "dry-run", os.Getenv("CI") == "", "only print the next release (default outside CI)")
	flag.Parse()
	o.rules, o.commit, o.artifacts = split(rules), split(commit), split(artifacts)

	if err := release(o); err != nil {
		fmt.Fprintln(os.Stderr, "releaser:", err)
		os.Exit(1)
	}
}

func split(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '\n' })
}
