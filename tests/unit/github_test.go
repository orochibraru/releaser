package unit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/orochibraru/releaser/internal/artifacts"
	"github.com/orochibraru/releaser/internal/github"
)

// api serves handler as GITHUB_API_URL for the test.
func api(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	s := httptest.NewServer(handler)
	t.Cleanup(s.Close)
	t.Setenv("GITHUB_API_URL", s.URL)
	return s
}

func TestCreateReleaseErrors(t *testing.T) {
	api(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Validation Failed"}`, http.StatusUnprocessableEntity)
	})
	err := github.CreateRelease("o/r", "t", "v1.0.0", "notes", nil, false, false)
	if err == nil || !strings.Contains(err.Error(), "422") || !strings.Contains(err.Error(), "Validation Failed") {
		t.Errorf("422 not surfaced with its body: %v", err)
	}

	// The asset vanished between Resolve and upload.
	s := api(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"upload_url": "http://" + r.Host + "/up{?name}"})
	})
	gone := []artifacts.Asset{{Path: filepath.Join(t.TempDir(), "gone.zip"), Name: "gone.zip"}}
	if err := github.CreateRelease("o/r", "t", "v1.0.0", "", gone, false, false); !os.IsNotExist(err) {
		t.Errorf("missing asset: %v", err)
	}

	s.Close() // connection refused
	if err := github.CreateRelease("o/r", "t", "v1.0.0", "", nil, false, false); err == nil {
		t.Error("closed server: no error")
	}
	t.Setenv("GITHUB_API_URL", "http://bad host")
	if err := github.CreateRelease("o/r", "t", "v1.0.0", "", nil, false, false); err == nil {
		t.Error("bad URL: no error")
	}
}

// UpsertPR looks for an open PR from owner:head into base; found → PATCH it, else POST a new one.
func TestUpsertPR(t *testing.T) {
	var calls []string
	open := "[]"
	api(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.RequestURI())
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(open))
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"html_url":"https://x/pull/1"}`))
		}
	})
	url, err := github.UpsertPR("o/r", "t", "releaser/release", "main", "chore(release): 1.0.0", "notes")
	if err != nil || url != "https://x/pull/1" {
		t.Fatalf("create: %s %v", url, err)
	}
	open = `[{"number":7,"html_url":"https://x/pull/7"}]`
	url, err = github.UpsertPR("o/r", "t", "releaser/release", "main", "chore(release): 1.1.0", "notes")
	if err != nil || url != "https://x/pull/7" {
		t.Fatalf("update: %s %v", url, err)
	}
	want := "GET /repos/o/r/pulls?base=main&head=o%3Areleaser%2Frelease&state=open," +
		"POST /repos/o/r/pulls," +
		"GET /repos/o/r/pulls?base=main&head=o%3Areleaser%2Frelease&state=open," +
		"PATCH /repos/o/r/pulls/7"
	if got := strings.Join(calls, ","); got != want {
		t.Errorf("calls:\n%s\nwant:\n%s", got, want)
	}
}

// The repo setting every release PR user hits first gets a pointer to the fix.
func TestUpsertPRForbidden(t *testing.T) {
	api(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Write([]byte("[]"))
			return
		}
		http.Error(w, `{"message":"GitHub Actions is not permitted to create or approve pull requests."}`, http.StatusForbidden)
	})
	_, err := github.UpsertPR("o/r", "t", "releaser/release", "main", "chore(release): 1.0.0", "")
	if err == nil || !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), `"Allow GitHub Actions to create and approve pull requests"`) {
		t.Errorf("err = %v", err)
	}
}

func TestSetOutput(t *testing.T) {
	t.Setenv("GITHUB_OUTPUT", "")
	if err := github.SetOutput("a=1"); err != nil {
		t.Errorf("outside Actions: %v", err)
	}
	path := filepath.Join(t.TempDir(), "out")
	t.Setenv("GITHUB_OUTPUT", path)
	github.SetOutput("a=1", "b=2")
	github.SetOutput("c=3")
	if got, _ := os.ReadFile(path); string(got) != "a=1\nb=2\nc=3\n" {
		t.Errorf("GITHUB_OUTPUT = %q", got)
	}
	t.Setenv("GITHUB_OUTPUT", filepath.Join(path, "not-a-dir", "out"))
	if err := github.SetOutput("a=1"); err == nil {
		t.Error("unwritable GITHUB_OUTPUT: no error")
	}
}
