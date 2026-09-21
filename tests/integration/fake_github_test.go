package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// fakeGitHub records releases and asset uploads, standing in for api.github.com.
type fakeGitHub struct {
	*httptest.Server
	mu       sync.Mutex
	releases []map[string]any // decoded POST /releases bodies
	assets   map[string][]byte
}

func newFakeGitHub(t *testing.T) *fakeGitHub {
	f := &fakeGitHub{assets: map[string][]byte{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /repos/o/r/releases", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
		var rel map[string]any
		if err := json.NewDecoder(r.Body).Decode(&rel); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		f.releases = append(f.releases, rel)
		f.mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"html_url":   f.URL + "/o/r/releases/" + fmt.Sprint(rel["tag_name"]),
			"upload_url": f.URL + "/uploads/assets{?name,label}",
		})
	})
	mux.HandleFunc("POST /uploads/assets", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.assets[r.URL.Query().Get("name")] = data
		f.mu.Unlock()
		w.WriteHeader(http.StatusCreated)
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

// env points releaser at the fake as if it ran in Actions for o/r.
func (f *fakeGitHub) env() []string {
	return []string{"GITHUB_API_URL=" + f.URL, "GITHUB_TOKEN=test-token", "GITHUB_REPOSITORY=o/r"}
}
