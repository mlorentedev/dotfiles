package lane

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/filelock"
	"github.com/mlorentedev/dotfiles/cli/internal/mem"
	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

const (
	LaneFeatPrimary   = "feat-lane-primary"
	LaneFeatSecondary = "feat-lane-secondary"
	LaneFastTrack     = "fast-track-lane"
)

// LanesMarkerFile is the committed declaration that a repository runs the
// persistent named lane system, and the only thing the pre-commit lane guard
// keys off. It carries no paths and no host names, so the same marker is valid
// on every OS, agent host and seat.
const LanesMarkerFile = ".dotf/lanes.json"

// CanonicalLanes lists the 3 persistent worktree lanes per repository (HARNESS-186).
var CanonicalLanes = []string{
	LaneFeatPrimary,
	LaneFeatSecondary,
	LaneFastTrack,
}

// ThirdPartyWorktreeRoots are directory names, under the projects boundary, that
// belong to an external agent session host rather than to a lane. They are
// declared into the marker instead of being hardcoded in the guard, so adding a
// host is a data change and no host is privileged in the code.
var ThirdPartyWorktreeRoots = []string{"copilot-worktrees"}

// lanesMarker is the on-disk shape of .dotf/lanes.json.
type lanesMarker struct {
	Version         int      `json:"version"`
	CanonicalLanes  []string `json:"canonical_lanes"`
	ThirdPartyRoots []string `json:"third_party_roots"`
}

var branchIssueRe = regexp.MustCompile(`^[a-z]{1,12}/([0-9]{1,7})(?:-[a-z0-9._-]+)?$`)

// InitLanes idempotently provisions the 3 persistent named lanes for repoRoot
// under <worktreeRoot>/<repo>/<lane> detached at origin's default branch with
// State="PARKED" and ReapOK=false. Returns the number of lanes created or updated.
func InitLanes(repoRoot, worktreeRoot string) (int, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return 0, fmt.Errorf("lane init: repository root cannot be empty")
	}
	if strings.TrimSpace(worktreeRoot) == "" {
		return 0, fmt.Errorf("lane init: WORKTREE_ROOT cannot be empty")
	}
	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return 0, err
	}
	absWTRoot, err := filepath.Abs(worktreeRoot)
	if err != nil {
		return 0, err
	}

	unlock, err := filelock.Lock(worktree.DefaultLockPath(), time.Minute)
	if err != nil {
		return 0, fmt.Errorf("lane init: acquiring worktree lock: %w", err)
	}
	defer unlock()

	return initLanesLocked(absRepo, absWTRoot)
}

func initLanesLocked(absRepo, absWTRoot string) (int, error) {
	baseRef, err := (&worktree.RealAddRunner{}).DefaultBase(absRepo)
	if err != nil || baseRef == "" {
		baseRef = "HEAD"
	}

	repoName := filepath.Base(filepath.Clean(absRepo))
	repoWTRoot := filepath.Join(absWTRoot, repoName)
	if err := os.MkdirAll(repoWTRoot, 0o755); err != nil {
		return 0, fmt.Errorf("lane init: mkdir %s: %w", repoWTRoot, err)
	}

	// info/exclude lives in the COMMON git dir, not in <repo>/.git: in a linked
	// worktree that path is a pointer FILE, so the exclusion silently did
	// nothing there and the metadata showed up as untracked — the BUG-043 shape
	// on the Go side. Both errors are returned rather than discarded: a missing
	// exclusion makes every `git status` dirty, and the marker below is written
	// only after this succeeds, so a failed init leaves the lane guard INACTIVE
	// instead of half-configured.
	gitCommonDir, err := resolveGitCommonDir(absRepo)
	if err != nil {
		return 0, err
	}
	for _, name := range []string{worktree.MetadataFileName, PromptFile} {
		if err := excludeFile(gitCommonDir, name); err != nil {
			return 0, err
		}
	}

	if _, err := writeLanesMarker(absRepo); err != nil {
		return 0, err
	}

	changed := 0
	for _, name := range CanonicalLanes {
		created, err := provisionLane(absRepo, filepath.Join(repoWTRoot, name), name, baseRef)
		if err != nil {
			return changed, err
		}
		if created {
			changed++
		}
	}
	return changed, nil
}

