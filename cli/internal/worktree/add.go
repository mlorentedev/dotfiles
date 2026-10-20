package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// AddOptions configures the creation of a new worktree.
type AddOptions struct {
	RepoRoot   string
	Slug       string
	CustomPath string
	// WorktreeRoot places the checkout at <root>/<repo>/<slug of branch>
	// (ResolveWorktreePath). Empty is refused unless CustomPath is set: there
	// is no fallback to the legacy <repo>-wt-<slug> sibling.
	WorktreeRoot string
	Branch       string
	// BaseRef is what the branch starts from. Empty means the remote's default
	// branch (AddRunner.DefaultBase), never the caller's HEAD.
	BaseRef string
	Issue   int
	TTL     time.Duration
	Creator string
}

// AddRunner executes the low-level git worktree creation.
type AddRunner interface {
	WorktreeAdd(repo, target, branch, base string) error
	// DefaultBase fetches origin and returns its default branch ref.
	DefaultBase(repo string) (string, error)
	// ExistingBranch reports whether branch exists, locally (remote "") or
	// only on origin (remote "origin").
	ExistingBranch(repo, branch string) (remote string, exists bool)
	// WorktreeAttach checks out an existing branch at target. With remote set,
	// it creates the local branch of the same name tracking remote's.
	WorktreeAttach(repo, target, branch, remote string) error
}

type RealAddRunner struct{}

// remoteBaseRefs is the order the remote's default branch is looked for in:
// origin/HEAD where the clone recorded it, else main, else master (the vault).
var remoteBaseRefs = []string{"origin/HEAD", "origin/main", "origin/master"}

func (r *RealAddRunner) DefaultBase(repo string) (string, error) {
	// A failed fetch (offline) still leaves the last-fetched refs, which are a
	// better base than none; say so instead of refusing.
	if out, err := exec.Command("git", "-C", repo, "fetch", "--quiet", "origin").CombinedOutput(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: git fetch origin failed, using the last-fetched refs: %s\n", strings.TrimSpace(string(out)))
	}
	if ref := firstRef(repo, remoteBaseRefs); ref != "" {
		return ref, nil
	}
	return "", fmt.Errorf("no remote default branch in %s (tried %s): pass --base", repo, strings.Join(remoteBaseRefs, ", "))
}

// firstRef returns the first of refs that resolves in repo, or "".
func firstRef(repo string, refs []string) string {
	for _, ref := range refs {
		if exec.Command("git", "-C", repo, "rev-parse", "--verify", "--quiet", ref).Run() == nil {
			return ref
		}
	}
	return ""
}

func (r *RealAddRunner) WorktreeAdd(repo, target, branch, base string) error {
	args := []string{"-C", repo, "worktree", "add", target, "-b", branch}
	if base != "" {
		args = append(args, base)
	}

	cmd := exec.Command("git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add failed (%w): %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (r *RealAddRunner) ExistingBranch(repo, branch string) (string, bool) {
	if firstRef(repo, []string{"refs/heads/" + branch}) != "" {
		return "", true
	}
	// Fetch only this branch. origin saying it has no such branch is the
	// answer, even over a stale remote ref; an unreachable origin leaves the
	// last-fetched ref as the best one.
	// LC_ALL=C: the message is matched, and git translates it.
	fetch := exec.Command("git", "-C", repo, "fetch", "--quiet", "origin", "refs/heads/"+branch+":refs/remotes/origin/"+branch)
	fetch.Env = append(os.Environ(), "LC_ALL=C")
	out, err := fetch.CombinedOutput()
	if err != nil && strings.Contains(string(out), "couldn't find remote ref") {
		return "", false
	}
	if firstRef(repo, []string{"refs/remotes/origin/" + branch}) != "" {
		return "origin", true
	}
	return "", false
}

func (r *RealAddRunner) WorktreeAttach(repo, target, branch, remote string) error {
	args := []string{"-C", repo, "worktree", "add", target, branch}
	if remote != "" {
		args = []string{"-C", repo, "worktree", "add", "--track", "-b", branch, target, remote + "/" + branch}
	}
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add failed (%w): %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ResolveSiblingPath computes the canonical external sibling path (<repo>-wt-<slug>).
func ResolveSiblingPath(repoRoot, slug string) string {
	cleanRoot := filepath.Clean(repoRoot)
	base := filepath.Base(cleanRoot)
	dir := filepath.Dir(cleanRoot)
	return filepath.Join(dir, fmt.Sprintf("%s-wt-%s", base, slug))
}

// ValidateIsolation ensures targetPath is strictly outside repoRoot (preventing 160000 gitlinks).
func ValidateIsolation(repoRoot, targetPath string) error {
	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return err
	}
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}

	if absTarget == absRepo {
		return fmt.Errorf("worktree path cannot be the repository root itself: %s", targetPath)
	}

	rel, err := filepath.Rel(absRepo, absTarget)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(rel, "..") && rel != "." {
		return fmt.Errorf("isolation violation: target path %q is nested inside repository %q", targetPath, repoRoot)
	}

	return nil
}

func validateAddOptions(opts AddOptions) error {
	if opts.RepoRoot == "" {
		return fmt.Errorf("repository root cannot be empty")
	}
	if opts.Slug == "" && opts.CustomPath == "" {
		return fmt.Errorf("either slug or custom path must be specified")
	}
	// No silent fallback to the legacy sibling: a worktree's layout must not
	// depend on whether this process could locate env-contract.json.
	if opts.CustomPath == "" && opts.WorktreeRoot == "" {
		return fmt.Errorf("WORKTREE_ROOT could not be resolved: run `dotf doctor`, or pass --path")
	}
	return nil
}

