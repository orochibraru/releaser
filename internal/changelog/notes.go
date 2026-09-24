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
	var builder strings.Builder
	version, day := strings.TrimPrefix(tag, "v"), date.Format("2006-01-02")
	if repoURL != "" && prevTag != "" {
		fmt.Fprintf(&builder, "## [%s](%s/compare/%s...%s) (%s)\n", version, repoURL, prevTag, tag, day)
	} else {
		fmt.Fprintf(&builder, "## %s (%s)\n", version, day)
	}
	item := func(commit conventional.Commit, text string) string {
		line := "* "
		if commit.Scope != "" {
			line += "**" + commit.Scope + ":** "
		}
		short := commit.Hash[:min(7, len(commit.Hash))]
		if repoURL != "" {
			return line + text + fmt.Sprintf(" ([%s](%s/commit/%s))", short, repoURL, commit.Hash)
		}
		return line + text + " (" + short + ")"
	}
	section := func(title string, items []string) {
		if len(items) > 0 {
			fmt.Fprintf(&builder, "\n### %s\n\n%s\n", title, strings.Join(items, "\n"))
		}
	}

	var breaking []string
	for _, commit := range commits {
		if commit.Breaking {
			breaking = append(breaking, item(commit, commit.BreakingNote))
		}
	}
	section("⚠ BREAKING CHANGES", breaking)
	for _, kind := range sections {
		var items []string
		for _, commit := range commits {
			if commit.Type == kind.typ {
				items = append(items, item(commit, commit.Subject))
			}
		}
		section(kind.title, items)
	}
	return builder.String()
}
