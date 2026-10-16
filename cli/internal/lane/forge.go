package lane

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// REST reads pull requests through the gh CLI's REST endpoint, in the
// repository of the working directory it runs in. `gh pr list` is GraphQL, and
// its quota is the one a fleet of worktrees would drain.
type REST struct{ Run Runner }

// PRFor implements PRSource. The newest pull request of the branch decides: a
// closed attempt followed by an open retry is Review, not Stalled. A forge
// that cannot answer is an error and never "no pull request", which would
// read a lane as Stalled.
func (r REST) PRFor(branch string) (PRInfo, error) {
	// A worktree on a detached HEAD has no branch, and `head={owner}:` with
	// nothing after it matches every pull request: the newest one would be
	// read as this worktree's own.
	if branch == "" {
		return PRInfo{}, nil
	}
	out, err := r.Run("gh", "api", "repos/{owner}/{repo}/pulls?state=all&sort=created&direction=desc&per_page=1&head={owner}:"+queryBranch(branch))
	if err != nil {
		return PRInfo{}, err
	}
	var pulls []struct {
		Number   int     `json:"number"`
		State    string  `json:"state"`
		MergedAt *string `json:"merged_at"`
		Draft    bool    `json:"draft"`
	}
	if err := json.Unmarshal([]byte(out), &pulls); err != nil {
		return PRInfo{}, fmt.Errorf("gh api pulls head=%s: %w", branch, err)
	}
	if len(pulls) == 0 {
		return PRInfo{}, nil
	}
	p := pulls[0]
	switch {
	case p.MergedAt != nil:
		return PRInfo{Number: p.Number, State: PRMerged}, nil
	case p.State == "open":
		return PRInfo{Number: p.Number, State: PROpen, Draft: p.Draft}, nil
	}
	return PRInfo{Number: p.Number, State: PRClosed}, nil
}

// queryBranch encodes a branch for a query string. git accepts & # and { in a
// ref name; left raw, the first would add a parameter, the second would cut
// the query, and gh would read the third as one of its own placeholders. The
// slash stays, as it is legal in a query and a branch is full of them.
func queryBranch(branch string) string {
	return strings.ReplaceAll(url.QueryEscape(branch), "%2F", "/")
}