func resolveTargetPath(opts AddOptions, branch string) string {
	if opts.CustomPath != "" {
		return opts.CustomPath
	}
	return ResolveWorktreePath(opts.WorktreeRoot, filepath.Base(filepath.Clean(opts.RepoRoot)), branch)
}

func resolveCreator(explicit string) string {
	if explicit != "" {
		return explicit
	}
	for _, envVar := range []string{"AGENT_NAME", "CLAUDE_CODE", "USER"} {
		if val := os.Getenv(envVar); val != "" {
			if envVar == "CLAUDE_CODE" {
				return "claude-code"
			}
			return val
		}
	}
	return "unknown"
}

func resolveAddDefaults(opts AddOptions) (string, time.Duration, string) {
	branchName := opts.Branch
	if branchName == "" {
		branchName = DefaultBranch(opts.Slug)
	}

	ttl := opts.TTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	creator := resolveCreator(opts.Creator)
	return branchName, ttl, creator
}

// AddWithRunner handles worktree creation with a given AddRunner.
func AddWithRunner(opts AddOptions, runner AddRunner, now time.Time) (*Info, error) {
	if err := validateAddOptions(opts); err != nil {
		return nil, err
	}

	branchName, ttl, creator := resolveAddDefaults(opts)
	// A branch that already exists is attached under its own name: the shape
	// rule gates the branches add creates, and a bot's release or dependabot
	// branch is not one of them (#2822). Only the directory is slugified.
	remote, exists := runner.ExistingBranch(opts.RepoRoot, branchName)
	if exists && opts.BaseRef != "" {
		return nil, fmt.Errorf("branch %q already exists, so add attaches to it and --base %s does not apply: drop --base, or name a new branch", branchName, opts.BaseRef)
	}
	if !exists {
		if err := ValidateBranch(branchName); err != nil {
			return nil, err
		}
	}
	targetPath := resolveTargetPath(opts, branchName)
	if err := ValidateIsolation(opts.RepoRoot, targetPath); err != nil {
		return nil, err
	}

	if _, err := os.Stat(targetPath); err == nil {
		return nil, fmt.Errorf("target path already exists: %s", targetPath)
	}

	// Gate A: warn if parent repository has active auto-commit hooks/plugins
	if warn := CheckAutoCommitHooks(opts.RepoRoot); warn != "" {
		_, _ = fmt.Fprintln(os.Stderr, warn)
	}

	if exists {
		if err := runner.WorktreeAttach(opts.RepoRoot, targetPath, branchName, remote); err != nil {
			return nil, err
		}
	} else if err := createBranch(opts, runner, targetPath, branchName); err != nil {
		return nil, err
	}

	meta := Metadata{
		Creator:        creator,
		Issue:          opts.Issue,
		CreatedAt:      now,
		LeaseExpiresAt: now.Add(ttl),
		ReapOK:         true,
	}
	if err := SaveMetadata(targetPath, meta); err != nil {
		return nil, fmt.Errorf("saving worktree metadata: %w", err)
	}

	ensureExclude(opts.RepoRoot, MetadataFileName)

	return &Info{
		Path:     targetPath,
		Branch:   branchName,
		Metadata: &meta,
		State:    StateActive,
	}, nil
}

// createBranch starts branch from opts.BaseRef, or from origin's default branch.
func createBranch(opts AddOptions, runner AddRunner, target, branch string) error {
	base := opts.BaseRef
	if base == "" {
		var err error
		if base, err = runner.DefaultBase(opts.RepoRoot); err != nil {
			return err
		}
	}
	return runner.WorktreeAdd(opts.RepoRoot, target, branch, base)
}

// Add creates a new worktree using RealAddRunner.
func Add(opts AddOptions) (*Info, error) {
	return AddWithRunner(opts, &RealAddRunner{}, time.Now())
}

func containsLine(content, target string) bool {
	lines := strings.Split(content, "\n")
	for _, l := range lines {
		if strings.TrimSpace(l) == target {
			return true
		}
	}
	return false
}

func ensureExclude(repoRoot, pattern string) {
	excludePath := filepath.Join(repoRoot, ".git", "info", "exclude")
	content, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	if !containsLine(string(content), pattern) {
		newContent := string(content)
		if len(newContent) > 0 && !strings.HasSuffix(newContent, "\n") {
			newContent += "\n"
		}
		newContent += pattern + "\n"
		_ = os.MkdirAll(filepath.Dir(excludePath), 0o755)
		_ = os.WriteFile(excludePath, []byte(newContent), 0o644)
	}
}

// CheckAutoCommitHooks detects if git hooks or plugin configs specify auto-commit behavior (Gate A).
func CheckAutoCommitHooks(repoRoot string) string {
	hooksDir := filepath.Join(repoRoot, ".git", "hooks")
	if out, err := exec.Command("git", "-C", repoRoot, "config", "--local", "--get", "core.hooksPath").Output(); err == nil {
		p := strings.TrimSpace(string(out))
		if p != "" {
			hooksDir = p
		}
	}
	postCommit := filepath.Join(hooksDir, "post-commit")
	if data, err := os.ReadFile(postCommit); err == nil {
		if strings.Contains(string(data), "auto-commit") || strings.Contains(string(data), "git commit") {
			return fmt.Sprintf("warning (Gate A): active auto-commit hook detected in %s", postCommit)
		}
	}
	return ""
}
