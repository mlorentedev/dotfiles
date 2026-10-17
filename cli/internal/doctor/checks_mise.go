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
	// Once mise is on PATH, whatever the pins below say: the PATH a fresh shell
	// starts with is a fact about the deployed files, and a missing pin must
	// not hide it. Without mise there are no shims to put on PATH, and the
	// WARN above already names the remedy.
	defer checkNonInteractivePath(sys, home, rep)
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
	// Reported, never removed here: removal is the converge's, which shows it
	// in its plan first (#2375), and doctor --fix stays non-destructive.
	if orphans, err := s.Orphans(pins); err != nil {
		rep.Warn("cannot list the mise installs no pin asks for (mise ls --prunable --json): " + err.Error())
	} else if len(orphans) > 0 {
		rep.Warn(fmt.Sprintf("%d mise install(s) asked for by no pin, each behind a shim that cannot run: %s (run: dotf converge --only tools, which prunes them)", len(orphans), strings.Join(orphans, ", ")))
	}
	checkShadowingCopies(sys, home, pins, rep, fix)
}

// shellProbes are the non-interactive shells whose startup PATH decides what
// an agent, a script or a command over ssh runs. `bash -c` is not one: it
// reads no file and inherits its parent's PATH, so what it runs is the
// parent's question, not the dotfiles' (#2013 W2b).
var shellProbes = [][2]string{{"zsh", "-c"}, {"bash", "-lc"}}

// checkNonInteractivePath asks each shell above, started from an environment
// that holds only HOME and a bare system PATH, what PATH it builds: none of
// them reads the rc that runs `mise activate`, so mise's shims on that PATH,
// ahead of ~/.local/bin, are the only way they run the pins. Starting from an
// empty environment is the point: doctor's own PATH says what the shell that
// launched doctor has, not what the deployed files give a fresh one.
func checkNonInteractivePath(sys *System, home string, rep *Report) {
	if sys.GOOS == "windows" {
		return
	}
	shims := tools.MiseShimsDir(home, sys.GOOS, sys.Getenv)
	localBin := filepath.Join(home, ".local", "bin")
	// Forwarded because a real caller inherits them too: mise's data dir moves
	// the shims, and a zsh that inherits ZDOTDIR reads $ZDOTDIR/.zshenv, not
	// the ~/.zshenv deploy installs. A WARN under ZDOTDIR is then true.
	env := []string{"HOME=" + home, "PATH=/usr/bin:/bin"}
	for _, k := range []string{"MISE_DATA_DIR", "XDG_DATA_HOME", "ZDOTDIR"} {
		if v := sys.Getenv(k); v != "" {
			env = append(env, k+"="+v)
		}
	}
	var ok, bad []string
	for _, p := range shellProbes {
		if !sys.has(p[0]) {
			continue
		}
		form := p[0] + " " + p[1]
		out, err := sys.CommandOutputEnv(env, p[0], p[1], `printf %s "$PATH"`)
		if err != nil {
			bad = append(bad, fmt.Sprintf("`%s` failed: %v", form, err))
			continue
		}
		// The last line: an rc that prints something prints it first.
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		if problem := shimsAhead(lines[len(lines)-1], shims, localBin); problem != "" {
			bad = append(bad, fmt.Sprintf("`%s` %s", form, problem)+loginFileHidingProfile(sys, home, p[0]))
			continue
		}
		ok = append(ok, form)
	}
	if len(bad) > 0 {
		rep.Warn(strings.Join(bad, "; ") + " (run: dotf deploy, which installs ~/.zshenv, ~/.profile and ~/.zsh/path.sh)")
		return
	}
	if len(ok) > 0 {
		rep.Pass("non-interactive shells start with mise's shims first: " + strings.Join(ok, ", "))
	}
}

// loginFileHidingProfile names the file a login bash reads instead of the
// deployed ~/.profile: bash -l reads only the first of ~/.bash_profile,
// ~/.bash_login and ~/.profile, so `dotf deploy` alone cannot fix a home that
// holds one of the first two.
func loginFileHidingProfile(sys *System, home, shell string) string {
	if shell != "bash" {
		return ""
	}
	for _, name := range []string{".bash_profile", ".bash_login"} {
		f := filepath.Join(home, name)
		if _, err := os.Stat(f); err == nil {
			return " (" + f + " exists, so bash -l never reads ~/.profile: source ~/.zsh/path.sh from it)"
		}
	}
	return ""
}

// shimsAhead says what is wrong with a PATH that must list shims before
// localBin, or "" when nothing is.
func shimsAhead(pathList, shims, localBin string) string {
	shimsAt, localAt := -1, -1
	for i, d := range filepath.SplitList(pathList) {
		switch filepath.Clean(d) {
		case filepath.Clean(shims):
			if shimsAt < 0 {
				shimsAt = i
			}
		case filepath.Clean(localBin):
			if localAt < 0 {
				localAt = i
			}
		}
	}
	switch {
	case shimsAt < 0:
		return "starts without mise's shims (" + shims + ") on PATH"
	case localAt >= 0 && localAt < shimsAt:
		return "puts " + localBin + " ahead of mise's shims, so a copy there shadows the pin"
	}
	return ""
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
	// Not the file tools: mise has no executables to list for a zsh plugin.
	for _, p := range tools.BinaryPins(pins) {
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