// resolveGitCommonDir answers the repository's shared git directory — where
// info/exclude lives — for a main checkout, a linked worktree and a bare repo
// alike. --path-format=absolute keeps a relative answer from being joined to
// the wrong base (git answers `.git` for an ordinary checkout).
func resolveGitCommonDir(repo string) (string, error) {
	out, err := exec.Command("git", "-C", repo, "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		return "", fmt.Errorf("lane init: resolving the git common dir of %s: %w", repo, err)
	}
	common := strings.TrimSpace(string(out))
	if common == "" {
		return "", fmt.Errorf("lane init: git reported no common dir for %s", repo)
	}
	return common, nil
}

// provisionLane makes the lane at lanePath exist detached at baseRef and carry
// lease metadata, reporting whether it wrote anything.
func provisionLane(absRepo, lanePath, name, baseRef string) (bool, error) {
	now := time.Now()

	if _, statErr := os.Stat(lanePath); statErr != nil {
		out, err := exec.Command("git", "-C", absRepo, "worktree", "add", "--detach", lanePath, baseRef).CombinedOutput()
		if err != nil {
			return false, fmt.Errorf("lane init: git worktree add --detach %s failed (%w): %s", name, err, strings.TrimSpace(string(out)))
		}
		return true, saveLaneMetadata(lanePath, worktree.Metadata{
			Lane:      name,
			State:     string(worktree.StateParked),
			CreatedAt: now,
			ReapOK:    false,
		})
	}

	meta, merr := worktree.LoadMetadata(lanePath)
	if merr == nil && meta != nil && meta.Lane == name && meta.State != "" && !meta.ReapOK {
		return false, nil
	}

	state := string(worktree.StateParked)
	issueNum := 0
	if meta != nil && meta.Issue > 0 {
		issueNum = meta.Issue
	}
	if out, berr := exec.Command("git", "-C", lanePath, "symbolic-ref", "--quiet", "--short", "HEAD").Output(); berr == nil && strings.TrimSpace(string(out)) != "" {
		state = string(worktree.StateActive)
	}
	return true, saveLaneMetadata(lanePath, worktree.Metadata{
		Lane:           name,
		State:          state,
		Issue:          issueNum,
		CreatedAt:      now,
		LeaseExpiresAt: now.Add(leaseTTL),
		ReapOK:         false,
	})
}

func saveLaneMetadata(lanePath string, meta worktree.Metadata) error {
	if err := worktree.SaveMetadata(lanePath, meta); err != nil {
		return fmt.Errorf("lane init: saving metadata for %s: %w", meta.Lane, err)
	}
	return nil
}

