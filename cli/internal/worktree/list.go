package worktree

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type GitRunner interface {
	WorktreeListPorcelain(repoRoot string) (string, error)
	IsDirty(worktreePath string) (bool, error)
	IsPRMerged(repoRoot, branch string) (bool, error)
}

type RealGitRunner struct{}

func (r *RealGitRunner) WorktreeListPorcelain(repoRoot string) (string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git worktree list: %w", err)
	}
	return string(out), nil
}

func (r *RealGitRunner) IsDirty(worktreePath string) (bool, error) {
	cmd := exec.Command("git", "-C", worktreePath, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}

func (r *RealGitRunner) IsPRMerged(repoRoot, branch string) (bool, error) {
	if branch == "" {
		return false, nil
	}
	// First check git merge-base --is-ancestor <branch> origin/main (or default branch)
	cmdBase := exec.Command("git", "-C", repoRoot, "rev-parse", "--abbrev-ref", "origin/HEAD")
	baseRef := "origin/main"
	if out, err := cmdBase.Output(); err == nil && len(strings.TrimSpace(string(out))) > 0 {
		baseRef = strings.TrimSpace(string(out))
	}

	ancestorCmd := exec.Command("git", "-C", repoRoot, "merge-base", "--is-ancestor", branch, baseRef)
	if ancestorCmd.Run() == nil {
		return true, nil
	}

	// Also check if GitHub PR is merged
	ghCmd := exec.Command("gh", "pr", "view", branch, "--repo", repoRoot, "--json", "state", "--jq", ".state")
	if out, err := ghCmd.Output(); err == nil {
		state := strings.TrimSpace(string(out))
		if state == "MERGED" {
			return true, nil
		}
	}

	return false, nil
}

type MockGitRunner struct {
	PorcelainOutput string
	DirtyPaths      map[string]bool
	MergedBranches  map[string]bool
}

func (m *MockGitRunner) WorktreeListPorcelain(repoRoot string) (string, error) {
	return m.PorcelainOutput, nil
}

func (m *MockGitRunner) IsDirty(worktreePath string) (bool, error) {
	return m.DirtyPaths[worktreePath], nil
}

func (m *MockGitRunner) IsPRMerged(repoRoot, branch string) (bool, error) {
	return m.MergedBranches[branch], nil
}

// SaveMetadata writes metadata to .dotf-worktree.json in target directory.
func SaveMetadata(worktreePath string, meta Metadata) error {
	metaPath := filepath.Join(worktreePath, MetadataFileName)
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}
	return os.WriteFile(metaPath, data, 0o644)
}

// ListWithRunner lists all worktrees for a repo using the provided runner.
func ListWithRunner(repoRoot string, runner GitRunner, now time.Time) ([]Info, error) {
	output, err := runner.WorktreeListPorcelain(repoRoot)
	if err != nil {
		return nil, err
	}

	raws, err := ParsePorcelain(output)
	if err != nil {
		return nil, err
	}

	absRepoRoot, _ := filepath.Abs(repoRoot)
	var list []Info

	for i, raw := range raws {
		// Ignore submodules (AC2)
		if raw.IsSubmodule {
			continue
		}

		absPath, _ := filepath.Abs(raw.Path)
		isMain := (i == 0) || (absPath == absRepoRoot)

		dirty, _ := runner.IsDirty(raw.Path)
		merged, _ := runner.IsPRMerged(repoRoot, raw.Branch)
		meta, _ := LoadMetadata(raw.Path)

		info := Info{
			Path:       raw.Path,
			Head:       raw.HEAD,
			Branch:     raw.Branch,
			IsBare:     raw.Bare,
			IsDetached: raw.Detached,
			IsMain:     isMain,
			Dirty:      dirty,
			PRMerged:   merged,
			Metadata:   meta,
		}

		info.State, info.StateReason = Classify(info, now)
		list = append(list, info)
	}

	return list, nil
}

// List scans all worktrees for a repository using RealGitRunner.
func List(repoRoot string) ([]Info, error) {
	return ListWithRunner(repoRoot, &RealGitRunner{}, time.Now())
}
