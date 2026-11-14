package forgewrite

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
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

// readyMutation marks a draft ready. REST can open a pull request as a draft
// but cannot clear the flag; this GraphQL mutation is the only call that does,
// and the one `gh pr ready` sends.
const readyMutation = `mutation($id: ID!) { markPullRequestReadyForReview(input: {pullRequestId: $id}) { pullRequest { isDraft } } }`

// ReadyPR implements Backend. The pull request is read over REST for its node
// id; a ready one gets no mutation.
func (g GitHub) ReadyPR(ctx context.Context, r Repo, number int) (string, error) {
	endpoint := fmt.Sprintf("repos/%s/%s/pulls/%d", r.Owner, r.Name, number)
	out, err := g.Run(ctx, nil, "api", endpoint)
	if err != nil {
		return "", fmt.Errorf("gh api %s: %w", endpoint, err)
	}
	var pr struct {
		NodeID  string `json:"node_id"`
		Draft   bool   `json:"draft"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(out, &pr); err != nil {
		return "", fmt.Errorf("ERROR: gh api %s answered something that is not JSON. WHY: %w. FIX: run the same gh api call by hand and read its answer", endpoint, err)
	}
	if pr.NodeID == "" {
		return "", fmt.Errorf("ERROR: gh api %s answered without a node_id. WHY: the answer is not a pull request. FIX: check the number names a pull request, not an issue", endpoint)
	}
	if !pr.Draft {
		return pr.HTMLURL, nil
	}
	out, err = g.Run(ctx, nil, "api", "graphql", "-f", "query="+readyMutation, "-f", "id="+pr.NodeID)
	if err != nil {
		return "", fmt.Errorf("gh api graphql markPullRequestReadyForReview: %w", err)
	}
	var resp struct {
		Data struct {
			Mark struct {
				PullRequest struct {
					IsDraft *bool `json:"isDraft"`
				} `json:"pullRequest"`
			} `json:"markPullRequestReadyForReview"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &resp); err != nil || resp.Data.Mark.PullRequest.IsDraft == nil || *resp.Data.Mark.PullRequest.IsDraft {
		return "", fmt.Errorf("ERROR: %s may still be a draft. WHY: the answer to markPullRequestReadyForReview did not say isDraft: false. FIX: check the pull request and rerun dotf forge pr ready", pr.HTMLURL)
	}
	return pr.HTMLURL, nil
}

// LandedHeads implements Backend. It reads the REST endpoint, as lane.REST
// does, not `gh pr list`: that is GraphQL, whose quota a fleet of worktrees
// drains. head=owner:branch names an account, not a repository, so a fork the
// same owner holds would match too: the head repository is compared as well.
// --paginate prints one JSON array per page, read here one after another.
func (g GitHub) LandedHeads(ctx context.Context, r Repo, branch string) ([]string, error) {
	endpoint := fmt.Sprintf("repos/%s/%s/pulls?state=closed&per_page=100&head=%s:%s", r.Owner, r.Name, r.Owner, queryBranch(branch))
	out, err := g.Run(ctx, nil, "api", "--paginate", endpoint)
	if err != nil {
		return nil, fmt.Errorf("gh api %s: %w", endpoint, err)
	}
	var heads []string
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var pulls []struct {
			MergedAt *string `json:"merged_at"`
			Head     struct {
				SHA  string `json:"sha"`
				Repo *struct {
					FullName string `json:"full_name"`
				} `json:"repo"`
			} `json:"head"`
		}
		if err := dec.Decode(&pulls); errors.Is(err, io.EOF) {
			return heads, nil
		} else if err != nil {
			return nil, fmt.Errorf("gh api %s: unreadable answer: %w", endpoint, err)
		}
		for _, p := range pulls {
			if p.MergedAt != nil && p.Head.Repo != nil && strings.EqualFold(p.Head.Repo.FullName, r.Owner+"/"+r.Name) {
				heads = append(heads, p.Head.SHA)
			}
		}
	}
}

// queryBranch encodes a branch for a query string, as lane.queryBranch does
// (forgewrite cannot import lane). git accepts & # and { in a ref name; left
// raw, the first adds a parameter, the second cuts the query, and gh reads the
// third as a placeholder. The slash stays: it is legal in a query.
func queryBranch(branch string) string {
	return strings.ReplaceAll(url.QueryEscape(branch), "%2F", "/")
}