// writeLanesMarker idempotently writes .dotf/lanes.json into absRepo and reports
// whether the file changed. The marker is the named lane system's own opt-in: it
// is written by `dotf lane init` and read by the pre-commit guard, so no other
// feature's config file can switch the guard on.
//
// canonical_lanes is policy owned by this command and is always rewritten.
// third_party_roots is host-owned data, so roots already declared on disk are
// preserved: adding a session host is a data change, never a code change.
func writeLanesMarker(absRepo string) (bool, error) {
	path := filepath.Join(absRepo, filepath.FromSlash(LanesMarkerFile))

	roots := append([]string(nil), ThirdPartyWorktreeRoots...)
	if existing, rerr := os.ReadFile(path); rerr == nil {
		var prev lanesMarker
		if json.Unmarshal(existing, &prev) == nil {
			roots = unionRoots(roots, prev.ThirdPartyRoots)
		}
	}

	want, err := json.MarshalIndent(lanesMarker{
		Version:         1,
		CanonicalLanes:  CanonicalLanes,
		ThirdPartyRoots: roots,
	}, "", "  ")
	if err != nil {
		return false, fmt.Errorf("lane init: encoding %s: %w", LanesMarkerFile, err)
	}
	want = append(want, '\n')

	if existing, rerr := os.ReadFile(path); rerr == nil && bytes.Equal(existing, want) {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("lane init: mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, want, 0o644); err != nil {
		return false, fmt.Errorf("lane init: writing %s: %w", LanesMarkerFile, err)
	}
	return true, nil
}

// unionRoots appends the entries of extra that base does not already carry,
// preserving order and dropping blanks and duplicates.
func unionRoots(base, extra []string) []string {
	seen := make(map[string]bool, len(base)+len(extra))
	out := make([]string, 0, len(base)+len(extra))
	for _, group := range [][]string{base, extra} {
		for _, root := range group {
			root = strings.TrimSpace(root)
			if root == "" || seen[root] {
				continue
			}
			seen[root] = true
			out = append(out, root)
		}
	}
	return out
}

// RouteBranchToLane deterministically selects a PARKED canonical lane for branch (M5).
func RouteBranchToLane(repoRoot, worktreeRoot, branch string) (string, string, error) {
	repoName := filepath.Base(filepath.Clean(repoRoot))
	repoWTRoot := filepath.Join(worktreeRoot, repoName)

	// If a canonical lane already holds this exact branch, return it idempotently.
	for _, name := range CanonicalLanes {
		lanePath := filepath.Join(repoWTRoot, name)
		if out, err := exec.Command("git", "-C", lanePath, "symbolic-ref", "--quiet", "--short", "HEAD").Output(); err == nil {
			if strings.TrimSpace(string(out)) == branch {
				return name, lanePath, nil
			}
		}
	}

	prefix, _, _ := strings.Cut(branch, "/")
	var candidates []string
	switch prefix {
	case "feat", "refactor", "perf", "build":
		candidates = []string{LaneFeatPrimary, LaneFeatSecondary}
	default:
		// fix, hotfix, chore, docs, ci, test, revert, etc.
		candidates = []string{LaneFastTrack, LaneFeatSecondary}
	}

	for _, name := range candidates {
		lanePath := filepath.Join(repoWTRoot, name)
		meta, err := worktree.LoadMetadata(lanePath)
		if err == nil && meta != nil && strings.EqualFold(meta.State, string(worktree.StateParked)) {
			return name, lanePath, nil
		}
	}

	return "", "", fmt.Errorf("lane switch: WIP cap reached for %q: candidate lanes (%s) are all ACTIVE; park a finished lane with `dotf lane park` first", branch, strings.Join(candidates, ", "))
}

// SwitchLane claims a PARKED persistent lane (or explicitLane), checks out branch
// inside it, and transitions its .dotf-worktree.json to State="ACTIVE".
func SwitchLane(repoRoot, worktreeRoot, branch, explicitLane string) (string, string, error) {
	if strings.TrimSpace(branch) == "" {
		return "", "", fmt.Errorf("lane switch: branch cannot be empty")
	}
	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", "", err
	}
	absWTRoot, err := filepath.Abs(worktreeRoot)
	if err != nil {
		return "", "", err
	}

	unlock, err := filelock.Lock(worktree.DefaultLockPath(), time.Minute)
	if err != nil {
		return "", "", fmt.Errorf("lane switch: acquiring worktree lock: %w", err)
	}
	defer unlock()

	return switchLaneLocked(absRepo, absWTRoot, branch, explicitLane)
}

func switchLaneLocked(absRepo, absWTRoot, branch, explicitLane string) (string, string, error) {
	var laneName, lanePath string
	if explicitLane != "" {
		repoName := filepath.Base(filepath.Clean(absRepo))
		laneName = explicitLane
		lanePath = filepath.Join(absWTRoot, repoName, explicitLane)
		meta, err := worktree.LoadMetadata(lanePath)
		if err != nil || meta == nil {
			return "", "", fmt.Errorf("lane switch: lane %q is not initialized at %s; run `dotf lane init` first", explicitLane, lanePath)
		}
		if !strings.EqualFold(meta.State, string(worktree.StateParked)) {
			curBranchOut, _ := exec.Command("git", "-C", lanePath, "symbolic-ref", "--quiet", "--short", "HEAD").Output()
			if strings.TrimSpace(string(curBranchOut)) != branch {
				return "", "", fmt.Errorf("lane switch: lane %q is already ACTIVE on %q; park it first", explicitLane, strings.TrimSpace(string(curBranchOut)))
			}
		}
	} else {
		var err error
		laneName, lanePath, err = RouteBranchToLane(absRepo, absWTRoot, branch)
		if err != nil {
			return "", "", err
		}
	}

	dirt, err := (&worktree.RealGitRunner{}).Dirt(lanePath)
	if err != nil {
		return "", "", fmt.Errorf("lane switch: checking %s status: %w", laneName, err)
	}
	if dirt.Any() {
		return "", "", fmt.Errorf("lane switch: lane %s has uncommitted or non-disposable ignored files", laneName)
	}

	if err := checkoutBranchInLane(absRepo, lanePath, branch); err != nil {
		return "", "", err
	}

	issueNum := parseIssueFromBranch(branch)
	now := time.Now()
	meta := worktree.Metadata{
		Lane:           laneName,
		State:          string(worktree.StateActive),
		Issue:          issueNum,
		CreatedAt:      now,
		LeaseExpiresAt: now.Add(leaseTTL),
		ReapOK:         false,
	}
	if err := worktree.SaveMetadata(lanePath, meta); err != nil {
		return "", "", fmt.Errorf("lane switch: saving metadata: %w", err)
	}
	return laneName, lanePath, nil
}

