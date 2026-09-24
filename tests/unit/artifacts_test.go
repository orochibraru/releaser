package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orochibraru/releaser/internal/artifacts"
)

func TestResolve(test *testing.T) {
	dir := test.TempDir()
	for _, name := range []string{"a.zip", "b.zip", "ext.zip"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			test.Fatal(err)
		}
	}
	got, err := artifacts.Resolve([]string{
		filepath.Join(dir, "ext.zip") + "=ext-${version}.zip",
		filepath.Join(dir, "[ab].zip"),
	}, "1.2.3")
	if err != nil {
		test.Fatal(err)
	}
	names := []string{}
	for _, asset := range got {
		names = append(names, asset.Name)
	}
	if len(names) != 3 || names[0] != "ext-1.2.3.zip" || names[1] != "a.zip" || names[2] != "b.zip" {
		test.Errorf("names = %v", names)
	}
	if _, err := artifacts.Resolve([]string{filepath.Join(dir, "missing.zip")}, "1.0.0"); err == nil {
		test.Error("missing artifact accepted")
	}
}
