// Command fakegithub stands in for api.github.com in the examples workflow: it accepts releases
// and asset uploads and writes them under -dir (<tag>.json, assets/<name>).
package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	address := flag.String("addr", "127.0.0.1:8080", "listen address")
	dir := flag.String("dir", "github", "where releases and assets are written")
	flag.Parse()
	if err := os.MkdirAll(filepath.Join(*dir, "assets"), 0o755); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("POST /repos/{owner}/{repo}/releases", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var release struct {
			Tag string `json:"tag_name"`
		}
		if err := json.Unmarshal(body, &release); err != nil || release.Tag == "" {
			http.Error(w, "bad release payload", http.StatusBadRequest)
			return
		}
		if err := os.WriteFile(filepath.Join(*dir, filepath.Base(release.Tag)+".json"), body, 0o644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"html_url":   "http://" + *address + "/releases/" + release.Tag,
			"upload_url": "http://" + *address + "/uploads{?name,label}",
		})
	})
	http.HandleFunc("POST /uploads", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		name := filepath.Base(r.URL.Query().Get("name"))
		if err := os.WriteFile(filepath.Join(*dir, "assets", name), data, 0o644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})
	log.Fatal(http.ListenAndServe(*address, nil))
}
