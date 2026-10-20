package forgewrite

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Gitea writes through Gitea's REST API. The token is fetched at call time
// through Token and is never stored, printed or placed in an error.
type Gitea struct {
	BaseURL string                 // https://host, no trailing slash
	Token   func() (string, error) // resolved per call from the secrets store
	Client  *http.Client           // nil means http.DefaultClient
}

// maxErrBody bounds how much of a failing response is quoted back.
const maxErrBody = 300

// maxLabelPages bounds the label lookup so a misbehaving server cannot loop it.
const maxLabelPages = 20

// giteaPullsPage is the page size the closed-PR listing asks for, and
// maxPullPages bounds it, newest close first. The server may answer with fewer
// rows per page ([api] MAX_RESPONSE_ITEMS clamps limit), so only an empty page
// ends the listing; a short one does not.
const (
	giteaPullsPage = 50
	maxPullPages   = 10
)

// CommentIssue implements Backend.
func (g Gitea) CommentIssue(ctx context.Context, r Repo, number int, body string) (string, error) {
	return g.writeURL(ctx, http.MethodPost, g.repoPath(r, fmt.Sprintf("issues/%d/comments", number)), map[string]string{"body": body})
}

// CreateIssue implements Backend. Gitea takes label IDs, not names, so the
// names are resolved first; an unknown name is an error rather than a
// silently unlabelled issue.
func (g Gitea) CreateIssue(ctx context.Context, r Repo, in NewIssue) (string, error) {
	payload := map[string]any{"title": in.Title, "body": in.Body}
	if len(in.Labels) > 0 {
		ids, err := g.labelIDs(ctx, r, in.Labels)
		if err != nil {
			return "", err
		}
		payload["labels"] = ids
	}
	return g.writeURL(ctx, http.MethodPost, g.repoPath(r, "issues"), payload)
}

// EditIssue implements Backend.
func (g Gitea) EditIssue(ctx context.Context, r Repo, number int, in IssueEdit) (string, error) {
	return g.writeURL(ctx, http.MethodPatch, g.repoPath(r, fmt.Sprintf("issues/%d", number)), editPayload(in))
}

// CreatePR implements Backend. Gitea's pull request API has no draft field (a
// draft is a title prefix the instance configures), so a draft request is
// refused rather than guessed.
func (g Gitea) CreatePR(ctx context.Context, r Repo, in NewPR) (string, error) {
	if in.Draft {
		return "", fmt.Errorf("ERROR: --draft is not supported on %s. WHY: Gitea marks a draft with a configurable title prefix, not a field, and dotf forge does not guess it. FIX: open the pull request without --draft", r.Host)
	}
	return g.writeURL(ctx, http.MethodPost, g.repoPath(r, "pulls"),
		map[string]string{"title": in.Title, "body": in.Body, "head": in.Head, "base": in.Base})
}

// LandedHeads implements Backend. Gitea 1.25's pull list has no head filter,
// so the closed pull requests are read newest close first and matched here:
// merged, from this repository and not a fork, with branch as their head
// label. Not the head ref: once the branch is deleted, which a squash merge
// usually leaves, Gitea reports the ref as refs/pull/<n>/head and names the
// branch only in the label. Every merge closes its pull request, so a recent
// one is on the first pages.
func (g Gitea) LandedHeads(ctx context.Context, r Repo, branch string) ([]string, error) {
	var heads []string
	for page := 1; page <= maxPullPages; page++ {
		raw, err := g.do(ctx, http.MethodGet, g.repoPath(r, fmt.Sprintf("pulls?state=closed&sort=recentclose&limit=%d&page=%d", giteaPullsPage, page)), nil)
		if err != nil {
			return nil, err
		}
		var pulls []struct {
			Merged bool `json:"merged"`
			Head   struct {
				Label  string `json:"label"`
				SHA    string `json:"sha"`
				RepoID int64  `json:"repo_id"`
			} `json:"head"`
			Base struct {
				RepoID int64 `json:"repo_id"`
			} `json:"base"`
		}
		if err := json.Unmarshal(raw, &pulls); err != nil {
			return nil, fmt.Errorf("gitea pull request list for %s was not readable", r)
		}
		for _, p := range pulls {
			if p.Merged && p.Head.Label == branch && p.Head.RepoID == p.Base.RepoID {
				heads = append(heads, p.Head.SHA)
			}
		}
		if len(pulls) == 0 {
			return heads, nil
		}
	}
	if len(heads) == 0 {
		return nil, fmt.Errorf("no merged pull request for %s in the newest %d pages of closed ones of %s, and the listing goes on: an older one may exist", branch, maxPullPages, r)
	}
	return heads, nil
}

