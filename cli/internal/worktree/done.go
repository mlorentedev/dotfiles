package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/filelock"
)

type DoneOptions struct {
	RepoRoot     string
	WorktreePath string
	LockPath     string
	// Cwd is the caller's working directory. done refuses a worktree that
	// contains it; empty skips that one check (the ancestor walk still runs).
	Cwd   string
	Force bool
	// Sessions are the harness session stores. done refuses a worktree with
	// a session there under SessionWindow old unless IgnoreRecentSession;
	// Force does not override it.
	Sessions            SessionStores
	IgnoreRecentSession bool
	// LandedHeads answers which merged pull requests had a branch as their
	// head, on the forge the origin names. nil asks GitHub through gh.
	LandedHeads LandedHeads
}

// DoneResult reports what happened to the worktree's branch.
type DoneResult struct {
	DeletedBranch string // the local branch removed after a verified landing
	DeletedSHA    string // its tip, to restore it with git branch <name> <sha>
	KeptBranch    string // the branch left in place, when there is one
	KeptReason    string // why it was kept
}

// Done tears down a completed worktree safely, then deletes its local branch
// when the branch is verified to have landed: its tip is an ancestor of the
// base, or the head of a merged pull request (the squash case). Anything else
// keeps the branch, which after --force may be the only copy of the work.
func Done(opts DoneOptions) (DoneResult, error) {
	absRepo, absWT, err := validateDoneOptions(opts)
	if err != nil {
		return DoneResult{}, err
	}
	if meta, merr := LoadMetadata(absWT); merr == nil && meta != nil && meta.Lane != "" {
		return Park(opts)
	}
	if err := checkCallerOutside(absWT, opts.Cwd); err != nil {
		return DoneResult{}, err
	}
	lockPath := opts.LockPath
	if lockPath == "" {
		lockPath = DefaultLockPath()
	}
	unlock, err := TryLockFile(lockPath)
	if err != nil {
		return DoneResult{}, fmt.Errorf("acquiring worktree lock: %w", err)
	}
	defer unlock()

	if err := checkUncommittedChanges(absWT, opts.Force); err != nil {
		return DoneResult{}, err
	}
	if err := checkUnpushedCommits(absRepo, absWT, opts.Force, opts.landedHeads()); err != nil {
		return DoneResult{}, err
	}
	// Last before removal, so a session that wrote during the checks above
	// still counts.
	if !opts.IgnoreRecentSession {
		if err := checkNoRecentSession(opts.Sessions, absWT, time.Now()); err != nil {
			return DoneResult{}, err
		}
	}
	// Read before removal: the answer needs the worktree's HEAD.
	branch, tip, landed, why := branchLanding(absRepo, absWT, opts.landedHeads())
	if err := removeWorktree(absRepo, absWT, opts.Force); err != nil {
		return DoneResult{}, err
	}
	switch {
	case branch == "":
		return DoneResult{}, nil
	case !landed:
		return DoneResult{KeptBranch: branch, KeptReason: why}, nil
	}
	if out, err := exec.Command("git", "-C", absRepo, "branch", "-D", branch).CombinedOutput(); err != nil {
		return DoneResult{KeptBranch: branch, KeptReason: "git branch -D failed: " + strings.TrimSpace(string(out))}, nil
	}
	return DoneResult{DeletedBranch: branch, DeletedSHA: tip}, nil
}

