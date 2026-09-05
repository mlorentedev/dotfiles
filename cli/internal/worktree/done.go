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
	Force        bool
}

// Done tears down a completed worktree safely.
func Done(opts DoneOptions) error {
	if opts.RepoRoot == "" {
		return fmt.Errorf("repository root cannot be empty")
	}

	wtPath := opts.WorktreePath
	if wtPath == "" {
		return fmt.Errorf("worktree path cannot be empty")
	}

	absRepo, err := filepath.Abs(opts.RepoRoot)
	if err != nil {
		return err
	}
	absWT, err := filepath.Abs(wtPath)
	if err != nil {
		return err
	}

	if absRepo == absWT {
		return fmt.Errorf("cannot remove main repository: %s", absRepo)
	}

	// Check if dirty
	statusCmd := exec.Command("git", "-C", absWT, "status", "--porcelain")
	if out, err := statusCmd.Output(); err == nil && len(strings.TrimSpace(string(out))) > 0 {
		if !opts.Force {
			return fmt.Errorf("worktree has uncommitted changes (%s); commit, stash, or pass --force", absWT)
		}
	}

	// Remove worktree
	args := []string{"-C", absRepo, "worktree", "remove", absWT}
	if opts.Force {
		args = append(args, "--force")
	}

	cmd := exec.Command("git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove failed (%w): %s", err, strings.TrimSpace(string(out)))
	}

	_ = exec.Command("git", "-C", absRepo, "worktree", "prune").Run()
	return nil
}
