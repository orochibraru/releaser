// Package semver handles vX.Y.Z tags. Prereleases are deliberately ignored.
package semver

import (
	"fmt"
	"slices"

	"github.com/orochibraru/releaser/internal/conventional"
)

type Version [3]int

var First = Version{1, 0, 0}

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