// Park resets a persistent named lane worktree in-place (HARNESS-186) without
// deleting the directory from disk: verifies clean status and branchLanding tip
// SHA containment (M3), detaches HEAD at the remote default branch, deletes the
// merged local branch while recording its 40-char SHA, removes any residual
// .dotf-lane-prompt.md (M4), and updates .dotf-worktree.json to State="PARKED".
func Park(opts DoneOptions) (DoneResult, error) {
	absRepo, absWT, err := validateDoneOptions(opts)
	if err != nil {
		return DoneResult{}, err
	}
	lockPath := opts.LockPath
	if lockPath == "" {
		lockPath = DefaultLockPath()
	}
	unlock, err := filelock.Lock(lockPath, time.Minute)
	if err != nil {
		return DoneResult{}, fmt.Errorf("acquiring worktree lock: %w", err)
	}
	defer unlock()

	// Best-effort fetch so origin/main reflects any remote PR merge before tip check (m-CODE-3).
	_ = exec.Command("git", "-C", absRepo, "fetch", "--quiet", "origin").Run()

	if err := checkUncommittedChanges(absWT, opts.Force); err != nil {
		return DoneResult{}, err
	}
	if err := checkUnpushedCommits(absRepo, absWT, opts.Force, opts.landedHeads()); err != nil {
		return DoneResult{}, err
	}
	if !opts.IgnoreRecentSession {
		if err := checkNoRecentSession(opts.Sessions, absWT, time.Now()); err != nil {
			return DoneResult{}, err
		}
	}

	branch, tip, landed, why := branchLanding(absRepo, absWT, opts.landedHeads())
	if branch != "" && !landed && !opts.Force {
		return DoneResult{}, fmt.Errorf("worktree %s branch %s is not landed (%s); push or merge before parking, or pass --force", absWT, branch, why)
	}

	meta, _ := LoadMetadata(absWT)
	laneName := filepath.Base(absWT)
	createdAt := time.Now()
	issueNum := 0
	if meta != nil {
		if meta.Lane != "" {
			laneName = meta.Lane
		}
		if !meta.CreatedAt.IsZero() {
			createdAt = meta.CreatedAt
		}
		issueNum = meta.Issue
	}

	// Write State="PARKED" before detaching/deleting branch so an interrupted park
	// never leaves State="ACTIVE" on a detached HEAD (M-CODE-2).
	parkedMeta := Metadata{
		Lane:      laneName,
		State:     string(StateParked),
		CreatedAt: createdAt,
		ReapOK:    false,
	}
	if err := SaveMetadata(absWT, parkedMeta); err != nil {
		return DoneResult{}, fmt.Errorf("saving parked metadata: %w", err)
	}

	baseRef := resolveBaseRef(absRepo)
	if baseRef == "" {
		baseRef = "HEAD"
	}
	detachArgs := []string{"-C", absWT, "checkout", "--detach", baseRef}
	if opts.Force {
		detachArgs = []string{"-C", absWT, "checkout", "--force", "--detach", baseRef}
	}
	if out, err := exec.Command("git", detachArgs...).CombinedOutput(); err != nil {
		if meta != nil {
			_ = SaveMetadata(absWT, *meta)
		}
		return DoneResult{}, fmt.Errorf("git checkout --detach %s failed (%w): %s", baseRef, err, strings.TrimSpace(string(out)))
	}

	_ = os.Remove(filepath.Join(absWT, ".dotf-lane-prompt.md"))
	releaseIssueClaimRefs(absRepo, issueNum)

	switch {
	case branch == "":
		return DoneResult{}, nil
	case !landed:
		return DoneResult{KeptBranch: branch, KeptReason: why}, nil
	}
	_, _ = fmt.Fprintf(os.Stderr, "Deleting branch %s (was %s)\n", branch, tip)
	if out, err := exec.Command("git", "-C", absRepo, "branch", "-D", branch).CombinedOutput(); err != nil {
		return DoneResult{KeptBranch: branch, KeptReason: "git branch -D failed: " + strings.TrimSpace(string(out))}, nil
	}
	return DoneResult{DeletedBranch: branch, DeletedSHA: tip}, nil
}

// releaseIssueClaimRefs cleans up local and remote claim refs for issueNum (M-CODE-1).
func releaseIssueClaimRefs(absRepo string, issueNum int) {
	if issueNum <= 0 {
		return
	}
	claimRef := fmt.Sprintf("refs/dotf/claims/%d", issueNum)
	seenRef := fmt.Sprintf("refs/dotf/seen/%d", issueNum)
	_ = exec.Command("git", "-C", absRepo, "update-ref", "-d", claimRef).Run()
	_ = exec.Command("git", "-C", absRepo, "update-ref", "-d", seenRef).Run()
	_ = exec.Command("git", "-C", absRepo, "push", "--quiet", "--no-verify", "origin", ":"+claimRef).Run()
}

// branchLanding reads the worktree's branch and tip, and whether that tip has
// landed. A detached HEAD has no branch to delete; neither has main or master.
func branchLanding(absRepo, absWT string, heads LandedHeads) (branch, tip string, landed bool, why string) {
	out, err := exec.Command("git", "-C", absWT, "symbolic-ref", "--quiet", "--short", "HEAD").Output()
	if err != nil {
		return "", "", false, ""
	}
	branch = strings.TrimSpace(string(out))
	if branch == "main" || branch == "master" {
		return "", "", false, ""
	}
	if out, err = exec.Command("git", "-C", absWT, "rev-parse", "HEAD").Output(); err != nil {
		return branch, "", false, "its tip could not be read"
	}
	tip = strings.TrimSpace(string(out))
	if base := resolveBaseRef(absRepo); base != "" && exec.Command("git", "-C", absRepo, "merge-base", "--is-ancestor", tip, base).Run() == nil {
		return branch, tip, true, ""
	}
	if ok, err := landedInMergedPR(absWT, branch, tip, heads); ok {
		return branch, tip, true, ""
	} else if err != nil {
		return branch, tip, false, fmt.Sprintf("could not check for a merged pull request: %v", err)
	}
	return branch, tip, false, "not landed: neither an ancestor of the base nor the head of a merged pull request"
}

