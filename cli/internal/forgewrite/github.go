package forgewrite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// GHRunner runs `gh` with stdin and returns its stdout; a seam so tests never
// reach GitHub.
type GHRunner func(ctx context.Context, stdin []byte, args ...string) ([]byte, error)

// GitHub writes through the gh REST endpoint, as lane.REST reads through it.
// The body travels on stdin as JSON, never as an argument.
type GitHub struct{ Run GHRunner }

// CommentIssue implements Backend.
func (g GitHub) CommentIssue(ctx context.Context, r Repo, number int, body string) (string, error) {
	payload, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return "", err
	}
	endpoint := fmt.Sprintf("repos/%s/%s/issues/%d/comments", r.Owner, r.Name, number)
	out, err := g.Run(ctx, payload, "api", "-X", "POST", endpoint, "--input", "-")
	if err != nil {
		return "", fmt.Errorf("gh api POST %s: %w", endpoint, err)
	}
	var resp struct {
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(out, &resp); err != nil || resp.HTMLURL == "" {
		return "", fmt.Errorf("gh api POST %s answered without a comment URL: %s", endpoint, strings.TrimSpace(string(out)))
	}
	return resp.HTMLURL, nil
}
