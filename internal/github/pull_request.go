package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type pullRequest struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
	Head    struct {
		Ref  string `json:"ref"`
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

// maxPRPages bounds the search for an API that ignores the page parameter.
const maxPRPages = 20

// UpsertPR opens a pull request from head into base, or retitles the one already open.
func UpsertPR(repo, token, head, base, title, body string) (string, error) {
	existing, err := findOpenPR(repo, token, head, base)
	if err != nil {
		return "", err
	}
	if existing != nil {
		payload, _ := json.Marshal(map[string]string{"title": title, "body": body})
		return existing.HTMLURL, send(http.MethodPatch, fmt.Sprintf("%s/repos/%s/pulls/%d", api(), repo, existing.Number), token, "application/json", payload, nil)
	}
	var created pullRequest
	payload, _ := json.Marshal(map[string]string{"title": title, "body": body, "head": head, "base": base})
	err = post(api()+"/repos/"+repo+"/pulls", token, "application/json", payload, &created)
	return created.HTMLURL, err
}

// findOpenPR returns the open pull request from this repository's head into base, or nil.
// GitHub narrows the list with the head and base filters; Gitea ignores them and lists every
// open pull request a page at a time, so each one is checked here.
func findOpenPR(repo, token, head, base string) (*pullRequest, error) {
	owner, _, _ := strings.Cut(repo, "/")
	for page := 1; page <= maxPRPages; page++ {
		query := url.Values{"head": {owner + ":" + head}, "base": {base}, "state": {"open"}, "page": {strconv.Itoa(page)}}
		var open []pullRequest
		if err := send(http.MethodGet, api()+"/repos/"+repo+"/pulls?"+query.Encode(), token, "application/json", nil, &open); err != nil {
			return nil, err
		}
		if len(open) == 0 {
			return nil, nil
		}
		for index := range open {
			candidate := &open[index]
			if candidate.Head.Ref == head && candidate.Base.Ref == base && strings.EqualFold(candidate.Head.Repo.FullName, repo) {
				return candidate, nil
			}
		}
	}
	return nil, nil
}
