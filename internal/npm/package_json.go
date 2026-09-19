// Package npm bumps the version in package.json without reformatting it.
package npm

import (
	"os"
	"regexp"
	"slices"
)

var versionRe = regexp.MustCompile(`"version"\s*:\s*"([^"]*)"`)

// SetVersion rewrites the first "version" value in place.
// ponytail: first match wins, which is the top-level key in any conventional package.json.
func SetVersion(data []byte, version string) ([]byte, bool) {
	m := versionRe.FindSubmatchIndex(data)
	if m == nil {
		return data, false
	}
	return slices.Concat(data[:m[2]], []byte(version), data[m[3]:]), true
}

// SetVersionFile reports whether path exists and was bumped.
func SetVersionFile(path, version string) (bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	data, ok := SetVersion(data, version)
	if !ok {
		return false, nil
	}
	return true, os.WriteFile(path, data, 0o644)
}
