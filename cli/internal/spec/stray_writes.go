package spec

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// TreeSnapshot is the working tree's state as `git status` reports it: one
// entry per path, its two-letter status as the value. A reviewer's only
// sanctioned output is review.md (HARNESS-142, #1649), so comparing a snapshot
// taken before the run with one taken after names what else it left behind:
// untracked probes, scratch dumps, staged restores.
type TreeSnapshot struct {
	prefix  string // the spec folder, relative to the repository root
	entries map[string]string
	// movedIn holds the sources, outside the spec folder, of renames into it:
	// the entries name only the destination, which the folder filter drops.
	// A copy's source is not recorded: a copy leaves it as committed, so the
	// reviewer's checkout at HEAD holds the same file and nothing is hidden
	// from it (#3223). Only a rename deletes its source.
	movedIn []string
}

// SnapshotTree records the state of the repository holding review. Paths under
// review's own folder are the review's sanctioned output and are left out.
func SnapshotTree(review string) (TreeSnapshot, error) {
	dir := filepath.Dir(review)
	prefix, err := gitOutput(dir, "rev-parse", "--show-prefix")
	if err != nil {
		return TreeSnapshot{}, err
	}
	out, err := gitOutput(dir, "status", "--porcelain", "-z", "--untracked-files=all")
	if err != nil {
		return TreeSnapshot{}, err
	}
	snap := TreeSnapshot{prefix: strings.TrimSpace(prefix), entries: map[string]string{}}
	fields := strings.Split(out, "\x00")
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if len(f) < 4 {
			continue
		}
		code, path := f[:2], f[3:]
		inside := snap.prefix != "" && strings.HasPrefix(path, snap.prefix)
		if code[0] == 'R' || code[0] == 'C' {
			i++ // -z writes a rename's source as the next field
			if code[0] == 'R' && inside && i < len(fields) && !strings.HasPrefix(fields[i], snap.prefix) {
				snap.movedIn = append(snap.movedIn, fields[i])
			}
		}
		if inside {
			continue
		}
		snap.entries[path] = code
	}
	return snap, nil
}

// CallerDirt lists, sorted, the tracked changes and untracked files outside
// specDir in the repository holding it (HARNESS-142 AC5). A reviewer reads a
// fresh worktree at the reviewed commit, so these are what it will not see;
// the spec folder's own content is carried into that worktree and is left out.
func CallerDirt(specDir string) ([]string, error) {
	snap, err := SnapshotTree(filepath.Join(specDir, "review.md"))
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(snap.entries)+len(snap.movedIn))
	for path := range snap.entries {
		paths = append(paths, path)
	}
	paths = append(paths, snap.movedIn...)
	sort.Strings(paths)
	return paths, nil
}

// StrayWrites lists the paths whose state after differs from before, sorted.
// A file that was already dirty and is changed again keeps its status and is
// not seen: the snapshot compares states, not contents. A path only the before
// side names was dirty and is now clean or gone, such as an operator's
// untracked file the reviewer deleted.
func (before TreeSnapshot) StrayWrites(after TreeSnapshot) []string {
	var stray []string
	for path, code := range after.entries {
		if before.entries[path] != code {
			stray = append(stray, path)
		}
	}
	for path := range before.entries {
		if _, ok := after.entries[path]; !ok {
			stray = append(stray, path)
		}
	}
	sort.Strings(stray)
	return stray
}

// Folder is the spec folder the snapshot excludes, as the repository names it.
func (s TreeSnapshot) Folder() string { return s.prefix }

func gitOutput(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		return "", fmt.Errorf("git %s in %s: %w", args[0], dir, err)
	}
	return string(out), nil
}
