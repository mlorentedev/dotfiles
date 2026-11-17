package forgewrite

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// giteaStatusError is a non-2xx answer, so a caller can tell a 404 from a failure.
type giteaStatusError struct {
	code int
	text string
}

func (e giteaStatusError) Error() string { return e.text }

// maxLabelPages bounds the label lookup so a misbehaving server cannot loop it.
const maxLabelPages = 20

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

// giteaWIPPrefixes are Gitea's default WORK_IN_PROGRESS_PREFIXES. A title that
// starts with one, in any case, makes the pull request a draft; Gitea has no
// draft field on create or edit, and no API that reads an instance's own list.
var giteaWIPPrefixes = []string{"WIP:", "[WIP]"}

// giteaPull is the part of a Gitea pull request dotf reads back.
type giteaPull struct {
	Title   string `json:"title"`
	Draft   bool   `json:"draft"`
	HTMLURL string `json:"html_url"`
}

// pull sends one request whose answer is a pull request and decodes it.
func (g Gitea) pull(ctx context.Context, method, path string, payload any) (giteaPull, error) {
	raw, err := g.do(ctx, method, path, payload)
	if err != nil {
		return giteaPull{}, err
	}
	var p giteaPull
	if err := json.Unmarshal(raw, &p); err != nil {
		return giteaPull{}, fmt.Errorf("gitea %s %s: the answer is not a pull request: %w", method, path, err)
	}
	return p, nil
}

// CreatePR implements Backend. A draft is opened with a "WIP: " title, and the
// draft field of the answer says whether the instance recognised it: Gitea
// exposes no API for its prefixes, so the prefix is verified, never assumed.
func (g Gitea) CreatePR(ctx context.Context, r Repo, in NewPR) (string, error) {
	path := g.repoPath(r, "pulls")
	if !in.Draft {
		return g.writeURL(ctx, http.MethodPost, path,
			map[string]string{"title": in.Title, "body": in.Body, "head": in.Head, "base": in.Base})
	}
	// A title that already carries a prefix keeps one, or pr ready would
	// strip one of two and still read a draft.
	title, _ := stripWIPPrefix(in.Title)
	p, err := g.pull(ctx, http.MethodPost, path,
		map[string]string{"title": giteaWIPPrefixes[0] + " " + title, "body": in.Body, "head": in.Head, "base": in.Base})
	if err != nil {
		return "", err
	}
	if !p.Draft {
		return "", fmt.Errorf("ERROR: %s was opened ready, not as a draft. WHY: %s does not treat a %q title prefix as work in progress. FIX: retitle it with the prefix that instance's WORK_IN_PROGRESS_PREFIXES names", p.HTMLURL, r.Host, giteaWIPPrefixes[0])
	}
	return p.HTMLURL, nil
}

// ReadyPR implements Backend. The draft prefix is removed from the title and
// the answer must say draft: false; a title that is a draft by a prefix dotf
// does not know is an error, not a guess. A ready pull request is not written.
func (g Gitea) ReadyPR(ctx context.Context, r Repo, number int) (string, error) {
	path := g.repoPath(r, fmt.Sprintf("pulls/%d", number))
	p, err := g.pull(ctx, http.MethodGet, path, nil)
	if err != nil || !p.Draft {
		return p.HTMLURL, err
	}
	title, ok := stripWIPPrefix(p.Title)
	if ok {
		p, err = g.pull(ctx, http.MethodPatch, path, map[string]string{"title": title})
		if err != nil {
			return "", err
		}
	}
	if p.Draft {
		return "", fmt.Errorf("ERROR: %s is still a draft. WHY: its title %q starts with a work-in-progress prefix of %s other than %s. FIX: remove that prefix from the title", p.HTMLURL, p.Title, r.Host, strings.Join(giteaWIPPrefixes, " or "))
	}
	return p.HTMLURL, nil
}

// stripWIPPrefix removes one leading default prefix, in any case, and the
// spaces after it.
func stripWIPPrefix(title string) (string, bool) {
	for _, prefix := range giteaWIPPrefixes {
		if len(title) >= len(prefix) && strings.EqualFold(title[:len(prefix)], prefix) {
			return strings.TrimLeft(title[len(prefix):], " "), true
		}
	}
	return title, false
}

// LandedHeads implements Backend. It asks for the pull request of branch against
// the default branch, which Gitea finds by head even once the branch is deleted.
// Reading the closed list instead took about 5.5s a page and could not finish
// on a repository with hundreds of them (#3072). The lookup returns the newest
// pull request only, against one base; what it misses reads "not landed", the
// side that keeps a worktree (ADR-052).
func (g Gitea) LandedHeads(ctx context.Context, r Repo, branch string) ([]string, error) {
	raw, err := g.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/repos/%s/%s", r.Owner, r.Name), nil)
	if err != nil {
		return nil, err
	}
	var repo struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.Unmarshal(raw, &repo); err != nil || repo.DefaultBranch == "" {
		return nil, fmt.Errorf("gitea names no default branch for %s", r)
	}
	raw, err = g.do(ctx, http.MethodGet, g.repoPath(r, "pulls/"+escapeRef(repo.DefaultBranch)+"/"+escapeRef(branch)), nil)
	var status giteaStatusError
	if errors.As(err, &status) && status.code == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var pull struct {
		Merged bool `json:"merged"`
		Head   struct {
			SHA    string `json:"sha"`
			RepoID int64  `json:"repo_id"`
		} `json:"head"`
		Base struct {
			RepoID int64 `json:"repo_id"`
		} `json:"base"`
	}
	if err := json.Unmarshal(raw, &pull); err != nil {
		return nil, fmt.Errorf("gitea pull request of %s in %s was not readable", branch, r)
	}
	if !pull.Merged || pull.Head.RepoID != pull.Base.RepoID {
		return nil, nil
	}
	return []string{pull.Head.SHA}, nil
}

// escapeRef escapes a ref for a URL path, keeping its slashes as separators:
// Gitea's route takes the rest of the path as the head.
func escapeRef(ref string) string {
	parts := strings.Split(ref, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
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
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	// The status outranks a cut-off body: a 404 is an answer however much of
	// its page arrived.
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		quoted := redact(string(raw), token)
		if len(quoted) > maxErrBody {
			quoted = quoted[:maxErrBody] + "..."
		}
		return nil, giteaStatusError{code: resp.StatusCode, text: fmt.Sprintf("gitea %s %s: HTTP %d: %s", method, url, resp.StatusCode, quoted)}
	}
	if readErr != nil {
		return nil, fmt.Errorf("gitea %s %s: reading the answer: %s", method, url, redact(readErr.Error(), token))
	}
	return raw, nil
}
