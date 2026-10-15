package worktree

import (
	"path/filepath"
	"slices"
	"strings"
)

// ResolveWorktreePath returns where a worktree for branch lives under the one
// worktree root: <root>/<repo>/<slug>, the layout herdr creates and the only
// one it can be configured for (its `worktrees.directory` sets the root, not
// the shape). The slug follows herdr 0.9.3's rule, measured on #2503: lowercase,
// every run of characters outside [a-z0-9] becomes one "-", trimmed at both ends.
func ResolveWorktreePath(root, repo, branch string) string {
	return filepath.Join(root, repo, slugifyBranch(branch))
}

// DefaultBranch is the branch add creates for slug when none is given.
func DefaultBranch(slug string) string { return "feat/" + slug }

// SlugCandidates lists every path a bare slug may name for the repository at
// repoRoot, without checking which exist: the legacy <repo>-wt-<slug> sibling,
// then, under worktreeRoot, the slug read as a branch and the default branch
// add would have made for it. Duplicates are dropped, order is kept.
func SlugCandidates(repoRoot, worktreeRoot, slug string) []string {
	out := []string{ResolveSiblingPath(repoRoot, slug)}
	if worktreeRoot != "" {
		repo := filepath.Base(filepath.Clean(repoRoot))
		for _, branch := range []string{slug, DefaultBranch(slug)} {
			if p := ResolveWorktreePath(worktreeRoot, repo, branch); !slices.Contains(out, p) {
				out = append(out, p)
			}
		}
	}
	return out
}

func slugifyBranch(branch string) string {
	var b strings.Builder
	pendingDash := false
	for _, r := range strings.ToLower(branch) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if pendingDash && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingDash = false
			b.WriteRune(r)
			continue
		}
		pendingDash = true
	}
	return b.String()
}
