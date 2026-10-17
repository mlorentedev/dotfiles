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

// CommentIssue implements Backend.
func (g Gitea) CommentIssue(ctx context.Context, r Repo, number int, body string) (string, error) {
	token, err := g.Token()
	if err != nil {
		return "", fmt.Errorf("gitea token for %s: %s", r.Host, err)
	}
	if token == "" {
		return "", fmt.Errorf("gitea token for %s is empty", r.Host)
	}
	payload, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments", strings.TrimRight(g.BaseURL, "/"), r.Owner, r.Name, number)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("%s", redact(err.Error(), token))
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")
	client := g.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gitea POST %s: %s", url, redact(err.Error(), token))
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		quoted := redact(string(raw), token)
		if len(quoted) > maxErrBody {
			quoted = quoted[:maxErrBody] + "..."
		}
		return "", fmt.Errorf("gitea POST %s: HTTP %d: %s", url, resp.StatusCode, quoted)
	}
	var out struct {
		HTMLURL string `json:"html_url"`
	}
	// A 2xx means the comment exists. An unreadable body must not turn that
	// into a failure, or a caller that retries posts the comment twice; the
	// URL is a convenience, so return it empty rather than an error.
	_ = json.Unmarshal(raw, &out)
	return out.HTMLURL, nil
}
