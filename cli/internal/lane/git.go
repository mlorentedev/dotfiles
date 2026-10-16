package lane

import "github.com/mlorentedev/dotfiles/cli/internal/worktree"

// ForgeGit is the worktree package's real git runner with one answer replaced:
// whether a branch's PR merged comes from the forge's REST endpoint, not from
// `gh pr list` (GraphQL) and not from ancestry. Ancestry reads a branch that
// has no commit yet as merged, since its tip is already in the base, and so
// would call a lane that has only just started Landed.
//
// The price is that a branch merged without a pull request reads as unmerged.
// That is the safe side for a reader: `worktree sweep` re-checks every gate
// itself before it removes anything, so a status that under-reports Landed
// costs a manual look and never a deleted worktree.
type ForgeGit struct {
	*worktree.RealGitRunner
	Forge PRSource
}

// IsPRMerged implements worktree.GitRunner.
func (g ForgeGit) IsPRMerged(_, branch string) (bool, error) {
	if branch == "" {
		return false, nil
	}
	pr, err := g.Forge.PRFor(branch)
	return pr.State == PRMerged, err
}
