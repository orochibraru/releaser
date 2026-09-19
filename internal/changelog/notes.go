// Package changelog renders release notes (conventionalcommits preset style) and maintains CHANGELOG.md.
package changelog

import (
	"fmt"
	"strings"
	"time"

	"github.com/orochibraru/releaser/internal/conventional"
)

var sections = []struct{ typ, title string }{
	{"feat", "Features"}, {"fix", "Bug Fixes"}, {"perf", "Performance Improvements"},
	{"revert", "Reverts"}, {"docs", "Documentation"}, {"refactor", "Code Refactoring"},
}

// Notes renders one release. repoURL may be empty (no links); prevTag empty means first release.
func Notes(repoURL, prevTag, tag string, date time.Time, commits []conventional.Commit) string {
	var b strings.Builder
	version, day := strings.TrimPrefix(tag, "v"), date.Format("2006-01-02")
	if repoURL != "" && prevTag != "" {
		fmt.Fprintf(&b, "## [%s](%s/compare/%s...%s) (%s)\n", version, repoURL, prevTag, tag, day)
	} else {
		fmt.Fprintf(&b, "## %s (%s)\n", version, day)
	}
	item := func(c conventional.Commit, text string) string {
		s := "* "
		if c.Scope != "" {
			s += "**" + c.Scope + ":** "
		}
		short := c.Hash[:min(7, len(c.Hash))]
		if repoURL != "" {
			return s + text + fmt.Sprintf(" ([%s](%s/commit/%s))", short, repoURL, c.Hash)
		}
		return s + text + " (" + short + ")"
	}
	section := func(title string, items []string) {
		if len(items) > 0 {
			fmt.Fprintf(&b, "\n### %s\n\n%s\n", title, strings.Join(items, "\n"))
		}
	}

	var breaking []string
	for _, c := range commits {
		if c.Breaking {
			breaking = append(breaking, item(c, c.BreakingNote))
		}
	}
	section("⚠ BREAKING CHANGES", breaking)
	for _, s := range sections {
		var items []string
		for _, c := range commits {
			if c.Type == s.typ {
				items = append(items, item(c, c.Subject))
			}
		}
		section(s.title, items)
	}
	return b.String()
}
