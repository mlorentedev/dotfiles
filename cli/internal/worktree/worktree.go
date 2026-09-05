package worktree

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LifecycleState defines the operational phase of a worktree.
type LifecycleState string

const (
	StateActive   LifecycleState = "ACTIVE"
	StateReapable LifecycleState = "REAPABLE"
	StateDirty    LifecycleState = "DIRTY"
	StateUnmerged LifecycleState = "UNMERGED"
	StateOrphan   LifecycleState = "ORPHAN"
)

const MetadataFileName = ".dotf-worktree.json"

// Metadata is stored in <worktree-root>/.dotf-worktree.json.
type Metadata struct {
	Creator        string    `json:"creator,omitempty"`
	Issue          int       `json:"issue,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	LeaseExpiresAt time.Time `json:"lease_expires_at"`
	ReapOK         bool      `json:"reap_ok"`
}

// RawWorktree represents a parsed block from `git worktree list --porcelain`.
type RawWorktree struct {
	Path        string
	HEAD        string
	Branch      string
	Bare        bool
	Detached    bool
	Locked      bool
	LockReason  string
	Prunable    bool
	GitDir      string
	IsSubmodule bool
}

// Info captures the evaluated state of a worktree.
type Info struct {
	Path        string         `json:"path"`
	Head        string         `json:"head"`
	Branch      string         `json:"branch"`
	IsBare      bool           `json:"is_bare,omitempty"`
	IsDetached  bool           `json:"is_detached,omitempty"`
	IsMain      bool           `json:"is_main,omitempty"`
	Dirty       bool           `json:"dirty"`
	PRMerged    bool           `json:"pr_merged"`
	Metadata    *Metadata      `json:"metadata,omitempty"`
	State       LifecycleState `json:"state"`
	StateReason string         `json:"state_reason,omitempty"`
}

// IsSubmoduleGitDir returns true if the gitdir path indicates a git submodule.
func IsSubmoduleGitDir(gitdir string) bool {
	normalized := filepath.ToSlash(gitdir)
	return strings.Contains(normalized, "/.git/modules/") || strings.Contains(normalized, "/modules/")
}

// ParsePorcelain parses the standard output of `git worktree list --porcelain`.
func ParsePorcelain(output string) ([]RawWorktree, error) {
	var results []RawWorktree
	var current *RawWorktree

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			if current != nil {
				results = append(results, *current)
				current = nil
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				results = append(results, *current)
			}
			current = &RawWorktree{
				Path: strings.TrimPrefix(line, "worktree "),
			}
			continue
		}

		if current == nil {
			continue
		}

		switch {
		case strings.HasPrefix(line, "HEAD "):
			current.HEAD = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "bare":
			current.Bare = true
		case line == "detached":
			current.Detached = true
		case strings.HasPrefix(line, "locked"):
			current.Locked = true
			current.LockReason = strings.TrimSpace(strings.TrimPrefix(line, "locked"))
		case strings.HasPrefix(line, "prunable"):
			current.Prunable = true
		case strings.HasPrefix(line, "gitdir "):
			current.GitDir = strings.TrimPrefix(line, "gitdir ")
			current.IsSubmodule = IsSubmoduleGitDir(current.GitDir)
		}
	}

	if current != nil {
		results = append(results, *current)
	}

	return results, scanner.Err()
}

// LoadMetadata reads .dotf-worktree.json from the worktree root if present.
func LoadMetadata(worktreePath string) (*Metadata, error) {
	metaPath := filepath.Join(worktreePath, MetadataFileName)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading worktree metadata: %w", err)
	}

	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing worktree metadata: %w", err)
	}
	return &meta, nil
}

// Classify evaluates all fail-closed criteria to decide the lifecycle state.
func Classify(info Info, now time.Time) (LifecycleState, string) {
	if info.IsMain {
		return StateActive, "main repository"
	}

	if info.Dirty {
		return StateDirty, "uncommitted changes"
	}

	// Gate (a): explicit metadata exists in .dotf-worktree.json with reap_ok: true
	if info.Metadata == nil {
		return StateActive, "no dotf metadata (reap refused: requires explicit reap_ok=true)"
	}

	if !info.Metadata.ReapOK {
		return StateActive, "reap hold (reap_ok=false)"
	}
	if info.Metadata.LeaseExpiresAt.After(now) {
		return StateActive, fmt.Sprintf("lease active until %s", info.Metadata.LeaseExpiresAt.Format(time.RFC3339))
	}
	if now.Sub(info.Metadata.CreatedAt) < 15*time.Minute {
		return StateActive, "newly created (age < 15m)"
	}

	if !info.PRMerged {
		return StateUnmerged, "PR not merged"
	}

	return StateReapable, "eligible for cleanup"
}
