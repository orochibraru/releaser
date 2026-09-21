// Package github talks to the GitHub REST API and Actions runtime.
package github

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/orochibraru/releaser/internal/artifacts"
)

// CreateRelease publishes a release for tag (or creates it as a draft) and uploads assets to it.
// A prerelease never becomes the repository's latest release.
func CreateRelease(repo, token, tag, body string, assets []artifacts.Asset, draft, prerelease bool) error {
	payload, _ := json.Marshal(map[string]any{"tag_name": tag, "name": tag, "body": body, "draft": draft, "prerelease": prerelease})
	var rel struct {
		HTMLURL   string `json:"html_url"`
		UploadURL string `json:"upload_url"`
	}
	if err := post(api()+"/repos/"+repo+"/releases", token, "application/json", payload, &rel); err != nil {
		return err
	}
	upload, _, _ := strings.Cut(rel.UploadURL, "{")
	for _, a := range assets {
		data, err := os.ReadFile(a.Path)
		if err != nil {
			return err
		}
		if err := post(upload+"?name="+url.QueryEscape(a.Name), token, "application/octet-stream", data, nil); err != nil {
			return err
		}
		fmt.Println("uploaded", a.Name)
	}
	if draft {
		fmt.Println("created draft", rel.HTMLURL)
	} else {
		fmt.Println("released", rel.HTMLURL)
	}
	return nil
}

func api() string { return cmp.Or(os.Getenv("GITHUB_API_URL"), "https://api.github.com") }

func post(u, token, contentType string, body []byte, out any) error {
	return send(http.MethodPost, u, token, contentType, body, out)
}

func send(method, u, token, contentType string, body []byte, out any) error {
	req, err := http.NewRequest(method, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", contentType)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return fmt.Errorf("%s: %s: %s", u, res.Status, data)
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}
