package unit

import (
	"testing"

	"github.com/orochibraru/releaser/internal/npm"
)

func TestSetVersion(t *testing.T) {
	in := "{\n\t\"name\": \"x\",\n\t\"version\" : \"0.1.0\",\n\t\"deps\": {\"version\": \"9\"}\n}\n"
	want := "{\n\t\"name\": \"x\",\n\t\"version\" : \"1.2.3\",\n\t\"deps\": {\"version\": \"9\"}\n}\n"
	got, ok := npm.SetVersion([]byte(in), "1.2.3")
	if !ok || string(got) != want {
		t.Errorf("SetVersion = %q, %v", got, ok)
	}
	if _, ok := npm.SetVersion([]byte(`{"name":"x"}`), "1.2.3"); ok {
		t.Error("bumped a package.json with no version")
	}
}