func (g Gitea) repoPath(r Repo, tail string) string {
	return fmt.Sprintf("/api/v1/repos/%s/%s/%s", r.Owner, r.Name, tail)
}

// editPayload keeps unset fields out of the request, so an edit of the title
// does not blank the body.
func editPayload(in IssueEdit) map[string]string {
	p := map[string]string{}
	if in.Title != nil {
		p["title"] = *in.Title
	}
	if in.Body != nil {
		p["body"] = *in.Body
	}
	return p
}

// labelIDs maps label names to Gitea IDs from the repository's own labels.
func (g Gitea) labelIDs(ctx context.Context, r Repo, names []string) ([]int64, error) {
	byName := map[string]int64{}
	truncated := true
	for page := 1; page <= maxLabelPages; page++ {
		raw, err := g.do(ctx, http.MethodGet, g.repoPath(r, fmt.Sprintf("labels?limit=50&page=%d", page)), nil)
		if err != nil {
			return nil, err
		}
		var labels []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(raw, &labels); err != nil {
			return nil, fmt.Errorf("gitea label list for %s was not readable", r)
		}
		for _, l := range labels {
			byName[l.Name] = l.ID
		}
		if len(labels) < 50 {
			truncated = false
			break
		}
	}
	ids := make([]int64, 0, len(names))
	for _, n := range names {
		id, ok := byName[n]
		if !ok && truncated {
			return nil, fmt.Errorf("ERROR: label %q was not found in the first %d labels of %s. WHY: the listing was cut off, so the label may exist beyond it. FIX: use a repository with fewer labels or create the issue without --label", n, maxLabelPages*50, r)
		}
		if !ok {
			return nil, fmt.Errorf("ERROR: label %q does not exist on %s. WHY: Gitea takes label IDs, so a name must already exist there. FIX: create the label first, or drop it from --label", n, r)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// writeURL sends a write and returns the html_url of the result. A 2xx means
// the write happened; an unreadable body must not turn that into a failure,
// or a caller that retries would write it twice. The URL is a convenience, so
// it is returned empty rather than as an error.
func (g Gitea) writeURL(ctx context.Context, method, path string, payload any) (string, error) {
	raw, err := g.do(ctx, method, path, payload)
	if err != nil {
		return "", err
	}
	var out struct {
		HTMLURL string `json:"html_url"`
	}
	_ = json.Unmarshal(raw, &out)
	return out.HTMLURL, nil
}

// do sends one authenticated request and returns the body of a 2xx answer.
// Every error path redacts the token, including a server that echoes it back.
func (g Gitea) do(ctx context.Context, method, path string, payload any) ([]byte, error) {
	token, err := g.Token()
	if err != nil {
		return nil, fmt.Errorf("gitea token: %s", err)
	}
	if token == "" {
		return nil, fmt.Errorf("gitea token is empty")
	}
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}
	url := strings.TrimRight(g.BaseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("%s", redact(err.Error(), token))
	}
	req.Header.Set("Authorization", "token "+token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := g.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitea %s %s: %s", method, url, redact(err.Error(), token))
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		quoted := redact(string(raw), token)
		if len(quoted) > maxErrBody {
			quoted = quoted[:maxErrBody] + "..."
		}
		return nil, fmt.Errorf("gitea %s %s: HTTP %d: %s", method, url, resp.StatusCode, quoted)
	}
	return raw, nil
}
