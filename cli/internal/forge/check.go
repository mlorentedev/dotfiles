package forge

import (
	"fmt"
	"sort"
	"strings"
)

// RepoStatus grades one declared repository against the live forge.
type RepoStatus string

const (
	StatusOK           RepoStatus = "ok"           // live matches the declared object
	StatusDrift        RepoStatus = "drift"        // live differs; Changes says where
	StatusState        RepoStatus = "state"        // declared unavailable/unprotected, and not contradicted
	StatusUnanswerable RepoStatus = "unanswerable" // the forge could not be asked
)

// RepoResult is the verdict on one repository.
type RepoResult struct {
	Repo    string
	Status  RepoStatus
	Changes []Change
	Detail  string
}

// Runner runs `gh` with args and returns stdout, stderr and the exit error.
type Runner func(args ...string) (stdout, stderr string, err error)

// CheckAll grades every declared repository, sorted by name.
func CheckAll(d Declaration, run Runner) []RepoResult {
	repos := make([]string, 0, len(d.Repos))
	for r := range d.Repos {
		repos = append(repos, r)
	}
	sort.Strings(repos)
	results := make([]RepoResult, 0, len(repos))
	for _, r := range repos {
		results = append(results, CheckRepo(r, d.Repos[r], run))
	}
	return results
}

// NeedsAttention reports whether any result is drift or unanswerable. An
// unanswerable check counts: a question nobody could answer must not read as
// a clean one — the `dotf pr triage-queue` contract.
func NeedsAttention(results []RepoResult) bool {
	for _, r := range results {
		if r.Status == StatusDrift || r.Status == StatusUnanswerable {
			return true
		}
	}
	return false
}

// CheckRepo compares one declaration with the live protection of its branch.
func CheckRepo(repo string, d RepoDecl, run Runner) RepoResult {
	res := RepoResult{Repo: repo}
	if d.State == StateUnavailable {
		// Asking would only reproduce the 403 the declaration already explains.
		res.Status, res.Detail = StatusState, "declared unavailable: "+d.Reason
		return res
	}
	out, errOut, err := run("api", fmt.Sprintf("repos/%s/branches/%s/protection", repo, d.Branch))
	notProtected := err != nil && strings.Contains(errOut, "HTTP 404") && strings.Contains(errOut, "Branch not protected")
	switch {
	case err != nil && !notProtected:
		res.Status, res.Detail = StatusUnanswerable, unanswerable(errOut, err)
	case notProtected && d.State == StateUnprotected:
		res.Status, res.Detail = StatusState, "unprotected, as declared: "+d.Reason
	case notProtected:
		res.Status = StatusDrift
		res.Changes = []Change{{Field: "protection", Declared: "required", Live: "none"}}
	case d.State == StateUnprotected:
		res.Status = StatusDrift
		res.Changes = []Change{{Field: "protection", Declared: "none", Live: "required"}}
	default:
		res = compareLive(res, d, out)
	}
	return res
}

func compareLive(res RepoResult, d RepoDecl, body string) RepoResult {
	live, err := Normalise([]byte(body))
	if err != nil {
		res.Status, res.Detail = StatusUnanswerable, err.Error()
		return res
	}
	if res.Changes = Diff(*d.Protection, live); len(res.Changes) > 0 {
		res.Status = StatusDrift
	} else {
		res.Status = StatusOK
	}
	return res
}

func unanswerable(errOut string, err error) string {
	line, _, _ := strings.Cut(strings.TrimSpace(errOut), "\n")
	if line == "" {
		line = err.Error()
	}
	if strings.Contains(errOut, "HTTP 403") {
		return line + " — if this is a private repository on the free plan, declare state " + StateUnavailable + " with that reason"
	}
	return line
}
