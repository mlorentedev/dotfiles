package mem

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Identity is what a working directory says about the work happening in it.
//
// ONE FUNCTION FEEDS EVERY CONSUMER, deliberately. Before this, `ThreadKey`
// derived the worktree from a naming convention while `SessionEnd` derived the
// project as `filepath.Base(cwd)` — two derivations of the same fact, in the same
// package, disagreeing. That is the divergent-parser defect this repository has
// now found five times in a week, and shipping it between two of my own
// functions would have been the sixth.
type Identity struct {
	// Project is the repository's name — the MAIN checkout's basename, even when
	// called from a linked worktree, because the vault is keyed by repository.
	Project string
	// Branch is the current branch, or "" on a detached HEAD.
	Branch string
	// Worktree is git's own name for a linked worktree, or "" in a main
	// checkout.
	Worktree string
	// DefaultBranches are the default branches the clone's remotes recorded
	// (refs/remotes/<remote>/HEAD), one per remote that recorded one.
	DefaultBranches []string
}

// RepoIdentity resolves the repository from a working directory by reading git's
// own on-disk state — no subprocess, no naming convention, so it behaves the same
// under every agent and every tool that creates worktrees.
//
// A linked worktree's `.git` is a FILE reading `gitdir: …/<repo>/.git/worktrees/<name>`;
// a main checkout's `.git` is a directory. Both state the branch in a `HEAD`
// file beside them.
//
// ok is false when the path is not in a git repository at all.
func RepoIdentity(cwd string) (Identity, bool) {
	for dir := filepath.Clean(cwd); ; {
		p := filepath.Join(dir, ".git")
		info, err := os.Lstat(p)
		if err == nil {
			if info.IsDir() {
				return Identity{
					Project:         filepath.Base(dir),
					Branch:          headBranch(p),
					DefaultBranches: remoteDefaultBranches(p),
				}, true
			}
			if gitdir, ok := readGitdirPointer(p); ok {
				return Identity{
					// `…/<repo>/.git/worktrees/<name>` — the repo root is three
					// levels up, and its basename is the project.
					Project:  filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(gitdir)))),
					Branch:   headBranch(gitdir),
					Worktree: filepath.Base(gitdir),
					// Remote refs live in the common dir, two levels above.
					DefaultBranches: remoteDefaultBranches(filepath.Dir(filepath.Dir(gitdir))),
				}, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return Identity{}, false
		}
		dir = parent
	}
}

// isDefaultBranch reports the branches that are ambient rather than a piece of
// work, and therefore need the machine to tell two of them apart: main,
// master, and whatever the remote says its default is. Hardcoding the first two
// left a repository whose default is `develop` or `trunk` with one key on every
// machine, so two machines overwrote each other's handoff (MEMORY-013, #1921).
// main and master stay ambient whatever the clone recorded, and so does the
// default of every remote, not only origin's (#2089): qualifying one branch too
// many costs two threads where one would do, while one too few is two machines
// overwriting one thread.
func (id Identity) isDefaultBranch() bool {
	b := id.Branch
	return b == "main" || b == "master" || slices.Contains(id.DefaultBranches, b)
}

// remoteDefaultBranches reads the symbolic ref git writes for each remote's
// default branch, `ref: refs/remotes/<remote>/<branch>`, from a common git dir:
// on clone for the remote it cloned from, and on `git remote set-head` for any
// other. Only the loose file exists, because packed-refs holds no symrefs. The
// walk is recursive because a remote's name may contain a slash (`foo/bar`
// keeps its HEAD at refs/remotes/foo/bar/HEAD). A HEAD that is not a symref
// into its own remote names nothing, which also rules out a remote-tracking
// branch that merely ends in `/HEAD`: that file holds a sha.
func remoteDefaultBranches(commonDir string) []string {
	root := filepath.Join(commonDir, "refs", "remotes")
	var branches []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "HEAD" {
			return nil
		}
		rel, err := filepath.Rel(root, filepath.Dir(p))
		if err != nil || rel == "." {
			return nil
		}
		raw, err := os.ReadFile(p) // #nosec G304 -- inside the resolved git dir
		if err != nil {
			return nil
		}
		remote := filepath.ToSlash(rel)
		if b, ok := strings.CutPrefix(strings.TrimSpace(string(raw)), "ref: refs/remotes/"+remote+"/"); ok && b != "" {
			branches = append(branches, b)
		}
		return nil
	})
	return branches
}

// sanitizeThread keeps a branch usable as both a markdown heading and a filename
// component. `/` is the character that actually occurs (`feat/x`) and the one
// that would break a path.
func sanitizeThread(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ' ', ':':
			return '-'
		}
		return r
	}, strings.TrimSpace(s))
}

// shortHost is the machine's name, lowercased and trimmed at the first dot, so a
// key stays readable in a heading. It appears in a key ONLY where it
// disambiguates — see ThreadKey.
func shortHost() string {
	h, err := os.Hostname()
	if err != nil || strings.TrimSpace(h) == "" {
		return "unknown-host"
	}
	h, _, _ = strings.Cut(strings.ToLower(strings.TrimSpace(h)), ".")
	return sanitizeThread(h)
}

// readGitdirPointer reads a linked worktree's `.git` file and returns the gitdir
// it names, if it points into a `worktrees/` directory.
//
// A relative pointer is relative to the `.git` file's own directory, which is
// what git writes under `worktree.useRelativePaths`. Read as given, it resolved
// against the process's working directory instead: from a subdirectory, the
// project became ".." and HEAD was read from the wrong place, so the thread
// key named the wrong line of work (MEMORY-016, #1930).
//
// A submodule's `.git` points into `<super>/.git/modules/<name>`, which is not
// a linked worktree, so it is refused and the walk resolves the superproject:
// the vault is keyed by project, and a submodule rarely has one of its own
// (#2089, pinned by TestRepoIdentityResolvesASubmoduleToItsSuperproject).
func readGitdirPointer(path string) (string, bool) {
	raw, err := os.ReadFile(path) // #nosec G304 -- the .git pointer of the cwd being resolved
	if err != nil {
		return "", false
	}
	target, ok := strings.CutPrefix(strings.TrimSpace(string(raw)), "gitdir:")
	if !ok {
		return "", false
	}
	target = strings.TrimSpace(target)
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(path), target)
	}
	target = filepath.Clean(target)
	if filepath.Base(filepath.Dir(target)) != "worktrees" {
		return "", false
	}
	if name := filepath.Base(target); name == "" || name == "." {
		return "", false
	}
	return target, true
}

// headBranch reads `HEAD` inside a git dir. A detached HEAD holds a raw sha
// rather than a `ref:` line, and yields "" — the caller decides what to do with
// that rather than being handed a plausible-looking wrong answer.
func headBranch(gitDir string) string {
	raw, err := os.ReadFile(filepath.Join(gitDir, "HEAD")) // #nosec G304 -- inside the resolved git dir
	if err != nil {
		return ""
	}
	ref, ok := strings.CutPrefix(strings.TrimSpace(string(raw)), "ref:")
	if !ok {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(ref), "refs/heads/")
}
