package changelog

import (
	"os"
	"strings"
)

const title = "# Changelog"

// Prepend puts notes on top of an existing changelog, keeping a single title.
func Prepend(existing, notes string) string {
	rest := strings.TrimLeft(strings.TrimPrefix(existing, title), "\n")
	out := title + "\n\n" + notes
	if rest != "" {
		out += "\n" + rest
	}
	return out
}

// Latest returns the newest entry of a changelog, exactly as Notes rendered it.
func Latest(changelog string) string {
	rest := strings.TrimLeft(strings.TrimPrefix(changelog, title), "\n")
	if index := strings.Index(rest, "\n## "); index >= 0 {
		return rest[:index]
	}
	return rest
}

// PrependFile applies Prepend to a file, creating it if missing.
func PrependFile(path, notes string) error {
	old, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(path, []byte(Prepend(string(old), notes)), 0o644)
}
