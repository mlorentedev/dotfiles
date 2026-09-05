package worktree

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type DoneOptions struct {
	RepoRoot     string
	WorktreePath string
	LockPath     string
	Force        bool
}

// Done tears down a completed worktree safely.
func Done(opts DoneOptions) error {
	absRepo, absWT, err := validateDoneOptions(opts)
	if err != nil {
		return err
	}

	lockPath := opts.LockPath
	if lockPath == "" {
		lockPath = DefaultLockPath()
	}
	unlock, err := TryLockFile(lockPath)
	if err != nil {
		return fmt.Errorf("acquiring worktree lock: %w", err)
	}
	defer unlock()

	if err := checkUncommittedChanges(absWT, opts.Force); err != nil {
		return err
	}
	if err := checkUnpushedCommits(absRepo, absWT, opts.Force); err != nil {
		return err
	}
	return removeWorktree(absRepo, absWT, opts.Force)
}

func validateDoneOptions(opts DoneOptions) (string, string, error) {
	if opts.RepoRoot == "" {
		return "", "", fmt.Errorf("repository root cannot be empty")
	}
	if opts.WorktreePath == "" {
		return "", "", fmt.Errorf("worktree path cannot be empty")
	}
	absRepo, err := filepath.Abs(opts.RepoRoot)
	if err != nil {
		return "", "", err
	}
	absWT, err := filepath.Abs(opts.WorktreePath)
	if err != nil {
		return "", "", err
	}
	if absRepo == absWT {
		return "", "", fmt.Errorf("cannot remove main repository: %s", absRepo)
	}
	return absRepo, absWT, nil
}

func checkUncommittedChanges(absWT string, force bool) error {
	if force {
		return nil
	}
	out, err := exec.Command("git", "-C", absWT, "status", "--porcelain").Output()
	if err != nil {
		return fmt.Errorf("git status failed (%w); pass --force to override", err)
	}
	if len(strings.TrimSpace(string(out))) > 0 {
		return fmt.Errorf("worktree has uncommitted changes (%s); commit, stash, or pass --force", absWT)
	}
	return nil
}

func checkUnpushedCommits(absRepo, absWT string, force bool) error {
	if force {
		return nil
	}
	// Check if upstream tracking branch is configured
	upstreamCmd := exec.Command("git", "-C", absWT, "rev-parse", "--symbolic-full-name", "@{u}")
	if err := upstreamCmd.Run(); err == nil {
		count, err := countRevList(absWT, "@{u}..HEAD")
		if err != nil {
			return fmt.Errorf("verifying upstream commits failed: %w", err)
		}
		if count > 0 {
			return fmt.Errorf("worktree has %d unpushed commit(s); push before done, or pass --force", count)
		}
		return nil
	}

	// No upstream configured: check against base branch
	baseRef := resolveBaseRef(absRepo)
	if baseRef == "" {
		return fmt.Errorf("branch has no upstream configured and base branch cannot be resolved; pass --force to remove")
	}

	count, err := countRevList(absWT, baseRef+"..HEAD")
	if err != nil {
		return fmt.Errorf("verifying unpushed commits against %s failed: %w; pass --force to override", baseRef, err)
	}
	if count > 0 {
		return fmt.Errorf("branch has no upstream configured and has %d unpushed commit(s) ahead of %s; push before done, or pass --force", count, baseRef)
	}
	return nil
}

func countRevList(dir, revRange string) (int, error) {
	out, err := exec.Command("git", "-C", dir, "rev-list", revRange, "--count").Output()
	if err != nil {
		return 0, err
	}
	countStr := strings.TrimSpace(string(out))
	var count int
	if _, err := fmt.Sscanf(countStr, "%d", &count); err != nil {
		return 0, fmt.Errorf("parsing rev-list count %q: %w", countStr, err)
	}
	return count, nil
}

func resolveBaseRef(repoDir string) string {
	candidates := []string{"origin/HEAD", "origin/main", "origin/master", "main", "master"}
	for _, ref := range candidates {
		if exec.Command("git", "-C", repoDir, "rev-parse", "--verify", ref).Run() == nil {
			return ref
		}
	}
	return ""
}

func removeWorktree(absRepo, absWT string, force bool) error {
	args := []string{"-C", absRepo, "worktree", "remove", absWT}
	if force {
		args = append(args, "--force")
	}
	cmd := exec.Command("git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove failed (%w): %s", err, strings.TrimSpace(string(out)))
	}
	_ = exec.Command("git", "-C", absRepo, "worktree", "prune").Run()
	return nil
}
