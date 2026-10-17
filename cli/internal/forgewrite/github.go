package forgewrite

import (
	"context"
	"encoding/json"
	"fmt"
)

// GHRunner runs `gh` with stdin and returns its stdout; a seam so tests never
// reach GitHub.
type GHRunner func(ctx context.Context, stdin []byte, args ...string) ([]byte, error)

// GitHub writes through the gh REST endpoint, as lane.REST reads through it.
// The body travels on stdin as JSON, never as an argument.
type GitHub struct{ Run GHRunner }

// CommentIssue implements Backend.
func (g GitHub) CommentIssue(ctx context.Context, r Repo, number int, body string) (string, error) {
	return g.writeURL(ctx, "POST", fmt.Sprintf("repos/%s/%s/issues/%d/comments", r.Owner, r.Name, number), map[string]string{"body": body})
}

// CreateIssue implements Backend. GitHub takes label names directly.
func (g GitHub) CreateIssue(ctx context.Context, r Repo, in NewIssue) (string, error) {
	payload := map[string]any{"title": in.Title, "body": in.Body}
	if len(in.Labels) > 0 {
		payload["labels"] = in.Labels
	}
	return g.writeURL(ctx, "POST", fmt.Sprintf("repos/%s/%s/issues", r.Owner, r.Name), payload)
}

// EditIssue implements Backend.
func (g GitHub) EditIssue(ctx context.Context, r Repo, number int, in IssueEdit) (string, error) {
	return g.writeURL(ctx, "PATCH", fmt.Sprintf("repos/%s/%s/issues/%d", r.Owner, r.Name, number), editPayload(in))
}

// writeURL sends a JSON write through `gh api` with the payload on stdin and
// returns the html_url of the result. gh exiting 0 means the write exists; an
// unreadable body must not read as a failure that a retry would duplicate.
func (g GitHub) writeURL(ctx context.Context, method, endpoint string, payload any) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	out, err := g.Run(ctx, data, "api", "-X", method, endpoint, "--input", "-")
	if err != nil {
		return "", fmt.Errorf("gh api %s %s: %w", method, endpoint, err)
	}
	var resp struct {
		HTMLURL string `json:"html_url"`
	}
	_ = json.Unmarshal(out, &resp)
	return resp.HTMLURL, nil
}

// CreatePR implements Backend.
func (g GitHub) CreatePR(ctx context.Context, r Repo, in NewPR) (string, error) {
	return g.writeURL(ctx, "POST", fmt.Sprintf("repos/%s/%s/pulls", r.Owner, r.Name),
		map[string]any{"title": in.Title, "body": in.Body, "head": in.Head, "base": in.Base, "draft": in.Draft})
}
