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
func api(test *testing.T, handler http.HandlerFunc) *httptest.Server {
	server := httptest.NewServer(handler)
	test.Cleanup(server.Close)
	test.Setenv("GITHUB_API_URL", server.URL)
	return server
}

func TestCreateReleaseErrors(test *testing.T) {
	api(test, func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, `{"message":"Validation Failed"}`, http.StatusUnprocessableEntity)
	})
	err := github.CreateRelease("o/r", "t", "v1.0.0", "notes", nil, false, false)
	if err == nil || !strings.Contains(err.Error(), "422") || !strings.Contains(err.Error(), "Validation Failed") {
		test.Errorf("422 not surfaced with its body: %v", err)
	}

	// The asset vanished between Resolve and upload.
	server := api(test, func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusCreated)
		json.NewEncoder(writer).Encode(map[string]string{"upload_url": "http://" + request.Host + "/up{?name}"})
	})
	gone := []artifacts.Asset{{Path: filepath.Join(test.TempDir(), "gone.zip"), Name: "gone.zip"}}
	if err := github.CreateRelease("o/r", "t", "v1.0.0", "", gone, false, false); !os.IsNotExist(err) {
		test.Errorf("missing asset: %v", err)
	}

	server.Close() // connection refused
	if err := github.CreateRelease("o/r", "t", "v1.0.0", "", nil, false, false); err == nil {
		test.Error("closed server: no error")
	}
	test.Setenv("GITHUB_API_URL", "http://bad host")
	if err := github.CreateRelease("o/r", "t", "v1.0.0", "", nil, false, false); err == nil {
		test.Error("bad URL: no error")
	}
}

// UpsertPR looks for an open PR from owner:head into base; found → PATCH it, else POST a new one.
func TestUpsertPR(test *testing.T) {
	var calls []string
	open := "[]"
	api(test, func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.RequestURI())
		switch request.Method {
		case http.MethodGet:
			writer.Write([]byte(open))
		case http.MethodPost:
			writer.WriteHeader(http.StatusCreated)
			writer.Write([]byte(`{"html_url":"https://x/pull/1"}`))
		}
	})
	url, err := github.UpsertPR("o/r", "t", "releaser/release", "main", "chore(release): 1.0.0", "notes")
	if err != nil || url != "https://x/pull/1" {
		test.Fatalf("create: %s %v", url, err)
	}
	open = `[{"number":7,"html_url":"https://x/pull/7"}]`
	url, err = github.UpsertPR("o/r", "t", "releaser/release", "main", "chore(release): 1.1.0", "notes")
	if err != nil || url != "https://x/pull/7" {
		test.Fatalf("update: %s %v", url, err)
	}
	want := "GET /repos/o/r/pulls?base=main&head=o%3Areleaser%2Frelease&state=open," +
		"POST /repos/o/r/pulls," +
		"GET /repos/o/r/pulls?base=main&head=o%3Areleaser%2Frelease&state=open," +
		"PATCH /repos/o/r/pulls/7"
	if got := strings.Join(calls, ","); got != want {
		test.Errorf("calls:\n%s\nwant:\n%s", got, want)
	}
}

func TestSetOutput(test *testing.T) {
	test.Setenv("GITHUB_OUTPUT", "")
	if err := github.SetOutput("a=1"); err != nil {
		test.Errorf("outside Actions: %v", err)
	}
	path := filepath.Join(test.TempDir(), "out")
	test.Setenv("GITHUB_OUTPUT", path)
	github.SetOutput("a=1", "b=2")
	github.SetOutput("c=3")
	if got, _ := os.ReadFile(path); string(got) != "a=1\nb=2\nc=3\n" {
		test.Errorf("GITHUB_OUTPUT = %q", got)
	}
	test.Setenv("GITHUB_OUTPUT", filepath.Join(path, "not-a-dir", "out"))
	if err := github.SetOutput("a=1"); err == nil {
		test.Error("unwritable GITHUB_OUTPUT: no error")
	}
}
