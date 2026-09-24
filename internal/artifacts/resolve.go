// Package artifacts turns "path[=name]" specs into concrete files to upload.
package artifacts

import (
	"cmp"
	"fmt"
	"path/filepath"
	"strings"
)

type Asset struct{ Path, Name string }

// Resolve expands globs and ${version} in names. Every spec must match at least one file.
func Resolve(specs []string, version string) ([]Asset, error) {
	var assets []Asset
	for _, spec := range specs {
		pattern, name, _ := strings.Cut(spec, "=")
		matches, err := filepath.Glob(pattern)
		if err != nil || len(matches) == 0 {
			return nil, fmt.Errorf("artifact %q matched no files", pattern)
		}
		for _, file := range matches {
			assetName := strings.ReplaceAll(cmp.Or(name, filepath.Base(file)), "${version}", version)
			assets = append(assets, Asset{Path: file, Name: assetName})
		}
	}
	return assets, nil
}
