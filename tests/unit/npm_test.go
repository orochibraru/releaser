package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orochibraru/releaser/internal/npm"
)

func TestSetVersion(test *testing.T) {
	in := "{\n\t\"name\": \"x\",\n\t\"version\" : \"0.1.0\",\n\t\"deps\": {\"version\": \"9\"}\n}\n"
	want := "{\n\t\"name\": \"x\",\n\t\"version\" : \"1.2.3\",\n\t\"deps\": {\"version\": \"9\"}\n}\n"
	got, ok := npm.SetVersion([]byte(in), "1.2.3")
	if !ok || string(got) != want {
		test.Errorf("SetVersion = %q, %v", got, ok)
	}
	if _, ok := npm.SetVersion([]byte(`{"name":"x"}`), "1.2.3"); ok {
		test.Error("bumped a package.json with no version")
	}
}

func TestSetVersionFile(test *testing.T) {
	dir := test.TempDir()
	if bumped, err := npm.SetVersionFile(filepath.Join(dir, "missing.json"), "1.0.0"); bumped || err != nil {
		test.Errorf("missing file: %v %v", bumped, err)
	}
	noVersion := filepath.Join(dir, "package.json")
	os.WriteFile(noVersion, []byte(`{"name":"x"}`), 0o644)
	if bumped, err := npm.SetVersionFile(noVersion, "1.0.0"); bumped || err != nil {
		test.Errorf("no version field: %v %v", bumped, err)
	}
	if _, err := npm.SetVersionFile(dir, "1.0.0"); err == nil {
		test.Error("reading a directory succeeded")
	}
}
