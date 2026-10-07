package worktree

import (
	"encoding/json"
	"errors"
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
	tip, err := exec.Command("git", "-C", repoRoot, "rev-parse", "--verify", branch+"^{commit}").Output()
	if err != nil {
		return false
	}
	landed, _ := landedInMergedPR(repoRoot, branch, strings.TrimSpace(string(tip)))
	return landed
}

// mergedPRHeads returns the head commit of every merged pull request whose head
// branch is branch. A variable so tests answer without reaching GitHub.
var mergedPRHeads = ghMergedPRHeads

func ghMergedPRHeads(repoRoot, branch string) ([]string, error) {
	args := []string{"pr", "list", "--head", branch, "--state", "merged", "--json", "headRefOid", "--jq", ".[].headRefOid"}
	if out, err := exec.Command("git", "-C", repoRoot, "config", "--get", "remote.origin.url").Output(); err == nil {
		if slug := ParseGitHubSlug(string(out)); slug != "" {
			args = append(args, "--repo", slug)
		}
	}
	cmd := exec.Command("gh", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh pr list --head %s: %w", branch, err)
	}
	return strings.Fields(string(out)), nil
}

// landedInMergedPR reports whether tip is contained in the head of a merged
// pull request for branch (#1608). A squash merge puts a new commit on the
// base, so the branch is never an ancestor of it and ancestry cannot answer;
// containment in the head that merged can. Containment, not equality: a
// checkout behind its PR head (the PR took a rebase or a merge of main from
// elsewhere) has nothing unpushed, while a commit made after the merge is in
// no merged head and still reads as unpushed.
func landedInMergedPR(repoRoot, branch, tip string) (bool, error) {
	heads, err := mergedPRHeads(repoRoot, branch)
	if err != nil {
		return false, err
	}
	var unfetched []error
	for _, head := range heads {
		if exec.Command("git", "-C", repoRoot, "cat-file", "-e", head+"^{commit}").Run() != nil {
			// The head may exist only on the remote; GitHub serves a merged
			// PR's head by its SHA.
			if err := fetchCommit(repoRoot, head); err != nil {
				unfetched = append(unfetched, err)
				continue
			}
		}
		if exec.Command("git", "-C", repoRoot, "merge-base", "--is-ancestor", tip, head).Run() == nil {
			return true, nil
		}
	}
	// A head that could not be fetched is a question nobody answered, not a
	// "no": reading it as one sent #1608's operator to push a branch GitHub had
	// already deleted, with no word that the check itself had failed.
	return false, errors.Join(unfetched...)
}

// fetchCommit fetches one commit from origin by its SHA. It never prompts for
// credentials: `list` and `sweep` ask this too, and a listing must fail fast
// rather than wait on a terminal nobody is watching.
func fetchCommit(repoRoot, sha string) error {
	cmd := exec.Command("git", "-C", repoRoot, "fetch", "--quiet", "--no-tags", "origin", sha)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("fetch merged head %s from origin: %w: %s", shortSHA(sha), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
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

func isMainWorktree(path string, index int) bool {
	// In Git architecture, the main worktree has a .git directory, while linked worktrees have a .git file.
	fi, err := os.Stat(filepath.Join(path, ".git"))
	if err == nil {
		return fi.IsDir()
	}
	// Fallback to git worktree list --porcelain invariant (entry 0 is main)
	return index == 0
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
		isMain := isMainWorktree(raw.Path, i)
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
