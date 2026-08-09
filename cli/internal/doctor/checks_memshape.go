package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/mlorentedev/dotfiles/cli/internal/memshape"
)

// checkMemoryShape verifies that no auto-memory MEMORY.md holds its body inside
// a YAML block scalar, and under --fix migrates any that do back to plain
// markdown (MEMORY-006 / #864).
//
// The shape is invalid state rather than a variant: the vault's scaffolding SSOT
// forbids it, nothing dotfiles emits it, and every affected file came from a
// single bulk edit on 2026-05-26. It matters because crystallize anchors every
// marker at column 0, so a wrapped file silently fails to crystallize (#857) —
// which is why the twins now refuse it outright (#862).
//
// This check is deliberately BOTH halves of incident→guard. Running it once is
// the migration; leaving it installed is what catches a recurrence. The May
// incident shipped only the rule — a template comment forbidding the shape — and
// never swept the files already in it, so the rule read as if the problem were
// solved while 17 files stayed broken for two and a half months.
//
// Mirrors checkAutoMemoryLink's contract: verify always, repair only under
// --fix, idempotent, and never silently destroy content it does not understand.
func checkMemoryShape(sys *System, rep *Report, fix bool) {
	rep.Section("Auto-memory file shape")

	projects := filepath.Join(sys.home(), ".claude", "projects")
	entries, err := os.ReadDir(projects)
	if err != nil {
		rep.Skip("no Claude projects directory — nothing to inspect")
		return
	}

	var wrapped []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(projects, e.Name(), "memory", "MEMORY.md")
		body, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if memshape.IsWrapped(string(body)) {
			wrapped = append(wrapped, path)
		}
	}
	sort.Strings(wrapped)

	if len(wrapped) == 0 {
		rep.Pass("no YAML-wrapped MEMORY.md found")
		return
	}

	if !fix {
		noun := "MEMORY.md files hold"
		if len(wrapped) == 1 {
			noun = "MEMORY.md holds"
		}
		rep.Fail(fmt.Sprintf("%d %s their body inside a YAML block scalar — crystallize cannot stamp them (#857). Run `dotf doctor --fix` to migrate.", len(wrapped), noun))
		for _, p := range wrapped {
			rep.Info("wrapped: " + p)
		}
		return
	}

	for _, p := range wrapped {
		src, err := os.ReadFile(p)
		if err != nil {
			rep.Fail("could not read " + p + ": " + err.Error())
			continue
		}
		out, err := memshape.Unwrap(string(src))
		if err != nil {
			// An unrecognised shape is left exactly as it is. Refusing beats
			// guessing on a file holding real session continuity.
			rep.Warn("left unchanged, shape not recognised: " + p + " (" + err.Error() + ")")
			continue
		}

		info, err := os.Stat(p)
		mode := os.FileMode(0o644)
		if err == nil {
			mode = info.Mode().Perm()
		}
		if err := writeAtomic(p, []byte(out), mode); err != nil {
			rep.Fail("could not write " + p + ": " + err.Error())
			continue
		}
		rep.Fix("migrated to plain markdown: " + p)
	}
}

// writeAtomic writes via a temp file in the same directory then renames, so a
// crash mid-write cannot leave a truncated MEMORY.md. Same directory because a
// cross-filesystem rename is not atomic.
func writeAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".MEMORY.md.*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, mode); err != nil {
		return err
	}
	return os.Rename(name, path)
}
