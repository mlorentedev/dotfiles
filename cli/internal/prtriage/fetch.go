package prtriage

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// ghPR mirrors what `gh pr list --json` returns. It is deliberately separate
// from PR: the vendor's shape is a boundary concern, and pinning the domain to
// it would make every test carry gh's nesting for no benefit.
type ghPR struct {
	Number   int    `json:"number"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Comments []struct {
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
		Body      string    `json:"body"`
		CreatedAt time.Time `json:"createdAt"`
	} `json:"comments"`
}

// Fetch asks GitHub for the open pull requests and returns the ones whose
// reviewer output is newer than their last recorded triage.
//
// It lives here rather than in the command because there are two callers and
// only one question. `dotf pr triage-queue` is the one an agent runs on purpose;
// the session-start brief is the one that runs whether or not anyone remembered
// — and a second copy of the gh call would be a second place for the domain
// conversion to drift.
//
// An empty repo means "the current repository", matching gh's own default. The
// context bounds the gh call: this runs on the session-start path, where an
// unreachable API must cost a bounded wait and a visible message rather than a
// hung shell.
//
// Every failure is returned, never softened into an empty queue. That rule is
// the entire point of this package: a queue that could not be computed must not
// read as a queue with nothing in it.
func Fetch(ctx context.Context, repo, registryPath string) ([]Status, error) {
	reg, err := LoadRegistry(registryPath)
	if err != nil {
		return nil, err
	}
	return FetchWithRegistry(ctx, repo, reg)
}

// FetchWithRegistry asks GitHub for open PRs using a pre-loaded Registry,
// avoiding redundant disk reads when the caller already has the registry.
func FetchWithRegistry(ctx context.Context, repo string, reg Registry) ([]Status, error) {
	const prLimit = 100
	args := []string{"pr", "list", "--state", "open", "--limit", fmt.Sprintf("%d", prLimit),
		"--json", "number,title,url,comments"}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	out, err := exec.CommandContext(ctx, "gh", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("gh pr list: %w", err)
	}

	var wire []ghPR
	if err := json.Unmarshal(out, &wire); err != nil {
		return nil, fmt.Errorf("parse gh output: %w", err)
	}

	if len(wire) >= prLimit {
		return nil, fmt.Errorf("gh pr list returned %d open PRs (hit boundary limit); queue may be truncated — investigate with gh pr list", prLimit)
	}

	prs := make([]PR, 0, len(wire))
	for _, w := range wire {
		p := PR{Number: w.Number, Title: w.Title, URL: w.URL}
		for _, cm := range w.Comments {
			p.Comments = append(p.Comments, Comment{
				Author: cm.Author.Login, Body: cm.Body, CreatedAt: cm.CreatedAt,
			})
		}
		prs = append(prs, p)
	}

	return Queue(prs, reg), nil
}

// Marker returns the triage heading a registry declares, for a caller that needs
// to name it in a message. It re-reads the registry rather than caching: this is
// invoked once per session, and a stale marker in a long-lived process would be
// the same two-file disagreement the registry exists to prevent.
func Marker(registryPath string) (string, error) {
	reg, err := LoadRegistry(registryPath)
	if err != nil {
		return "", err
	}
	return reg.Triage.Marker, nil
}
