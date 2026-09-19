package unit

import (
	"testing"

	"github.com/orochibraru/releaser/internal/conventional"
)

func TestParse(t *testing.T) {
	cases := []struct {
		msg                 string
		ok                  bool
		typ, scope, subject string
		breaking            bool
		breakingNote        string
	}{
		{"feat(api): add thing", true, "feat", "api", "add thing", false, ""},
		{"fix!: drop node 18", true, "fix", "", "drop node 18", true, "drop node 18"},
		{"Feat: caps\n\nBREAKING CHANGE: config moved", true, "feat", "", "caps", true, "config moved"},
		{"Merge branch 'x'", false, "", "", "", false, ""},
		{"wip stuff", false, "", "", "", false, ""},
	}
	for _, c := range cases {
		got, ok := conventional.Parse("abc", c.msg)
		if ok != c.ok || got.Type != c.typ || got.Scope != c.scope || got.Subject != c.subject ||
			got.Breaking != c.breaking || got.BreakingNote != c.breakingNote {
			t.Errorf("Parse(%q) = %+v, %v", c.msg, got, ok)
		}
	}
}

func TestBump(t *testing.T) {
	feat := conventional.Commit{Type: "feat"}
	breakingFix := conventional.Commit{Type: "fix", Breaking: true}
	chore := conventional.Commit{Type: "chore"}

	def, _ := conventional.ParseRules(nil)
	if got := conventional.Bump([]conventional.Commit{chore}, def); got != conventional.None {
		t.Errorf("chore bumped %d", got)
	}
	if got := conventional.Bump([]conventional.Commit{feat, chore}, def); got != conventional.Minor {
		t.Errorf("feat = %d, want minor", got)
	}
	if got := conventional.Bump([]conventional.Commit{feat, breakingFix}, def); got != conventional.Major {
		t.Errorf("breaking = %d, want major", got)
	}

	// The "everything is a patch" setup from bercail's .releaserc.json.
	patchy, err := conventional.ParseRules([]string{"breaking=patch", "feat=patch", "docs=patch", "refactor=patch"})
	if err != nil {
		t.Fatal(err)
	}
	if got := conventional.Bump([]conventional.Commit{feat, breakingFix, {Type: "docs"}}, patchy); got != conventional.Patch {
		t.Errorf("patchy = %d, want patch", got)
	}

	if _, err := conventional.ParseRules([]string{"feat=huge"}); err == nil {
		t.Error("bad level accepted")
	}
}
