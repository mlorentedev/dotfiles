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
	"os/exec"
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
func createLink(src, target string) error {
	if runtime.GOOS == "windows" {
		// `mklink /J <link> <existing-target>` is a cmd builtin, so it runs via
		// `cmd /c`. Args are passed as argv elements (not concatenated into a shell
		// string), and both paths are CLI-resolved vault/.claude locations, not user
		// input — no shell-metachar surface.
		return exec.Command("cmd", "/c", "mklink", "/J", target, src).Run()
	}
	return os.Symlink(src, target)
}

// linkNoun is the OS-accurate word for the created link, used in the message.
func linkNoun() string {
	if runtime.GOOS == "windows" {
		return "junction"
	}
	return "symlink"
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// isLink reports whether p is a symlink (POSIX) or a junction/reparse point
// (Windows). Go surfaces both as ModeSymlink via Lstat on the supported toolchain.
func isLink(p string) bool {
	info, err := os.Lstat(p)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func dirNotEmpty(p string) bool {
	entries, err := os.ReadDir(p)
	return err == nil && len(entries) > 0
}

// isUnder reports whether path is the vault prefix (string-prefix, matching the
// shell's `case "$CWD" in "$VAULT"*)`).
func isUnder(path, base string) bool {
	return base != "" && strings.HasPrefix(path, base)
}
