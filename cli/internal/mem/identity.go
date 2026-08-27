package mem

import (
	"os"
	"path/filepath"
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
					Project: filepath.Base(dir),
					Branch:  headBranch(p),
				}, true
			}
			if gitdir, ok := readGitdirPointer(p); ok {
				return Identity{
					// `…/<repo>/.git/worktrees/<name>` — the repo root is three
					// levels up, and its basename is the project.
					Project:  filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(gitdir)))),
					Branch:   headBranch(gitdir),
					Worktree: filepath.Base(gitdir),
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
// work, and therefore need the machine to tell two of them apart.
func isDefaultBranch(b string) bool {
	return b == "main" || b == "master"
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
func readGitdirPointer(path string) (string, bool) {
	raw, err := os.ReadFile(path) // #nosec G304 -- the .git pointer of the cwd being resolved
	if err != nil {
		return "", false
	}
	target, ok := strings.CutPrefix(strings.TrimSpace(string(raw)), "gitdir:")
	if !ok {
		return "", false
	}
	target = filepath.Clean(strings.TrimSpace(target))
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
