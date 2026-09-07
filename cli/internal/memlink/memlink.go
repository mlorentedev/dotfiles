// Package memlink is the OS-agnostic vault->memory link primitive (CLI-025 PR2b /
// MEMORY-002). It links an agent's per-project memory directory to its source in
// the knowledge vault, so agent memory lives in the single sink (the vault) and is
// surfaced wherever the agent expects it.
//
// It converges TWO drifting twins into one Go noun: the POSIX
// scripts/ensure-memory-symlink.sh (symlink) and the Windows-only PowerShell
// junction hook (the MEMORY-002 R4 gap the shell script deferred). The vault-source
// resolution is shared and OS-agnostic; only the final link creation branches —
// os.Symlink on POSIX, a directory junction on Windows (which, unlike a symlink,
// needs no special privilege). It is consumed by the Claude session-start adapter
// and by `dotf doctor --fix` (#551).
package memlink

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Ensure links target (an agent's per-project memory dir) to its vault source,
// idempotently and best-effort. It returns the "[auto-memory] Created …" message
// when it creates the link, and "" on every no-op. vault is injected (the caller
// resolves it via vault.ResolveVault); project "" defaults to filepath.Base(cwd).
//
// Resilience: a session-start hook must never crash, so a failed link creation is a
// silent no-op (returns "", nil), mirroring the shell twin's `ln … || true`.
func Ensure(cwd, target, project, vault string) (string, error) {
	if project == "" {
		project = filepath.Base(cwd)
	}

	// Already linked, or a non-empty real dir (the agent's own data)? Leave it.
	if isLink(target) {
		return "", nil
	}
	if isDir(target) && dirNotEmpty(target) {
		return "", nil
	}

	src := resolveVaultMemory(cwd, project, vault)
	if src == "" {
		return "", nil
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	// Remove an empty placeholder target dir so the link can be created.
	if isDir(target) {
		_ = os.Remove(target)
	}
	if err := createLink(src, target); err != nil {
		return "", nil // best-effort: a failed link must never fail the session
	}
	return fmt.Sprintf("[auto-memory] Created %s for %s", linkNoun(), project), nil
}

// LinkState classifies an auto-memory target for reporting — the read-only
// counterpart to Ensure. Ensure mutates; Status only observes, so `dotf doctor`
// can report PASS/WARN/FAIL/SKIP without the side effect of creating a link.
type LinkState int

const (
	// StateLinked: target is already a symlink/junction. Healthy.
	StateLinked LinkState = iota
	// StateRealDir: target is a non-empty real directory — the agent's own data.
	// Ensure deliberately leaves it untouched; doctor must NOT destroy it either.
	// A manual reconcile into the vault is required (knowledge#120 divergence).
	StateRealDir
	// StateRepairable: a vault source exists and the target is missing or empty,
	// so Ensure (or doctor --fix) would create the link.
	StateRepairable
	// StateNoSource: no vault source resolves for this project — nothing to link.
	StateNoSource
)

// Status classifies target without mutating it, mirroring Ensure's decision
// order exactly so the two never disagree about what Ensure would do.
func Status(cwd, target, project, vault string) LinkState {
	if project == "" {
		project = filepath.Base(cwd)
	}
	if isLink(target) {
		return StateLinked
	}
	if isDir(target) && dirNotEmpty(target) {
		return StateRealDir
	}
	if resolveVaultMemory(cwd, project, vault) == "" {
		return StateNoSource
	}
	return StateRepairable
}

// ClaudeProjectKey encodes a working directory into Claude Code's per-project key
// — the directory name under ~/.claude/projects. Claude maps every path separator
// AND the Windows drive colon to '-', so `/home/me/proj` and `C:\Users\me\proj`
// become `-home-me-proj` and `C--Users-me-proj`. The retired shell twin mapped
// only '/', silently producing the wrong key — and thus the wrong junction target
// — on Windows (the root cause of the unlinked auto-memory dir, #551).
//
// '.' maps to '-' as well (#1553): Claude writes `svqtriana.github.io` as
// `svqtriana-github-io`, so a key that kept the dot named a directory Claude
// never reads — no vault link at session start, a second MEMORY.md, and a
// crystallize that stamped nothing. Any further character Claude mangles is
// unknown; this list is what has been observed.
func ClaudeProjectKey(cwd string) string {
	return strings.NewReplacer("/", "-", `\`, "-", ":", "-", ".", "-").Replace(cwd)
}

// ClaudeMemoryTarget is the per-project auto-memory directory Claude surfaces:
// `<home>/.claude/projects/<ClaudeProjectKey(cwd)>/memory`. Shared by the
// session-start adapter (which creates the link) and `dotf doctor` (which
// verifies it), so the two compute an identical path on every OS.
func ClaudeMemoryTarget(home, cwd string) string {
	return filepath.Join(home, ".claude", "projects", ClaudeProjectKey(cwd), "memory")
}

// resolveVaultMemory finds the vault memory source for a project via the three
// conventions in precedence order, returning "" when none resolves to a real dir.
// This is the agent- and OS-agnostic core shared by every caller.
func resolveVaultMemory(cwd, project, vault string) string {
	// 1) 10_projects/<project>/memory (personal projects convention).
	if vm := filepath.Join(vault, "10_projects", project, "memory"); isDir(vm) {
		return vm
	}
	// 2) <cwd>/memory when CWD is inside the vault itself.
	if isUnder(cwd, vault) {
		if vm := filepath.Join(cwd, "memory"); isDir(vm) {
			return vm
		}
	}
	// 3) 50_work/45-development/<family>/<component>/memory (nested work-SDK repos).
	return resolveWorkSDK(cwd, vault)
}

// resolveWorkSDK matches CWD path segments against vault family/component dir slugs,
// faithfully reproducing the shell's break-on-first-match: it commits to the first
// (family, component) slug pair found and yields its memory/ dir only if that dir
// exists — it does NOT keep scanning for a later pair that happens to have memory/.
func resolveWorkSDK(cwd, vault string) string {
	devDir := filepath.Join(vault, "50_work", "45-development")
	if !isDir(devDir) {
		return ""
	}
	cwdSlug := slugify(cwd, true)

	candidate := ""
	families, _ := os.ReadDir(devDir)
outer:
	for _, fam := range families {
		if !fam.IsDir() || !strings.Contains(cwdSlug, slugify(fam.Name(), false)) {
			continue
		}
		comps, _ := os.ReadDir(filepath.Join(devDir, fam.Name()))
		for _, comp := range comps {
			if comp.IsDir() && strings.Contains(cwdSlug, slugify(comp.Name(), false)) {
				candidate = filepath.Join(devDir, fam.Name(), comp.Name(), "memory")
				break outer
			}
		}
	}
	if candidate != "" && isDir(candidate) {
		return candidate
	}
	return ""
}

// slugify lowercases s and keeps only [a-z0-9] (plus '/' when keepSlash), mirroring
// the shell's `tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9[/]'`.
func slugify(s string, keepSlash bool) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case keepSlash && r == '/':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// createLink makes a directory link from target to src: a junction on Windows
// (no privilege required, unlike os.Symlink there) and a symlink on POSIX.
// Implemented per-OS in memlink_windows.go / memlink_unix.go (HARNESS-050):
// the Windows junction needs cmd.exe-specific quoting os/exec's ordinary argv
// escaping does not provide, which pulled it out of this shared file.

// linkNoun is the OS-accurate word for the created link, used in the message.
func linkNoun() string {
	if runtime.GOOS == "windows" {
		return "junction"
	}
	return "symlink"
}

// isDir reports whether p exists and is a directory.
func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// isLink reports whether p is a symlink (POSIX) or a junction/reparse point
// (Windows). POSIX symlinks carry ModeSymlink; Windows junctions — which `dotf`
// uses because they need no privilege — surface via Lstat as ModeIrregular (NOT
// ModeSymlink) on the Go 1.26 toolchain (verified empirically), so accept both.
// A plain directory is ModeDir and matches neither, so this never misfires on
// the agent's own real memory dir.
func isLink(p string) bool {
	info, err := os.Lstat(p)
	return err == nil && info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0
}

// dirNotEmpty reports whether p is a readable directory holding at least one entry
// (dotfiles included), mirroring the shell's `[ -d … ] && [ "$(ls -A …)" ]`.
func dirNotEmpty(p string) bool {
	entries, err := os.ReadDir(p)
	return err == nil && len(entries) > 0
}

// isUnder reports whether path is the vault prefix (string-prefix, matching the
// shell's `case "$CWD" in "$VAULT"*)`).
func isUnder(path, base string) bool {
	return base != "" && strings.HasPrefix(path, base)
}
