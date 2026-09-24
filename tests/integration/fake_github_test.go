package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
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

func newFakeGitHub(test *testing.T) *fakeGitHub {
	fake := &fakeGitHub{assets: map[string][]byte{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /repos/o/r/releases", func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(writer, "bad token", http.StatusUnauthorized)
			return
		}
		var rel map[string]any
		if err := json.NewDecoder(request.Body).Decode(&rel); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		fake.mu.Lock()
		fake.releases = append(fake.releases, rel)
		fake.mu.Unlock()
		writer.WriteHeader(http.StatusCreated)
		json.NewEncoder(writer).Encode(map[string]string{
			"html_url":   fake.URL + "/o/r/releases/" + fmt.Sprint(rel["tag_name"]),
			"upload_url": fake.URL + "/uploads/assets{?name,label}",
		})
	})
	mux.HandleFunc("POST /uploads/assets", func(writer http.ResponseWriter, request *http.Request) {
		data, _ := io.ReadAll(request.Body)
		fake.mu.Lock()
		fake.assets[request.URL.Query().Get("name")] = data
		fake.mu.Unlock()
		writer.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("GET /repos/o/r/pulls", func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		open := []map[string]any{}
		fake.mu.Lock()
		for index, pr := range fake.pulls {
			if "o:"+fmt.Sprint(pr["head"]) == query.Get("head") && pr["base"] == query.Get("base") && pr["state"] == query.Get("state") {
				open = append(open, map[string]any{"number": index + 1, "html_url": fake.URL + "/o/r/pull/" + fmt.Sprint(index+1)})
			}
		}
		fake.mu.Unlock()
		json.NewEncoder(writer).Encode(open)
	})
	mux.HandleFunc("POST /repos/o/r/pulls", func(writer http.ResponseWriter, request *http.Request) {
		var pr map[string]any
		json.NewDecoder(request.Body).Decode(&pr)
		pr["state"] = "open"
		fake.mu.Lock()
		fake.pulls = append(fake.pulls, pr)
		number := len(fake.pulls)
		fake.mu.Unlock()
		writer.WriteHeader(http.StatusCreated)
		json.NewEncoder(writer).Encode(map[string]string{"html_url": fake.URL + "/o/r/pull/" + fmt.Sprint(number)})
	})
	mux.HandleFunc("PATCH /repos/o/r/pulls/{n}", func(writer http.ResponseWriter, request *http.Request) {
		number, _ := strconv.Atoi(request.PathValue("n"))
		fake.mu.Lock()
		defer fake.mu.Unlock()
		if number < 1 || number > len(fake.pulls) {
			http.NotFound(writer, request)
			return
		}
		patched := maps.Clone(fake.pulls[number-1]) // a copy: the test may hold the old map
		json.NewDecoder(request.Body).Decode(&patched)
		fake.pulls[number-1] = patched
	})
	fake.Server = httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		fake.mu.Lock()
		fail := fake.failing != "" && strings.HasPrefix(request.Method+" "+request.URL.Path, fake.failing)
		fake.mu.Unlock()
		if fail {
			http.Error(writer, "boom", http.StatusInternalServerError)
			return
		}
		mux.ServeHTTP(writer, request)
	}))
	fake.Start() // after fake.Server is set: handlers read fake.URL
	test.Cleanup(fake.Close)
	return fake
}

// env points releaser at the fake as if it ran in Actions for o/r.
func (fake *fakeGitHub) env() []string {
	return []string{"GITHUB_API_URL=" + fake.URL, "GITHUB_TOKEN=test-token", "GITHUB_REPOSITORY=o/r"}
}
