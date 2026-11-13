package worktree

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LeaseFileName is the lease's name inside the worktree's git admin dir,
// <repo>/.git/worktrees/<id>/ (ADR-062). MetadataFileName keeps naming the
// in-tree copy: the mirror written for one release, the fallback for a lease
// written before the move, and the file the exclude, clean and audit paths
// still have to name.
const LeaseFileName = "dotf-worktree.json"

// ErrNotLinkedWorktree is returned for a path whose .git is not a pointer
// into a repository's worktrees/ directory: a main checkout, a plain
// directory, or a submodule checkout.
var ErrNotLinkedWorktree = errors.New("not a linked worktree")

// AdminDir resolves a linked worktree's git admin dir from its own .git
// pointer file (`gitdir: <path>`, absolute or relative to the worktree), with
// no subprocess, the way cli/internal/mem/identity.go does for the memory
// thread. Git composes that directory, moves it with `git worktree move` and
// removes it with `git worktree remove` and `prune`; `git clean` never reaches
// it, which is why the lease lives there.
func AdminDir(worktreePath string) (string, error) {
	pointer := filepath.Join(worktreePath, ".git")
	raw, err := os.ReadFile(pointer) // #nosec G304 -- the .git pointer of the worktree being resolved
	if err != nil {
		if fi, statErr := os.Stat(pointer); statErr == nil && fi.IsDir() {
			return "", fmt.Errorf("%s is a main checkout, %w", worktreePath, ErrNotLinkedWorktree)
		}
		return "", fmt.Errorf("%s has no readable .git pointer, %w", worktreePath, ErrNotLinkedWorktree)
	}
	target, ok := strings.CutPrefix(strings.TrimSpace(string(raw)), "gitdir:")
	if !ok {
		return "", fmt.Errorf("%s: .git is not a gitdir pointer, %w", worktreePath, ErrNotLinkedWorktree)
	}
	target = strings.TrimSpace(target)
	if !filepath.IsAbs(target) {
		target = filepath.Join(worktreePath, target)
	}
	target = filepath.Clean(target)
	if filepath.Base(filepath.Dir(target)) != "worktrees" {
		return "", fmt.Errorf("%s: .git points at %s, %w", worktreePath, target, ErrNotLinkedWorktree)
	}
	return target, nil
}

// LoadMetadata reads the worktree's lease: the admin dir first, then the
// legacy in-tree file, which a path that is not a linked worktree reaches
// too. It answers nil, nil when neither exists.
func LoadMetadata(worktreePath string) (*Metadata, error) {
	if admin, err := AdminDir(worktreePath); err == nil {
		meta, err := readLease(filepath.Join(admin, LeaseFileName))
		if err != nil || meta != nil {
			return meta, err
		}
	}
	return readLease(filepath.Join(worktreePath, MetadataFileName))
}

// SaveMetadata writes the lease into the worktree's admin dir and, for one
// release, the same bytes to the in-tree file (ADR-062): the guard a seat has
// not redeployed and the pinned binary the hooks call read the tree file. A
// path that is not a linked worktree is refused, never written in the tree.
// The mirror is written first and the admin dir last, and a mirror whose
// admin-dir write then fails is put back as it was, so an error always
// means both copies, the one a new reader prefers and the one an old guard
// reads, are as they were.
func SaveMetadata(worktreePath string, meta Metadata) error {
	admin, err := AdminDir(worktreePath)
	if err != nil {
		return fmt.Errorf("saving worktree metadata: %w", err)
	}
	data, err := marshalMetadata(meta)
	if err != nil {
		return err
	}
	mirror := filepath.Join(worktreePath, MetadataFileName)
	previous, readErr := os.ReadFile(mirror)                  // #nosec G304 -- the mirror path this package composed
	if err := os.WriteFile(mirror, data, 0o644); err != nil { // #nosec G306 -- the lease is not a secret
		return fmt.Errorf("writing the worktree metadata mirror: %w", err)
	}
	if err := os.WriteFile(filepath.Join(admin, LeaseFileName), data, 0o644); err != nil { // #nosec G306 -- the lease is not a secret
		if restoreErr := restoreMirror(mirror, previous, readErr); restoreErr != nil {
			return fmt.Errorf("writing worktree metadata: %w (and the in-tree mirror could not be put back, so it is ahead of the lease: %v)", err, restoreErr)
		}
		return fmt.Errorf("writing worktree metadata: %w", err)
	}
	return nil
}

// restoreMirror puts the in-tree mirror back to what it held before a Save
// whose admin-dir write failed: the previous bytes, or no file. Its error is
// reported alongside the one that matters, because a mirror left ahead of the
// lease is what an old guard would read.
func restoreMirror(mirror string, previous []byte, readErr error) error {
	if readErr != nil {
		return os.Remove(mirror)
	}
	return os.WriteFile(mirror, previous, 0o644) // #nosec G306 -- the lease is not a secret
}

// LeaseLegacyOnly reports whether worktreePath's lease exists only at the
// legacy in-tree location: the admin dir resolves and holds none, and the
// in-tree file is there. A path that is not a linked worktree answers false,
// as does one whose admin dir holds the lease whatever the mirror's state.
func LeaseLegacyOnly(worktreePath string) bool {
	admin, err := AdminDir(worktreePath)
	if err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(admin, LeaseFileName)); err == nil {
		return false
	}
	_, err = os.Stat(filepath.Join(worktreePath, MetadataFileName))
	return err == nil
}

func readLease(path string) (*Metadata, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- a lease path this package composed
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

func marshalMetadata(meta Metadata) ([]byte, error) {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling metadata: %w", err)
	}
	return data, nil
}
