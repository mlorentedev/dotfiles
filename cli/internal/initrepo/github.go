package initrepo

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// repoSlugPattern matches a GitHub "owner/name" slug — the grammar
// init-repo-github-defaults.sh validates the derived target against.
var repoSlugPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// ParseOriginRepo extracts the "owner/name" slug from a GitHub remote URL in
// either SSH (git@github.com:owner/name[.git]) or HTTPS/SSH-URL
// (https://github.com/owner/name[.git], ssh://git@github.com/owner/name) form.
// It is the Go twin of the script's sed pipeline, and errors for non-github.com
// remotes or anything that does not reduce to a clean owner/name.
func ParseOriginRepo(remoteURL string) (string, error) {
	s := strings.TrimSpace(remoteURL)
	s = strings.TrimSuffix(s, ".git")

	switch {
	case strings.HasPrefix(s, "git@github.com:"):
		s = strings.TrimPrefix(s, "git@github.com:")
	case strings.Contains(s, "github.com/"):
		s = s[strings.Index(s, "github.com/")+len("github.com/"):]
	default:
		return "", fmt.Errorf("not a github.com remote: %q", remoteURL)
	}

	if !repoSlugPattern.MatchString(s) {
		return "", fmt.Errorf("could not derive owner/name from %q (got %q)", remoteURL, s)
	}
	return s, nil
}

// ValidRepoSlug reports whether s is a well-formed "owner/name".
func ValidRepoSlug(s string) bool { return repoSlugPattern.MatchString(s) }

// OriginRepo resolves repoRoot's origin remote to an "owner/name" slug. It errors
// when there is no origin remote (the caller turns that into a graceful skip).
func OriginRepo(repoRoot string) (string, error) {
	out, err := exec.Command("git", "-C", repoRoot, "remote", "get-url", "origin").Output()
	if err != nil {
		return "", fmt.Errorf("no origin remote in %s", repoRoot)
	}
	return ParseOriginRepo(string(out))
}

// GitHubResult reports the outcome of applying the GitHub repo defaults.
type GitHubResult struct {
	Repo   string
	Action string // "skipped" | "already-enabled" | "dry-run" | "enabled"
	Reason string // populated for "skipped"
}

// ApplyDeleteBranchOnMerge ensures delete_branch_on_merge=true on the given
// owner/name repo via `gh api`, mirroring init-repo-github-defaults.sh: query the
// current state, no-op if already enabled, honor dry-run, otherwise PATCH.
//
// Host-coupling degrades gracefully (ADR-022 C7): if gh is absent or the repo
// state can't be read (unauthenticated / offline / missing repo), it returns a
// "skipped" result with err==nil so the orchestrator never aborts. A genuine
// failure — an attempted PATCH the server rejects (permissions, archived repo) —
// returns an error, which the orchestrator wraps non-fatally.
func ApplyDeleteBranchOnMerge(repo string, dryRun bool) (GitHubResult, error) {
	res := GitHubResult{Repo: repo}

	if _, err := exec.LookPath("gh"); err != nil {
		res.Action = "skipped"
		res.Reason = "gh CLI not found (install https://cli.github.com/, then re-run)"
		return res, nil
	}

	enabled, err := ghDeleteBranchOnMerge(repo)
	if err != nil {
		res.Action = "skipped"
		res.Reason = "could not query repo via gh (not authenticated, offline, or repo not found)"
		return res, nil
	}
	if enabled {
		res.Action = "already-enabled"
		return res, nil
	}
	if dryRun {
		res.Action = "dry-run"
		return res, nil
	}
	if err := ghEnableDeleteBranchOnMerge(repo); err != nil {
		return res, fmt.Errorf("PATCH delete_branch_on_merge on %s failed "+
			"(insufficient permissions, or repo archived?): %w", repo, err)
	}
	res.Action = "enabled"
	return res, nil
}

// ghDeleteBranchOnMerge reads the current delete_branch_on_merge flag for repo.
func ghDeleteBranchOnMerge(repo string) (bool, error) {
	out, err := exec.Command("gh", "api", "/repos/"+repo,
		"--jq", ".delete_branch_on_merge").Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) == "true", nil
}

// ghEnableDeleteBranchOnMerge PATCHes delete_branch_on_merge=true on repo.
func ghEnableDeleteBranchOnMerge(repo string) error {
	return exec.Command("gh", "api", "-X", "PATCH", "/repos/"+repo,
		"-f", "delete_branch_on_merge=true").Run()
}
