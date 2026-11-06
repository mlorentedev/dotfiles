package worktree

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DetachedOptions configures a worktree checked out at a commit with no
// branch, for a tool rather than a person: `dotf spec review` runs its
// reviewer in one (HARNESS-142).
type DetachedOptions struct {
	RepoRoot     string
	WorktreeRoot string
	// Name is the directory under <WorktreeRoot>/<repo>/. It carries whatever
	// keeps two concurrent callers apart.
	Name    string
	Commit  string
	TTL     time.Duration
	Creator string
	Kind    string
}

// AddDetachedWithRunner creates the worktree and its lease. The lease holds no
// issue and is reapable, so the worktree never blocks `add --issue` and a
// caller that dies before removing it leaves something `sweep` can reap. No
// branch is made, fetched or checked: the caller names the commit.
func AddDetachedWithRunner(opts DetachedOptions, runner AddRunner, now time.Time) (*Info, error) {
	// Kind is required: a detached, issue-less worktree with no kind is one
	// nothing labels and no reap rule recognises.
	if opts.RepoRoot == "" || opts.Name == "" || opts.Commit == "" || opts.Kind == "" {
		return nil, errors.New("a detached worktree needs a repository, a name, a commit and a kind")
	}
	if opts.WorktreeRoot == "" {
		return nil, errors.New("WORKTREE_ROOT could not be resolved: run `dotf doctor`")
	}
	target := ResolveWorktreePath(opts.WorktreeRoot, filepath.Base(filepath.Clean(opts.RepoRoot)), opts.Name)
	if err := ValidateIsolation(opts.RepoRoot, target); err != nil {
		return nil, err
	}
	if _, err := os.Stat(target); err == nil {
		return nil, fmt.Errorf("target path already exists: %s", target)
	}
	if err := runner.WorktreeAddDetached(opts.RepoRoot, target, opts.Commit); err != nil {
		return nil, err
	}
	ttl := opts.TTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	meta := Metadata{
		Creator:        resolveCreator(opts.Creator),
		CreatedAt:      now,
		LeaseExpiresAt: now.Add(ttl),
		ReapOK:         true,
		Kind:           opts.Kind,
	}
	if err := SaveMetadata(target, meta); err != nil {
		return nil, fmt.Errorf("saving worktree metadata: %w", err)
	}
	ensureExclude(opts.RepoRoot, MetadataFileName)
	return &Info{Path: target, Head: opts.Commit, IsDetached: true, Metadata: &meta, State: StateActive}, nil
}

// AddDetached creates a detached worktree using RealAddRunner.
func AddDetached(opts DetachedOptions) (*Info, error) {
	return AddDetachedWithRunner(opts, &RealAddRunner{}, time.Now())
}
