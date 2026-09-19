package github

import (
	"os"
	"strings"
)

// SetOutput appends key=value lines to $GITHUB_OUTPUT; no-op outside Actions.
func SetOutput(lines ...string) error {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(strings.Join(lines, "\n") + "\n")
	return err
}
