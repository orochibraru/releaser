package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// UpsertPR opens a pull request from head into base, or retitles the one already open.
func UpsertPR(repo, token, head, base, title, body string) (string, error) {
	owner, _, _ := strings.Cut(repo, "/")
	var open []struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
	}
	q := url.Values{"head": {owner + ":" + head}, "base": {base}, "state": {"open"}}
	if err := send(http.MethodGet, api()+"/repos/"+repo+"/pulls?"+q.Encode(), token, "application/json", nil, &open); err != nil {
		return "", err
	}
	var pr struct {
		HTMLURL string `json:"html_url"`
	}
	if len(open) > 0 {
		payload, _ := json.Marshal(map[string]string{"title": title, "body": body})
		return open[0].HTMLURL, send(http.MethodPatch, fmt.Sprintf("%s/repos/%s/pulls/%d", api(), repo, open[0].Number), token, "application/json", payload, nil)
	}
	payload, _ := json.Marshal(map[string]string{"title": title, "body": body, "head": head, "base": base})
	err := post(api()+"/repos/"+repo+"/pulls", token, "application/json", payload, &pr)
	if err != nil && strings.Contains(err.Error(), "not permitted to create or approve pull requests") {
		err = fmt.Errorf("%w\nenable Settings > Actions > General > \"Allow GitHub Actions to create and approve pull requests\"", err)
	}
	return pr.HTMLURL, err
}
