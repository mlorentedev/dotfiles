package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/mem"
	"github.com/mlorentedev/dotfiles/cli/internal/memlink"
	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// threadHeading is a thread entry's heading in MEMORY.md.
var threadHeading = regexp.MustCompile(`(?m)^### thread: (\S+)`)

// checkHandoffThreads reports a worktree whose handoff was written and whose
// entry has since left MEMORY.md (HARNESS-185, #1881).
//
// `dotf mem handoff-write` stops one session overwriting another's thread, but
// a hand edit, a wrongly resolved merge conflict or an older agent can still
// delete an entry, and nothing said so: the owner found out when the session
// that owned the thread next looked.
//
// The proof a handoff was written is the per-thread file `threads/<key>.md`
// (MEMORY-008, ADR-056), not a journal under sessions/: a journal name cannot be
// split back into a key. The key and the file name come from mem.ThreadKey and
// mem.ThreadFileName, the writer's own functions, so the check cannot disagree
// with the writer about what a thread is. The sanctioned archive step moves the
// file to archive/ as it deletes the entry, so an archived thread never trips it.
//
// Read-only and WARN-only by contract: restoring an entry needs a human to say
// whether the thread is live, so there is no --fix and no path to rep.Fail.
func checkHandoffThreads(sys *System, start string, rep *Report) {
	rep.Section("Handoff threads")

	trees, memDir, ok := handoffProject(sys, start, rep)
	if !ok {
		return
	}
	entries, threadsPath, ok := handoffStore(memDir, rep)
	if !ok {
		return
	}

	checked, warnings := lostHandoffs(trees, threadsPath, entries)
	for _, w := range warnings {
		rep.Warn(w)
	}
	if len(warnings) == 0 {
		rep.Pass(fmt.Sprintf("every worktree with a handoff file keeps its entry (%d checked)", checked))
	}
}

// handoffProject lists the repository's worktrees and finds its vault memory
// directory, reporting the SKIP itself when either is missing.
func handoffProject(sys *System, start string, rep *Report) ([]worktree.RawWorktree, string, bool) {
	vault := sys.vaultPath()
	if vault == "" {
		rep.Skip("no vault resolves on this machine (VAULT_PATH, machine.json) — no handoffs to check")
		return nil, "", false
	}
	out, err := sys.CommandOutputDir(start, "git", "worktree", "list", "--porcelain")
	if err != nil {
		rep.Skip("git worktrees unreadable from " + start + " — no handoffs to check")
		return nil, "", false
	}
	trees, err := worktree.ParsePorcelain(out)
	if err != nil || len(trees) == 0 {
		rep.Skip("no git worktrees found from " + start + " — no handoffs to check")
		return nil, "", false
	}
	// The first entry is the main checkout, whose directory names the project the
	// way the vault does; start may be a linked worktree named after its branch.
	main := trees[0].Path
	memDir := memlink.VaultMemory(main, filepath.Base(main), vault)
	if memDir == "" {
		rep.Skip("no vault project for " + filepath.Base(main) + " — no handoffs to check")
		return nil, "", false
	}
	return trees, memDir, true
}

// handoffStore reads the entry headings of MEMORY.md and locates threads/,
// reporting the SKIP itself when the project has no per-thread store.
func handoffStore(memDir string, rep *Report) (map[string]bool, string, bool) {
	memory, err := os.ReadFile(filepath.Join(memDir, "MEMORY.md")) // #nosec G304 -- under the resolved vault project
	if err != nil {
		rep.Skip("no MEMORY.md in " + memDir + " — no handoffs to check")
		return nil, "", false
	}
	threadsPath := filepath.Join(memDir, mem.ThreadsDir)
	if info, err := os.Stat(threadsPath); err != nil || !info.IsDir() {
		rep.Skip("no " + mem.ThreadsDir + "/ beside MEMORY.md: this project keeps its handoffs inline, which this check does not cover")
		return nil, "", false
	}
	entries := map[string]bool{}
	for _, m := range threadHeading.FindAllStringSubmatch(string(memory), -1) {
		entries[m[1]] = true
	}
	return entries, threadsPath, true
}

// lostHandoffs returns how many worktrees have a handoff file and one warning
// for each whose entry is missing.
func lostHandoffs(trees []worktree.RawWorktree, threadsPath string, entries map[string]bool) (int, []string) {
	checked := 0
	var warnings []string
	for _, t := range trees {
		if t.Bare {
			continue
		}
		key := mem.ThreadKey(t.Path)
		name, err := mem.ThreadFileName(key)
		if err != nil {
			continue
		}
		file := filepath.Join(threadsPath, name)
		if info, err := os.Stat(file); err != nil || !info.Mode().IsRegular() {
			continue // never handed off: nothing to lose
		}
		checked++
		if hasThreadEntry(entries, key) {
			continue
		}
		warnings = append(warnings, fmt.Sprintf("thread %q (worktree %s) has a handoff file but no `### thread: %s` entry in MEMORY.md — %s still holds its body. "+
			"Re-run `dotf mem handoff-write --thread %s` with that body, or, if the thread is finished, move the file to archive/ (the archive step).",
			key, t.Path, key, file, key))
	}
	return checked, warnings
}

// hasThreadEntry reports whether MEMORY.md holds an entry for key: its own
// heading, or a fork `<key>+<agent>` (handoff-write keeps another agent's block
// by writing to the forked key).
func hasThreadEntry(entries map[string]bool, key string) bool {
	if entries[key] {
		return true
	}
	for heading := range entries {
		if strings.HasPrefix(heading, key+"+") {
			return true
		}
	}
	return false
}
