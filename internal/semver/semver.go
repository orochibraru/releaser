// Package semver handles vX.Y.Z tags. Prerelease tags never count as the previous release;
// they only number the next prerelease.
package semver

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/orochibraru/releaser/internal/conventional"
)

type Version [3]int

var First = Version{1, 0, 0}

// releaseRe matches a merged release PR: the PR title, with the " (#N)" GitHub adds on squash merges.
var releaseRe = regexp.MustCompile(`^chore\(release\): (\d+\.\d+\.\d+)(?: \(#\d+\))?$`)

func (version Version) String() string {
	return fmt.Sprintf("%d.%d.%d", version[0], version[1], version[2])
}

func (version Version) Tag() string { return "v" + version.String() }

// Latest returns the highest vX.Y.Z tag, skipping anything else.
func Latest(tags []string) (tag string, latest Version, found bool) {
	for _, candidate := range tags {
		var parsed Version
		if _, err := fmt.Sscanf(candidate, "v%d.%d.%d", &parsed[0], &parsed[1], &parsed[2]); err != nil || parsed.Tag() != candidate {
			continue
		}
		if !found || slices.Compare(parsed[:], latest[:]) > 0 {
			tag, latest, found = candidate, parsed, true
		}
	}
	return
}

// Next applies a conventional.Patch/Minor/Major bump.
func (version Version) Next(level int) Version {
	switch level {
	case conventional.Major:
		return Version{version[0] + 1, 0, 0}
	case conventional.Minor:
		return Version{version[0], version[1] + 1, 0}
	}
	return Version{version[0], version[1], version[2] + 1}
}

// Pre returns the next prerelease of v, "X.Y.Z-id.N": N is one above the highest vX.Y.Z-id.N in tags.
func (version Version) Pre(id string, tags []string) string {
	prefix, highest := version.Tag()+"-"+id+".", 0
	for _, tag := range tags {
		if rest, ok := strings.CutPrefix(tag, prefix); ok {
			if number, err := strconv.Atoi(rest); err == nil {
				highest = max(highest, number)
			}
		}
	}
	return fmt.Sprintf("%s-%s.%d", version, id, highest+1)
}

// FromReleaseCommit returns the version of a merged release PR commit, from its subject.
func FromReleaseCommit(message string) (string, bool) {
	subject, _, _ := strings.Cut(message, "\n")
	if match := releaseRe.FindStringSubmatch(strings.TrimSpace(subject)); match != nil {
		return match[1], true
	}
	return "", false
}