func checkoutBranchInLane(absRepo, lanePath, branch string) error {
	if exec.Command("git", "-C", absRepo, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch).Run() == nil {
		if out, err := exec.Command("git", "-C", lanePath, "checkout", branch).CombinedOutput(); err != nil {
			return fmt.Errorf("lane switch: git checkout %s failed (%w): %s", branch, err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	if exec.Command("git", "-C", absRepo, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+branch).Run() == nil {
		if out, err := exec.Command("git", "-C", lanePath, "checkout", "--track", "-b", branch, "origin/"+branch).CombinedOutput(); err != nil {
			return fmt.Errorf("lane switch: git checkout --track %s failed (%w): %s", branch, err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	baseRef := resolveLaneBaseRef(absRepo)
	if out, err := exec.Command("git", "-C", lanePath, "checkout", "--no-track", "-B", branch, baseRef).CombinedOutput(); err != nil {
		return fmt.Errorf("lane switch: git checkout -B %s %s failed (%w): %s", branch, baseRef, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func resolveLaneBaseRef(absRepo string) string {
	for _, ref := range []string{"origin/HEAD", "origin/main", "origin/master", "main", "master"} {
		if exec.Command("git", "-C", absRepo, "rev-parse", "--verify", "--quiet", ref).Run() == nil {
			return ref
		}
	}
	return "HEAD"
}

func parseIssueFromBranch(branch string) int {
	m := branchIssueRe.FindStringSubmatch(branch)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// HasNamedLanes reports whether repoRoot has persistent named lanes initialized under worktreeRoot.
func HasNamedLanes(repoRoot, worktreeRoot string) bool {
	if repoRoot == "" || worktreeRoot == "" {
		return false
	}
	repoName := filepath.Base(filepath.Clean(repoRoot))
	meta, err := worktree.LoadMetadata(filepath.Join(worktreeRoot, repoName, LaneFeatPrimary))
	return err == nil && meta != nil && meta.Lane == LaneFeatPrimary
}

// ParkLane parks the specified lane (or the lane at cwd) in-place, releases any
// issue claim ref, and archives/prunes its memory thread in vaultPath.
func ParkLane(repoRoot, lanePath, vaultPath string, force bool) (worktree.DoneResult, error) {
	meta, err := worktree.LoadMetadata(lanePath)
	if err != nil || meta == nil || meta.Lane == "" {
		return worktree.DoneResult{}, fmt.Errorf("lane park: %s is not a persistent named lane", lanePath)
	}
	branchOut, _ := exec.Command("git", "-C", lanePath, "symbolic-ref", "--quiet", "--short", "HEAD").Output()
	oldBranch := strings.TrimSpace(string(branchOut))

	res, err := worktree.Park(worktree.DoneOptions{
		RepoRoot:            repoRoot,
		WorktreePath:        lanePath,
		Force:               force,
		IgnoreRecentSession: true,
	})
	if err != nil {
		return res, err
	}
	if vaultPath != "" && oldBranch != "" {
		project := filepath.Base(filepath.Clean(repoRoot))
		_ = mem.ArchiveAndPruneThread(filepath.Join(vaultPath, "10_projects", project), oldBranch)
	}
	return res, nil
}
