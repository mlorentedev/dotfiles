package lane

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// leaseTTL is the lease dotf worktree add gives a worktree when no --ttl is
// passed (worktree.resolveAddDefaults); a lane worktree is held to the same.
const leaseTTL = 24 * time.Hour

// ensureLease gives a lane worktree the metadata dotf worktree add writes, so
// `dotf worktree list` shows it with a lease and `sweep` can reap it once its
// PR lands (#2547). A lease already there is kept: a rerun must not move
// created_at. The agent goes in Creator until the metadata carries intent
// fields of its own (CLI-118, #2503); the branch is git's to answer.
func ensureLease(o Options, path string, issue int, agent string) error {
	if path == "" {
		return errors.New("lane: the session backend named no worktree path, so no lease can be written; no agent started")
	}
	if err := excludeFile(o.GitDir, worktree.MetadataFileName); err != nil {
		return err
	}
	meta, err := worktree.LoadMetadata(path)
	if err != nil || meta != nil {
		return err
	}
	now := time.Now()
	if o.Now != nil {
		now = o.Now()
	}
	lease := worktree.Metadata{Creator: agent, Issue: issue, CreatedAt: now, LeaseExpiresAt: now.Add(leaseTTL), ReapOK: true}
	if err := worktree.SaveMetadata(path, lease); err != nil {
		return fmt.Errorf("lane: writing the lease: %w", err)
	}
	return nil
}

// excludeFile keeps a file a lane leaves in its worktrees (the lease, the
// prompt) out of `git status` in every worktree of the repo, as dotf worktree
// add does for the lease. The worktree package keeps its own helper
// unexported, so this is the one line of the same rule, checked first so a
// rerun appends nothing.
func excludeFile(gitDir, name string) error {
	if gitDir == "" {
		return fmt.Errorf("lane: no git directory to exclude %s in", name)
	}
	path := filepath.Join(gitDir, "info", "exclude")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("lane: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == name {
			return nil
		}
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	data = append(data, name+"\n"...)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("lane: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}
