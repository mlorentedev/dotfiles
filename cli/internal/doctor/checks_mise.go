package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// checkMiseTools checks that the CLIs versions.conf marks "# mise: cli" run at
// their pin through `mise which`: the plan `dotf tools sync --dry-run` prints,
// on every OS (ADR-044, PLAT-001c W2b). It reads the versions.conf the rest of
// doctor reads, checkout first, and runs mise from HOME as the sync does, so a
// project mise.toml in the working directory cannot answer for the machine.
// With fix it runs the sync when anything is pending. Once every pin runs, it
// reports the copies in ~/.local/bin that shadow mise's, and with fix replaces
// each with a link to mise's shim.
func checkMiseTools(sys *System, cfg *Config, rep *Report, fix bool) {
	rep.Section("Pinned CLIs (mise)")
	if cfg.VersionsPath == "" {
		rep.Skip("no versions.conf found")
		return
	}
	raw, err := os.ReadFile(cfg.VersionsPath)
	if err != nil {
		rep.Fail("versions.conf unreadable: " + err.Error())
		return
	}
	all, err := tools.ParseMisePins(raw)
	if err != nil {
		rep.Fail(err.Error())
		return
	}
	pins := all.Tools
	if !sys.has("mise") {
		rep.Warn(fmt.Sprintf("mise not on PATH: the %d CLI(s) pinned in versions.conf are not managed (install mise, then run: dotf tools sync)", len(pins)))
		return
	}
	home := sys.home()
	s := miseSync(sys, all.PythonPackages, all.Latest)
	p, err := s.Plan(pins)
	if err != nil {
		rep.Fail("mise plan: " + err.Error())
		return
	}
	if fix && (p.ConfigChanged || len(p.Missing) > 0 || len(p.MissingPackages) > 0) {
		// The fix is the sync itself, the one owner of mise's config and
		// installs; checkPython only re-probes what it leaves behind.
		if _, err := s.Apply(pins); err != nil {
			rep.Fail("dotf tools sync: " + err.Error())
			return
		}
		rep.Fix("ran the mise sync (dotf tools sync)")
		if p, err = s.Plan(pins); err != nil {
			rep.Fail("mise plan: " + err.Error())
			return
		}
	}
	if p.ConfigChanged {
		rep.Warn(s.ConfigPath() + " is not what versions.conf renders (run: dotf tools sync)")
	}
	if len(p.Missing) > 0 {
		rep.Fail("not running at their pin through mise: " + strings.Join(p.Missing, ", ") + " (run: dotf tools sync)")
		return
	}
	if len(p.MissingPackages) > 0 {
		rep.Fail("python packages missing from mise's python at their pin: " + strings.Join(p.MissingPackages, ", ") + " (run: dotf tools sync)")
		return
	}
	rep.Pass(fmt.Sprintf("%d CLI(s) at their pin through mise", len(pins)))
	checkShadowingCopies(sys, home, pins, rep, fix)
}

// miseSync is the sync `dotf tools sync` runs, with mise run from HOME so a
// project mise.toml in the working directory cannot answer for the machine.
// It never asks `mise outdated`: a release upstream is not a fault here.
func miseSync(sys *System, pythonPackages []tools.MiseTool, latest []string) tools.MiseSync {
	home := sys.home()
	return tools.MiseSync{
		ConfigDir: tools.MiseConfigDir(home, sys.Getenv),
		Run: func(name string, args ...string) ([]byte, error) {
			out, err := sys.CommandOutputDir(home, name, args...)
			return []byte(out), err
		},
		Stdout: func(name string, args ...string) ([]byte, error) {
			out, err := sys.CommandStdoutDir(home, name, args...)
			return []byte(out), err
		},
		PythonPackages: pythonPackages,
		Latest:         latest,
	}
}

// checkShadowingCopies finds the executables mise provides for the pins that
// also sit in ~/.local/bin as regular files: the copies setup's own installers
// placed before mise owned these CLIs (#2013 W2). An activated shell puts mise
// first, but any PATH that lists ~/.local/bin without mise's shims ahead of it
// (a systemd unit, a launchd plist, a cron line) runs a copy no pin governs.
// mise names the executables (`bin-paths --bin-names`), so a companion such as
// uvx or age-keygen counts, not only the pin's own name.
//
// The fix replaces each copy with a link to mise's shim rather than deleting
// it: a consumer that relied on ~/.local/bin keeps finding the tool, now at its
// pin. It runs only after every pin is proven to run through mise, and only
// links to a shim that exists. A symlink is never touched: it is deliberate, or
// a link this fix made.
func checkShadowingCopies(sys *System, home string, pins []tools.MiseTool, rep *Report, fix bool) {
	args := []string{"bin-paths", "--bin-names"}
	for _, p := range pins {
		args = append(args, p.Name+"@"+p.Version)
	}
	out, err := sys.CommandStdoutDir(home, "mise", args...)
	if err != nil {
		rep.Warn("cannot list the executables mise provides (mise bin-paths --bin-names): " + err.Error())
		return
	}
	localBin := filepath.Join(home, ".local", "bin")
	seen := map[string]bool{}
	var copies, odd []string
	for _, name := range strings.Fields(out) {
		if seen[name] {
			continue
		}
		seen[name] = true
		// A path, not a name: joined onto ~/.local/bin it would reach a file
		// outside it, which the fix must never rename over.
		if name != filepath.Base(name) || name == "." || name == ".." {
			odd = append(odd, name)
			continue
		}
		if fi, err := os.Lstat(filepath.Join(localBin, name)); err == nil && fi.Mode().IsRegular() {
			copies = append(copies, name)
		}
	}
	if len(odd) > 0 {
		rep.Warn("mise bin-paths --bin-names printed entries that are not file names, left alone: " + strings.Join(odd, ", "))
	}
	if len(copies) == 0 {
		return
	}
	sort.Strings(copies)
	if !fix {
		paths := make([]string, len(copies))
		for i, name := range copies {
			paths[i] = filepath.Join(localBin, name)
		}
		rep.Warn(fmt.Sprintf("%d file(s) shadow mise's pinned CLIs for any PATH that lists %s without mise's shims ahead of it: %s (run: dotf doctor --fix, which links each to mise's shim)",
			len(copies), localBin, strings.Join(paths, ", ")))
		return
	}
	shims := tools.MiseShimsDir(home, sys.GOOS, sys.Getenv)
	for _, name := range copies {
		dst := filepath.Join(localBin, name)
		shim := filepath.Join(shims, name)
		if _, err := os.Stat(shim); err != nil {
			rep.Warn(fmt.Sprintf("%s shadows mise's pin, and mise has no shim for it at %s to link to (run: mise reshim, then dotf doctor --fix)", dst, shim))
			continue
		}
		if err := replaceWithLink(dst, shim); err != nil {
			rep.Warn("cannot link " + dst + " to mise's shim: " + err.Error())
			continue
		}
		rep.Fix("replaced " + dst + " with a link to mise's shim " + shim)
	}
}

// replaceWithLink makes dst a symlink to target in one rename, so dst is the
// old file or the link and never missing. A failed symlink leaves dst as it
// was (Windows without the privilege to create one, for instance).
func replaceWithLink(dst, target string) error {
	tmp := dst + ".dotf-link"
	_ = os.Remove(tmp) // a leftover from an interrupted run
	if err := os.Symlink(target, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
