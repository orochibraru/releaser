package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orochibraru/releaser/internal/artifacts"
)

func TestResolve(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"a.zip", "b.zip", "ext.zip"} {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := artifacts.Resolve([]string{
		filepath.Join(dir, "ext.zip") + "=ext-${version}.zip",
		filepath.Join(dir, "[ab].zip"),
	}, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, a := range got {
		names = append(names, a.Name)
	}
	if len(names) != 3 || names[0] != "ext-1.2.3.zip" || names[1] != "a.zip" || names[2] != "b.zip" {
		t.Errorf("names = %v", names)
	}
	if _, err := artifacts.Resolve([]string{filepath.Join(dir, "missing.zip")}, "1.0.0"); err == nil {
		t.Error("missing artifact accepted")
	}
}
