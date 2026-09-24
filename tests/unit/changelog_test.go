package unit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/orochibraru/releaser/internal/changelog"
	"github.com/orochibraru/releaser/internal/conventional"
)

func TestNotes(test *testing.T) {
	commits := []conventional.Commit{
		{Hash: "1111111aaaa", Type: "feat", Scope: "ui", Subject: "dark mode"},
		{Hash: "2222222bbbb", Type: "fix", Subject: "crash", Breaking: true, BreakingNote: "config moved"},
		{Hash: "3333333cccc", Type: "chore", Subject: "hidden"},
	}
	got := changelog.Notes("https://github.com/o/r", "v1.0.0", "v2.0.0", time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC), commits)
	want := `## [2.0.0](https://github.com/o/r/compare/v1.0.0...v2.0.0) (2026-09-19)

### ⚠ BREAKING CHANGES

* config moved ([2222222](https://github.com/o/r/commit/2222222bbbb))

### Features

* **ui:** dark mode ([1111111](https://github.com/o/r/commit/1111111aaaa))

### Bug Fixes

* crash ([2222222](https://github.com/o/r/commit/2222222bbbb))
`
	if got != want {
		test.Errorf("Notes:\n%s\nwant:\n%s", got, want)
	}

	first := changelog.Notes("", "", "v1.0.0", time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC), commits[:1])
	if !strings.HasPrefix(first, "## 1.0.0 (2026-09-19)\n") || !strings.Contains(first, "dark mode (1111111)") {
		test.Errorf("first release notes without repo:\n%s", first)
	}
}

func TestPrepend(test *testing.T) {
	if got := changelog.Prepend("", "## 1.0.0\n"); got != "# Changelog\n\n## 1.0.0\n" { // single trailing newline: end-of-file-fixer friendly
		test.Errorf("empty: %q", got)
	}
	if got := changelog.Prepend("# Changelog\n\n## 1.0.0\n", "## 1.1.0\n"); got != "# Changelog\n\n## 1.1.0\n\n## 1.0.0\n" {
		test.Errorf("existing: %q", got)
	}
}

func TestLatestEntry(test *testing.T) {
	older, newer := "## 1.0.0\n\n### Features\n\n* a\n", "## 1.1.0\n\n### Bug Fixes\n\n* b\n"
	if got := changelog.Latest(changelog.Prepend(changelog.Prepend("", older), newer)); got != newer {
		test.Errorf("Latest = %q, want %q", got, newer)
	}
	if got := changelog.Latest(changelog.Prepend("", older)); got != older {
		test.Errorf("Latest of one entry = %q", got)
	}
}

func TestPrependFile(test *testing.T) {
	path := filepath.Join(test.TempDir(), "CHANGELOG.md")
	if err := changelog.PrependFile(path, "## 1.0.0\n"); err != nil {
		test.Fatal(err)
	}
	if err := changelog.PrependFile(path, "## 1.1.0\n"); err != nil {
		test.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != "# Changelog\n\n## 1.1.0\n\n## 1.0.0\n" {
		test.Errorf("CHANGELOG.md = %q", got)
	}
	if err := changelog.PrependFile(test.TempDir(), "## 1.0.0\n"); err == nil {
		test.Error("prepending to a directory succeeded")
	}
}
