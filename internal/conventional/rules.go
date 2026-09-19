package conventional

import (
	"fmt"
	"maps"
	"strings"
)

// Bump levels, ordered so max() picks the strongest.
const (
	None = iota
	Patch
	Minor
	Major
)

var levels = map[string]int{"none": None, "patch": Patch, "minor": Minor, "major": Major}

// DefaultRules maps a commit type (or "breaking") to a bump level.
var DefaultRules = map[string]string{"breaking": "major", "feat": "minor", "fix": "patch", "perf": "patch", "revert": "patch"}

// ParseRules merges overrides like "breaking=patch,feat=patch" over DefaultRules.
func ParseRules(overrides []string) (map[string]string, error) {
	rules := maps.Clone(DefaultRules)
	for _, kv := range overrides {
		k, v, _ := strings.Cut(kv, "=")
		if _, ok := levels[v]; !ok {
			return nil, fmt.Errorf("bad rule %q: level must be none, patch, minor or major", kv)
		}
		rules[k] = v
	}
	return rules, nil
}

// Bump returns the strongest level any commit asks for.
func Bump(commits []Commit, rules map[string]string) int {
	lvl := None
	for _, c := range commits {
		lvl = max(lvl, levels[rules[c.Type]])
		if c.Breaking {
			lvl = max(lvl, levels[rules["breaking"]])
		}
	}
	return lvl
}
