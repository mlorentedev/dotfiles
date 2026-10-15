package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// MigrateOptions moves a repository's legacy <repo>-wt-<slug> siblings under
// the worktree root.
type MigrateOptions struct {
	RepoRoot     string
	WorktreeRoot string
	DryRun       bool
	// Cwd and ClaudeProjectsDir feed the same live-session refusals done
	// applies (#1653): a worktree a session lives in is skipped, never moved.
	Cwd               string
	ClaudeProjectsDir string
}

// Move is one legacy worktree and where it goes. Skip is empty when it is (or,
// in a dry run, would be) moved, and says why otherwise.
type Move struct {
	From string `json:"from"`
	To   string `json:"to"`
	Skip string `json:"skip,omitempty"`
}

// MigrateReport lists every legacy worktree found; none means nothing to do.
type MigrateReport struct {
	Moves  []Move `json:"moves"`
	DryRun bool   `json:"dry_run"`
}

// Migrate moves each <repo>-wt-* sibling to <root>/<repo>/<branch-slug> with
// git worktree move, so git's own records follow the directory. It skips, and
// names why, a worktree that is locked, that holds the caller or a recent
// Claude Code session, or whose target exists. A moved worktree is no longer
// legacy, so a second run finds nothing.
func Migrate(opts MigrateOptions, now time.Time) (*MigrateReport, error) {
	if opts.WorktreeRoot == "" {
		return nil, fmt.Errorf("WORKTREE_ROOT could not be resolved: run `dotf doctor`")
	}
	out, err := (&RealGitRunner{}).WorktreeListPorcelain(opts.RepoRoot)
	if err != nil {
		return nil, err
	}
	raws, err := ParsePorcelain(out)
	if err != nil {
		return nil, err
	}
	repoDir := resolvedPath(opts.RepoRoot)
	repo := filepath.Base(repoDir)
	prefix := filepath.Join(filepath.Dir(repoDir), repo+"-wt-")
	report := &MigrateReport{DryRun: opts.DryRun}
	for _, raw := range raws {
		from := resolvedPath(raw.Path)
		if !strings.HasPrefix(from, prefix) || filepath.Dir(from) != filepath.Dir(repoDir) {
			continue
		}
		name := raw.Branch
		if name == "" {
			name = strings.TrimPrefix(from, prefix) // detached: keep the slug
		}
		m := Move{From: from, To: ResolveWorktreePath(opts.WorktreeRoot, repo, name)}
		m.Skip = migrateSkip(raw, m, opts, now)
		if m.Skip == "" && !opts.DryRun {
			if err := moveWorktree(opts.RepoRoot, m); err != nil {
				m.Skip = err.Error()
			}
		}
		report.Moves = append(report.Moves, m)
	}
	return report, nil
}

func migrateSkip(raw RawWorktree, m Move, opts MigrateOptions, now time.Time) string {
	if raw.Locked {
		return "locked (git worktree unlock it first)"
	}
	if err := checkCallerOutside(m.From, opts.Cwd); err != nil {
		return err.Error()
	}
	if err := checkNoRecentSession(opts.ClaudeProjectsDir, m.From, now); err != nil {
		return err.Error()
	}
	if _, err := os.Stat(m.To); err == nil {
		return fmt.Sprintf("the target %s already exists", m.To)
	}
	return ""
}

func moveWorktree(repoRoot string, m Move) error {
	if err := os.MkdirAll(filepath.Dir(m.To), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(m.To), err)
	}
	if out, err := exec.Command("git", "-C", repoRoot, "worktree", "move", m.From, m.To).CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree move failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}
