package worktree

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type GitRunner interface {
	WorktreeListPorcelain(repoRoot string) (string, error)
	IsDirty(worktreePath string) (bool, error)
	IsPRMerged(repoRoot, branch string) (bool, error)
	IsOrphan(repoRoot, branch string) (bool, error)
}

type RealGitRunner struct {
	prCacheMu sync.Mutex
	prCache   map[string]bool
}

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
	if len(strings.TrimSpace(string(out))) > 0 {
		return true, nil
	}

	// Fail-closed data loss guard: check for non-disposable gitignored local content (e.g. .env, scratchpad notes)
	ignoredCmd := exec.Command("git", "-C", worktreePath, "status", "--ignored", "--porcelain")
	ignoredOut, err := ignoredCmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status --ignored: %w", err)
	}
	return HasNonDisposableIgnored(string(ignoredOut)), nil
}

// ParseGitHubSlug extracts owner/repo from SSH or HTTPS GitHub URLs.
func ParseGitHubSlug(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	rawURL = strings.TrimSuffix(rawURL, ".git")
	if strings.HasPrefix(rawURL, "git@github.com:") {
		return strings.TrimPrefix(rawURL, "git@github.com:")
	}
	if idx := strings.Index(rawURL, "github.com/"); idx != -1 {
		return rawURL[idx+len("github.com/"):]
	}
	return ""
}

func (r *RealGitRunner) IsPRMerged(repoRoot, branch string) (bool, error) {
	if branch == "" {
		return false, nil
	}

	r.prCacheMu.Lock()
	if r.prCache != nil {
		if merged, ok := r.prCache[branch]; ok {
			r.prCacheMu.Unlock()
			return merged, nil
		}
	}
	r.prCacheMu.Unlock()

	// Fast path: check local git merge-base --is-ancestor <branch> origin/HEAD
	if isBranchAncestor(repoRoot, branch) {
		r.cachePRResult(branch, true)
		return true, nil
	}

	merged := queryGHPRMerged(repoRoot, branch)
	r.cachePRResult(branch, merged)
	return merged, nil
}

func isBranchAncestor(repoRoot, branch string) bool {
	baseRef := resolveBaseRef(repoRoot)
	if baseRef == "" {
		return false
	}
	ancestorCmd := exec.Command("git", "-C", repoRoot, "merge-base", "--is-ancestor", branch, baseRef)
	return ancestorCmd.Run() == nil
}

func queryGHPRMerged(repoRoot, branch string) bool {
	slug := ""
	if out, err := exec.Command("git", "-C", repoRoot, "config", "--get", "remote.origin.url").Output(); err == nil {
		slug = ParseGitHubSlug(string(out))
	}

	var ghCmd *exec.Cmd
	if slug != "" {
		ghCmd = exec.Command("gh", "pr", "view", branch, "--repo", slug, "--json", "state", "--jq", ".state")
	} else {
		ghCmd = exec.Command("gh", "pr", "view", branch, "--json", "state", "--jq", ".state")
		ghCmd.Dir = repoRoot
	}

	if out, err := ghCmd.Output(); err == nil {
		state := strings.TrimSpace(string(out))
		return state == "MERGED"
	}
	return false
}

func (r *RealGitRunner) cachePRResult(branch string, merged bool) {
	r.prCacheMu.Lock()
	defer r.prCacheMu.Unlock()
	if r.prCache == nil {
		r.prCache = make(map[string]bool)
	}
	r.prCache[branch] = merged
}

func (r *RealGitRunner) IsOrphan(repoRoot, branch string) (bool, error) {
	if branch == "" {
		return false, nil
	}
	cmd := exec.Command("git", "-C", repoRoot, "rev-parse", "--symbolic-full-name", branch+"@{u}")
	if err := cmd.Run(); err != nil {
		cfgCmd := exec.Command("git", "-C", repoRoot, "config", "--get", "branch."+branch+".remote")
		if out, cfgErr := cfgCmd.Output(); cfgErr == nil && len(strings.TrimSpace(string(out))) > 0 {
			return true, nil
		}
	}
	return false, nil
}

type MockGitRunner struct {
	PorcelainOutput string
	DirtyPaths      map[string]bool
	DirtyErrors     map[string]error
	MergedBranches  map[string]bool
	OrphanBranches  map[string]bool
}

func (m *MockGitRunner) WorktreeListPorcelain(repoRoot string) (string, error) {
	return m.PorcelainOutput, nil
}

func (m *MockGitRunner) IsDirty(worktreePath string) (bool, error) {
	if m.DirtyErrors != nil && m.DirtyErrors[worktreePath] != nil {
		return false, m.DirtyErrors[worktreePath]
	}
	return m.DirtyPaths[worktreePath], nil
}

func (m *MockGitRunner) IsPRMerged(repoRoot, branch string) (bool, error) {
	return m.MergedBranches[branch], nil
}

func (m *MockGitRunner) IsOrphan(repoRoot, branch string) (bool, error) {
	return m.OrphanBranches[branch], nil
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
		// Git invariant: git worktree list --porcelain always lists the main worktree as the first entry (index 0).
		isMain := (i == 0)
		isCurrent := (absPath == absRepoRoot)

		dirty, err := runner.IsDirty(raw.Path)
		if err != nil {
			// Fail-closed: treat error in status check as dirty so it is never reaped (F2)
			dirty = true
		}
		merged, _ := runner.IsPRMerged(repoRoot, raw.Branch)
		orphan, _ := runner.IsOrphan(repoRoot, raw.Branch)
		meta, err := LoadMetadata(raw.Path)
		if err != nil {
			// Fail-closed: unparseable or corrupted metadata -> nil -> refused reap (F2)
			meta = nil
		}

		info := Info{
			Path:       raw.Path,
			Head:       raw.HEAD,
			Branch:     raw.Branch,
			IsBare:     raw.Bare,
			IsDetached: raw.Detached,
			IsMain:     isMain,
			IsCurrent:  isCurrent,
			IsOrphan:   orphan,
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

// ListAll scans all sibling repositories in parentDir and collects their worktrees.
func ListAll(parentDir string) ([]Info, error) {
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", parentDir, err)
	}

	var all []Info
	seen := make(map[string]bool)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidate := filepath.Join(parentDir, entry.Name())
		gitDir := filepath.Join(candidate, ".git")
		fi, err := os.Stat(gitDir)
		if err != nil {
			continue
		}

		// Only inspect actual repository roots (directory .git), not worktrees (.git is a file)
		if !fi.IsDir() {
			continue
		}

		infos, err := List(candidate)
		if err != nil {
			continue
		}

		for _, info := range infos {
			if !seen[info.Path] {
				seen[info.Path] = true
				all = append(all, info)
			}
		}
	}

	return all, nil
}
