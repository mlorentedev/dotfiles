package worktree

import (
	"fmt"
	"time"
)

// AdoptOptions names a linked worktree created without dotf (by herdr, or by
// git directly) that should get a lease.
type AdoptOptions struct {
	RepoRoot     string
	WorktreePath string
	TTL          time.Duration // 0 means 24h, as for add
	Creator      string
	Issue        int
}

// AdoptResult says whether Adopt wrote a lease or found one.
type AdoptResult struct {
	Metadata       Metadata
	AlreadyAdopted bool
}

// Adopt writes the lease add would have written, so sweep may reap the
// worktree once it is merged, clean and past the lease. Missing metadata stays
// a refusal everywhere else (lesson 272): adoption is the one explicit way in.
// A worktree that already has a lease keeps it, so adopting twice changes
// nothing.
func Adopt(opts AdoptOptions, now time.Time) (AdoptResult, error) {
	wt, err := linkedWorktreeOf(opts.RepoRoot, opts.WorktreePath)
	if err != nil {
		return AdoptResult{}, err
	}
	existing, err := LoadMetadata(wt)
	if err != nil {
		return AdoptResult{}, fmt.Errorf("reading the lease of %s: %w", wt, err)
	}
	if existing != nil {
		return AdoptResult{Metadata: *existing, AlreadyAdopted: true}, nil
	}
	ttl := opts.TTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	meta := Metadata{
		Creator:        resolveCreator(opts.Creator),
		Issue:          opts.Issue,
		CreatedAt:      now,
		LeaseExpiresAt: now.Add(ttl),
		ReapOK:         true,
	}
	if err := SaveMetadata(wt, meta); err != nil {
		return AdoptResult{}, fmt.Errorf("writing the lease of %s: %w", wt, err)
	}
	ensureExclude(opts.RepoRoot, MetadataFileName)
	return AdoptResult{Metadata: meta}, nil
}

// linkedWorktreeOf returns path's worktree top when it is a linked worktree of
// the repository at repoRoot, and refuses the main checkout, a plain directory
// and another repository's worktree.
func linkedWorktreeOf(repoRoot, path string) (string, error) {
	top, err := ResolveWorktreeRoot(path)
	if err != nil {
		return "", fmt.Errorf("%s is not a git worktree", path)
	}
	main, err := ResolveMainRepoRoot(top)
	if err != nil {
		return "", fmt.Errorf("%s: %w", top, err)
	}
	switch {
	case resolvedPath(main) != resolvedPath(repoRoot):
		return "", fmt.Errorf("%s belongs to %s, not %s", top, main, repoRoot)
	case resolvedPath(top) == resolvedPath(main):
		return "", fmt.Errorf("%s is the main checkout, not a linked worktree", top)
	}
	return top, nil
}
