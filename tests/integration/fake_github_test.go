package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeGitHub records releases, asset uploads and pull requests, standing in for api.github.com.
type fakeGitHub struct {
	*httptest.Server
	mu       sync.Mutex
	releases []map[string]any // decoded POST /releases bodies
	assets   map[string][]byte
	pulls    []map[string]any // POST /pulls bodies, updated by PATCH; "state" is "open" until merged
	failing  string           // requests whose "METHOD /path" starts with this get a 500
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
	mux.HandleFunc("GET /repos/o/r/pulls", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		open := []map[string]any{}
		f.mu.Lock()
		for i, pr := range f.pulls {
			if "o:"+fmt.Sprint(pr["head"]) == q.Get("head") && pr["base"] == q.Get("base") && pr["state"] == q.Get("state") {
				open = append(open, map[string]any{"number": i + 1, "html_url": f.URL + "/o/r/pull/" + fmt.Sprint(i+1)})
			}
		}
		f.mu.Unlock()
		json.NewEncoder(w).Encode(open)
	})
	mux.HandleFunc("POST /repos/o/r/pulls", func(w http.ResponseWriter, r *http.Request) {
		var pr map[string]any
		json.NewDecoder(r.Body).Decode(&pr)
		pr["state"] = "open"
		f.mu.Lock()
		f.pulls = append(f.pulls, pr)
		n := len(f.pulls)
		f.mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"html_url": f.URL + "/o/r/pull/" + fmt.Sprint(n)})
	})
	mux.HandleFunc("PATCH /repos/o/r/pulls/{n}", func(w http.ResponseWriter, r *http.Request) {
		n, _ := strconv.Atoi(r.PathValue("n"))
		f.mu.Lock()
		defer f.mu.Unlock()
		if n < 1 || n > len(f.pulls) {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&f.pulls[n-1])
	})
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		fail := f.failing != "" && strings.HasPrefix(r.Method+" "+r.URL.Path, f.failing)
		f.mu.Unlock()
		if fail {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(f.Close)
	return f
}

// env points releaser at the fake as if it ran in Actions for o/r.
func (f *fakeGitHub) env() []string {
	return []string{"GITHUB_API_URL=" + f.URL, "GITHUB_TOKEN=test-token", "GITHUB_REPOSITORY=o/r"}
}