// landedHeads is the caller's forge reader, or GitHub through gh without one.
func (o DoneOptions) landedHeads() LandedHeads {
	if o.LandedHeads != nil {
		return o.LandedHeads
	}
	return mergedPRHeads
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

// callerInside is the ancestor walk of done_proc_{linux,darwin,other}.go,
// behind a variable so a test can drive the refusal on every platform.
var callerInside = isCallerInside

// ancestor is a process the caller runs under, named in a refusal so the
// operator can tell which one sits in the worktree.
type ancestor struct {
	PID  int
	Comm string
	Cwd  string
}

// checkCallerOutside refuses a worktree that holds the caller's working
// directory, or that of a process the caller runs under (#1653). It takes no
// force flag on purpose: --force is about uncommitted changes and gets passed
// routinely, while removing the directory a session lives in strands it, and a
// Claude Code session launched from the worktree cannot be resumed once the
// directory is gone. The way out is to run done from somewhere else.
func checkCallerOutside(absWT, cwd string) error {
	target := resolvedPath(absWT)
	if cwd != "" && pathWithin(resolvedPath(cwd), target) {
		return fmt.Errorf("the current directory %s is inside the worktree %s; run done from outside it, naming the worktree by path or slug", cwd, absWT)
	}
	if a, inside := callerInside(target); inside {
		return fmt.Errorf("process %d (%s), which this command runs under, works in %s inside the worktree %s; "+
			"removing it would strand that process, and a Claude Code session launched there could not be resumed. "+
			"Remove the worktree from a session that was not started inside it", a.PID, a.Comm, a.Cwd, absWT)
	}
	return nil
}

// resolvedPath is path made absolute and symlink-resolved, so it compares
// equal to what the kernel reports for a process cwd. A path that cannot be
// resolved is compared as given.
func resolvedPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	if r, err := filepath.EvalSymlinks(abs); err == nil {
		return r
	}
	return abs
}

// pathWithin reports whether path is root or below it. The separator matters:
// repo-wt-feat-2 shares a prefix with repo-wt-feat and is not inside it.
func pathWithin(path, root string) bool {
	return path == root || strings.HasPrefix(path, root+string(filepath.Separator))
}

func checkUncommittedChanges(absWT string, force bool) error {
	if force {
		return nil
	}
	runner := &RealGitRunner{}
	dirt, err := runner.Dirt(absWT)
	if err != nil {
		return fmt.Errorf("checking worktree status: %w; pass --force to override", err)
	}
	if dirt.Tracked {
		return fmt.Errorf("worktree has uncommitted changes (%s); commit or stash them, or pass --force", absWT)
	}
	if len(dirt.Ignored) > 0 {
		return fmt.Errorf("worktree holds ignored files that are not known build output (%s): %s; move or delete them, or pass --force", absWT, summarizePaths(dirt.Ignored))
	}
	return nil
}

func checkUnpushedCommits(absRepo, absWT string, force bool, heads LandedHeads) error {
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
	if count == 0 {
		return nil
	}
	// Ahead of the base is also what a squash-merged branch looks like once
	// GitHub deleted its remote: ask whether the work landed (#1608).
	landed, landErr := landedOnMergedPR(absWT, heads)
	if landed {
		return nil
	}
	reason := ""
	if landErr != nil {
		reason = fmt.Sprintf(" (could not check for a merged pull request: %v)", landErr)
	}
	return fmt.Errorf("branch has no upstream configured and has %d unpushed commit(s) ahead of %s%s; push before done, or pass --force", count, baseRef, reason)
}

// landedOnMergedPR asks landedInMergedPR about the worktree's own branch and
// tip. A detached HEAD names no pull request and answers no.
func landedOnMergedPR(absWT string, heads LandedHeads) (bool, error) {
	branch, err := exec.Command("git", "-C", absWT, "symbolic-ref", "--quiet", "--short", "HEAD").Output()
	if err != nil {
		return false, nil //nolint:nilerr // detached: no branch, no pull request
	}
	tip, err := exec.Command("git", "-C", absWT, "rev-parse", "HEAD").Output()
	if err != nil {
		return false, err
	}
	return landedInMergedPR(absWT, strings.TrimSpace(string(branch)), strings.TrimSpace(string(tip)), heads)
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

// resolveBaseRef is what a branch's landing is checked against: the remote's
// default branch, else a local main or master.
func resolveBaseRef(repoDir string) string {
	return firstRef(repoDir, append(slices.Clone(remoteBaseRefs), "main", "master"))
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
