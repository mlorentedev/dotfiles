package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// checkShadowingSystemCopies finds regular files in ~/.local/bin named like a
// packages.json `system` entry's command (#2320): the copies setup-linux.sh
// downloaded from `releases/latest` before #2013 W2 handed eza and gh to the
// OS manager. ~/.local/bin comes ahead of the manager's directory on PATH, so
// such a copy shadows the managed one, and no pin or manager governs it. W2c
// (checkShadowingCopies) does the same for mise's pins and
// checkShadowedCatalogTools for npm entries.
//
// The fix removes a copy only while the manager's copy runs (`--version`),
// so it never trades a working tool for a broken one; it removes rather than
// links, because the manager's directory is already on every PATH. A copy
// that is the only one on PATH is named and kept: the presence rule counts it
// as installed, so the manager never installs its own, and removing it would
// leave no tool at all. A symlink is never touched: it is deliberate.
func checkShadowingSystemCopies(sys *System, cat tools.Catalog, rep *Report, fix bool) {
	home := sys.home()
	if home == "" {
		return
	}
	localBin := filepath.Join(home, ".local", "bin")
	var shadows, managedBy []string
	for _, t := range cat.Tools {
		cmd := t.Source.Command
		if t.Source.Type != "system" || cmd == "" || cmd != filepath.Base(cmd) {
			continue
		}
		manager, pkg := t.Source.SystemPackage(sys.GOOS)
		if pkg == "" {
			continue
		}
		stale := regularCopy(sys.GOOS, localBin, cmd)
		if stale == "" {
			continue
		}
		managed := managedCopy(sys, localBin, cmd)
		switch {
		case managed == "":
			rep.Warn(fmt.Sprintf("%s is the only %s on PATH, a copy %s does not manage: the catalog counts it as installed, so `dotf tools install` never installs %s's %s (install it through %s, then run: dotf doctor --fix)",
				stale, cmd, manager, manager, pkg, manager))
		case !fix:
			shadows = append(shadows, stale)
			managedBy = append(managedBy, managed)
		default:
			if _, err := sys.versionLine(managed); err != nil {
				rep.Warn(fmt.Sprintf("%s shadows %s, which does not run (`%s --version`: %v), so it is left in place", stale, managed, managed, err))
				continue
			}
			if err := os.Remove(stale); err != nil {
				rep.Warn("cannot remove " + stale + ": " + err.Error())
				continue
			}
			rep.Fix("removed " + stale + ", which shadowed " + managed)
		}
	}
	if len(shadows) > 0 {
		pairs := make([]string, len(shadows))
		for i := range shadows {
			pairs[i] = shadows[i] + " (ahead of " + managedBy[i] + ")"
		}
		rep.Warn(fmt.Sprintf("%d file(s) in %s shadow the OS manager's copy of a packages.json system entry, and no pin governs them: %s (run: dotf doctor --fix, which removes each once the managed copy runs)",
			len(shadows), localBin, strings.Join(pairs, ", ")))
	}
}

// regularCopy is the regular file in localBin named cmd (on Windows also
// cmd.exe), or "" when there is none or it is a link.
func regularCopy(goos, localBin, cmd string) string {
	names := []string{cmd}
	if goos == "windows" {
		names = append(names, cmd+".exe")
	}
	for _, n := range names {
		p := filepath.Join(localBin, n)
		if fi, err := os.Lstat(p); err == nil && fi.Mode().IsRegular() {
			return p
		}
	}
	return ""
}

// managedCopy is the first copy of cmd on PATH outside localBin, or "".
func managedCopy(sys *System, localBin, cmd string) string {
	skip := localBin
	if resolved, err := filepath.EvalSymlinks(localBin); err == nil {
		skip = resolved
	}
	for _, dir := range sys.dirsProviding(cmd) {
		key := dir
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			key = resolved
		}
		if key == skip {
			continue
		}
		if p := regularOrLinkedExec(sys.GOOS, dir, cmd); p != "" {
			return p
		}
	}
	return ""
}

// regularOrLinkedExec is the executable named cmd in dir, the form
// dirsProviding found there.
func regularOrLinkedExec(goos, dir, cmd string) string {
	names := []string{cmd}
	if goos == "windows" {
		names = append(names, cmd+".exe", cmd+".cmd", cmd+".ps1")
	}
	for _, n := range names {
		if p := filepath.Join(dir, n); isExecFile(p) {
			return p
		}
	}
	return ""
}
