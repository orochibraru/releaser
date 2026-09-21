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

func (v Version) String() string { return fmt.Sprintf("%d.%d.%d", v[0], v[1], v[2]) }

func (v Version) Tag() string { return "v" + v.String() }

// Latest returns the highest vX.Y.Z tag, skipping anything else.
func Latest(tags []string) (tag string, v Version, found bool) {
	for _, t := range tags {
		var c Version
		if _, err := fmt.Sscanf(t, "v%d.%d.%d", &c[0], &c[1], &c[2]); err != nil || c.Tag() != t {
			continue
		}
		if !found || slices.Compare(c[:], v[:]) > 0 {
			tag, v, found = t, c, true
		}
	}
	return
}

// Next applies a conventional.Patch/Minor/Major bump.
func (v Version) Next(level int) Version {
	switch level {
	case conventional.Major:
		return Version{v[0] + 1, 0, 0}
	case conventional.Minor:
		return Version{v[0], v[1] + 1, 0}
	}
	return Version{v[0], v[1], v[2] + 1}
}

// Pre returns the next prerelease of v, "X.Y.Z-id.N": N is one above the highest vX.Y.Z-id.N in tags.
func (v Version) Pre(id string, tags []string) string {
	prefix, n := v.Tag()+"-"+id+".", 0
	for _, t := range tags {
		if rest, ok := strings.CutPrefix(t, prefix); ok {
			if k, err := strconv.Atoi(rest); err == nil {
				n = max(n, k)
			}
		}
	}
	return fmt.Sprintf("%s-%s.%d", v, id, n+1)
}

// FromReleaseCommit returns the version of a merged release PR commit, from its subject.
func FromReleaseCommit(message string) (string, bool) {
	subject, _, _ := strings.Cut(message, "\n")
	if m := releaseRe.FindStringSubmatch(strings.TrimSpace(subject)); m != nil {
		return m[1], true
	}
	return "", false
}
