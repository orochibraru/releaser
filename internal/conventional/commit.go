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
	m := headerRe.FindStringSubmatch(header)
	if m == nil {
		return Commit{}, false
	}
	c := Commit{Hash: hash, Type: strings.ToLower(m[1]), Scope: m[2], Subject: m[4], Breaking: m[3] == "!"}
	if b := breakingRe.FindStringSubmatch(body); b != nil {
		c.Breaking, c.BreakingNote = true, strings.TrimSpace(b[1])
	}
	if c.Breaking && c.BreakingNote == "" {
		c.BreakingNote = c.Subject
	}
	return c, true
}
