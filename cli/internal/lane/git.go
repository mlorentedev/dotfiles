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
func (g ForgeGit) IsPRMerged(_, branch string) (bool, error) { return forgeMerged(g.Forge, branch) }

// ForgeSweep is the sweep runner that gives the same answer ForgeGit gives.
// status and plan decide what is reapable from the forge, so the sweep apply
// runs must too: left on its own it asks ancestry, and reaps a worktree whose
// branch has no commit yet, which plan handed to a human as ambiguous.
type ForgeSweep struct {
	*worktree.RealSweepRunner
	Forge PRSource
}

// IsPRMerged implements worktree.GitRunner.
func (g ForgeSweep) IsPRMerged(_, branch string) (bool, error) { return forgeMerged(g.Forge, branch) }

func forgeMerged(forge PRSource, branch string) (bool, error) {
	if branch == "" {
		return false, nil
	}
	pr, err := forge.PRFor(branch)
	return pr.State == PRMerged, err
}
