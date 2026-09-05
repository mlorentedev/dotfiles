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
	Branch     string
	BaseRef    string
	Issue      int
	TTL        time.Duration
	Creator    string
}

// AddRunner executes the low-level git worktree creation.
type AddRunner interface {
	WorktreeAdd(repo, target, branch, base string) error
}

type RealAddRunner struct{}

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

// AddWithRunner handles worktree creation with a given AddRunner.
func AddWithRunner(opts AddOptions, runner AddRunner, now time.Time) (*Info, error) {
	if opts.RepoRoot == "" {
		return nil, fmt.Errorf("repository root cannot be empty")
	}
	if opts.Slug == "" && opts.CustomPath == "" {
		return nil, fmt.Errorf("either slug or custom path must be specified")
	}

	targetPath := opts.CustomPath
	if targetPath == "" {
		targetPath = ResolveSiblingPath(opts.RepoRoot, opts.Slug)
	}

	if err := ValidateIsolation(opts.RepoRoot, targetPath); err != nil {
		return nil, err
	}

	if _, err := os.Stat(targetPath); err == nil {
		return nil, fmt.Errorf("target path already exists: %s", targetPath)
	}

	branchName := opts.Branch
	if branchName == "" {
		branchName = "feat/" + opts.Slug
	}

	ttl := opts.TTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	creator := opts.Creator
	if creator == "" {
		if ag := os.Getenv("AGENT_NAME"); ag != "" {
			creator = ag
		} else if cc := os.Getenv("CLAUDE_CODE"); cc != "" {
			creator = "claude-code"
		} else if u := os.Getenv("USER"); u != "" {
			creator = u
		} else {
			creator = "unknown"
		}
	}

	// 1. Create worktree
	if err := runner.WorktreeAdd(opts.RepoRoot, targetPath, branchName, opts.BaseRef); err != nil {
		return nil, err
	}

	// 2. Write metadata
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

	// 3. Exclude metadata in repo .git/info/exclude
	ensureExclude(opts.RepoRoot, MetadataFileName)

	return &Info{
		Path:     targetPath,
		Branch:   branchName,
		Metadata: &meta,
		State:    StateActive,
	}, nil
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
