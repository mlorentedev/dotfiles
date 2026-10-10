package harness

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// PrunedDeployDirTrees are the deploy-dir trees that hold nothing but copies of
// the checkout, so a file there that the checkout deleted is a leftover. The
// copy into them only ever added files: msi carried 45 scripts the repo had
// deleted, still on PATH through ~/.dotfiles/scripts (#2266, #802).
//
// sensitive/ and secrets/ are not here. sensitive/ holds machine-local state
// the checkout never had (env-mapping.conf, #802), and its orphans go through
// doctor's secrets check and `dotf doctor --fix`.
var PrunedDeployDirTrees = []string{".zsh", "ssh", "scripts"}

// GitRunner runs git with args in dir and returns its stdout.
type GitRunner func(dir string, args ...string) (string, error)

// ExecGit is the production GitRunner.
func ExecGit(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output() //nolint:gosec // fixed binary, args built here
	return string(out), err
}

// Orphans are the files in the pruned trees that the checkout does not have.
// Paths are slash-separated and relative to the deploy dir.
type Orphans struct {
	// Deleted are leftovers: the checkout's git history records deleting the
	// path, so it was a copy of ours. A prune removes them.
	Deleted []string
	// Unknown are files git never tracked at that path. Absence from the
	// checkout does not make them garbage (#802), so they stay, named.
	Unknown []string
	// Skipped says why the history could not be read, when it could not; every
	// orphan is then Unknown.
	Skipped string
	// Unreadable are entries under the pruned trees the scan could not read (a
	// root-owned 0700 directory, say). They were not checked for leftovers. One
	// of them must not fail the mirror: before the prune existed, the mirror
	// never read the deploy dir at all.
	Unreadable []string
}

// ScanOrphans lists the orphans of the pruned trees. It reads the checkout's
// history only when there is an orphan to classify, so a converged deploy dir
// costs no git call. A tree the checkout lacks is not scanned: that is the
// wrong checkout or a broken one, and reading every deployed file there as an
// orphan would prune the whole tree.
func ScanOrphans(repoRoot, deployDir string, git GitRunner) (Orphans, error) {
	var orphans, unreadable []string
	for _, tree := range PrunedDeployDirTrees {
		if !isDir(filepath.Join(repoRoot, tree)) || !isDir(filepath.Join(deployDir, tree)) {
			continue
		}
		err := filepath.WalkDir(filepath.Join(deployDir, tree), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if rel, rerr := filepath.Rel(deployDir, p); rerr == nil {
					unreadable = append(unreadable, filepath.ToSlash(rel))
				}
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(deployDir, p)
			if err != nil {
				return err
			}
			if _, err := os.Lstat(filepath.Join(repoRoot, rel)); errors.Is(err, fs.ErrNotExist) {
				orphans = append(orphans, filepath.ToSlash(rel))
			}
			return nil
		})
		if err != nil {
			return Orphans{}, fmt.Errorf("scanning %s for leftovers: %w", tree, err)
		}
	}
	if len(orphans) == 0 {
		return Orphans{Unreadable: unreadable}, nil
	}
	sort.Strings(orphans)

	deleted, skipped := deletedPaths(repoRoot, git)
	o := Orphans{Skipped: skipped, Unreadable: unreadable}
	for _, rel := range orphans {
		if deleted[rel] {
			o.Deleted = append(o.Deleted, rel)
		} else {
			o.Unknown = append(o.Unknown, rel)
		}
	}
	return o, nil
}

// deletedPaths is every path under the pruned trees that the checkout's
// history deleted, on any ref. --no-renames reports a rename's old path as a
// deletion, which it is for the deploy dir. A shallow clone is refused: its
// log lacks the commits that deleted things, and the empty answer would read
// as "never ours" rather than "unknown".
func deletedPaths(repoRoot string, git GitRunner) (map[string]bool, string) {
	shallow, err := git(repoRoot, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return nil, "the checkout's git history cannot be read (" + err.Error() + "), so no orphan is proven a leftover"
	}
	if strings.TrimSpace(shallow) == "true" {
		return nil, "the checkout is a shallow clone, so its history cannot prove what it deleted"
	}
	args := append([]string{"log", "--all", "--no-renames", "--diff-filter=D", "--name-only", "--pretty=format:", "--"}, PrunedDeployDirTrees...)
	out, err := git(repoRoot, args...)
	if err != nil {
		return nil, "git log failed (" + err.Error() + "), so no orphan is proven a leftover"
	}
	deleted := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			deleted[line] = true
		}
	}
	return deleted, ""
}

// PruneOrphans removes each path from deployDir, then every directory the
// removal left empty, up to but not including the tree root. A path it cannot
// remove is reported in the joined error after the rest are pruned. It refuses a path
// outside the pruned trees before removing anything, so a wrong list cannot
// reach sensitive/ or climb out of the deploy dir.
func PruneOrphans(deployDir string, rels []string) error {
	for _, rel := range rels {
		if prunedTree(rel) == "" {
			return fmt.Errorf("refusing to prune %q: not inside %v", rel, PrunedDeployDirTrees)
		}
	}
	var errs []error
	for _, rel := range rels {
		p := filepath.Join(deployDir, filepath.FromSlash(rel))
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			// Keep going: one leftover in a directory the user cannot write
			// (an old sudo setup) must not shield the rest. The failure still
			// surfaces, because a leftover left on PATH is a real defect.
			errs = append(errs, fmt.Errorf("pruning %s: %w", rel, err))
			continue
		}
		root := filepath.Join(deployDir, prunedTree(rel))
		for dir := filepath.Dir(p); dir != root && strings.HasPrefix(dir, root); dir = filepath.Dir(dir) {
			if os.Remove(dir) != nil {
				break // not empty, or already gone: stop climbing
			}
		}
	}
	return errors.Join(errs...)
}

// prunedTree is the pruned tree a clean, slash-separated relative path lies
// strictly inside, or "" when it lies in none.
func prunedTree(rel string) string {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	if clean != rel || !filepath.IsLocal(filepath.FromSlash(rel)) {
		return ""
	}
	for _, tree := range PrunedDeployDirTrees {
		if strings.HasPrefix(rel, tree+"/") {
			return tree
		}
	}
	return ""
}
