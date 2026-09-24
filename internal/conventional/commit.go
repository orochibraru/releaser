// Package conventional parses Conventional Commits and decides the release bump.
package conventional

import (
	"regexp"
	"strings"
)

var (
	headerRe   = regexp.MustCompile(`^(\w+)(?:\(([^)]*)\))?(!)?: (.+)`)
	breakingRe = regexp.MustCompile(`(?m)^BREAKING[ -]CHANGE: ([\s\S]+)`)
)

type Commit struct {
	Hash, Type, Scope, Subject string
	Breaking                   bool
	BreakingNote               string // footer text, or the subject for "type!:" commits
}

// Parse returns false for commits that don't follow the convention.
func Parse(hash, message string) (Commit, bool) {
	header, body, _ := strings.Cut(strings.TrimSpace(message), "\n")
	match := headerRe.FindStringSubmatch(header)
	if match == nil {
		return Commit{}, false
	}
	commit := Commit{Hash: hash, Type: strings.ToLower(match[1]), Scope: match[2], Subject: match[4], Breaking: match[3] == "!"}
	if footer := breakingRe.FindStringSubmatch(body); footer != nil {
		commit.Breaking, commit.BreakingNote = true, strings.TrimSpace(footer[1])
	}
	if commit.Breaking && commit.BreakingNote == "" {
		commit.BreakingNote = commit.Subject
	}
	return commit, true
}
