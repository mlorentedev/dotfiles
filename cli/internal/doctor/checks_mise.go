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
// Once every pin runs, it reports the copies in ~/.local/bin that shadow mise's,
// and with fix removes them.
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
	pins, err := tools.ParseMiseTools(raw)
	if err != nil {
		rep.Fail(err.Error())
		return
	}
	if !sys.has("mise") {
		rep.Warn(fmt.Sprintf("mise not on PATH: the %d CLI(s) pinned in versions.conf are not managed (install mise, then run: dotf tools sync)", len(pins)))
		return
	}
	home := sys.home()
	s := tools.MiseSync{
		ConfigDir: tools.MiseConfigDir(home, sys.Getenv),
		Run: func(name string, args ...string) ([]byte, error) {
			out, err := sys.CommandOutputDir(home, name, args...)
			return []byte(out), err
		},
		Stdout: func(name string, args ...string) ([]byte, error) {
			out, err := sys.CommandStdoutDir(home, name, args...)
			return []byte(out), err
		},
	}
	p, err := s.Plan(pins)
	if err != nil {
		rep.Fail("mise plan: " + err.Error())
		return
	}
	if p.ConfigChanged {
		rep.Warn(s.ConfigPath() + " is not what versions.conf renders (run: dotf tools sync)")
	}
	if len(p.Missing) > 0 {
		rep.Fail("not running at their pin through mise: " + strings.Join(p.Missing, ", ") + " (run: dotf tools sync)")
		return
	}
	rep.Pass(fmt.Sprintf("%d CLI(s) at their pin through mise", len(pins)))
	checkShadowingCopies(sys, home, pins, rep, fix)
}

// checkShadowingCopies finds the executables mise provides for the pins that
// also sit in ~/.local/bin as regular files: the copies setup's own installers
// placed before mise owned these CLIs (#2013 W2). An activated shell puts mise
// first, but a GUI app, launchd or cron finds ~/.local/bin and runs a copy no
// pin governs. mise names the executables (`bin-paths --bin-names`), so a
// companion such as uvx or age-keygen counts, not only the pin's own name.
//
// It runs only after every pin is proven to run through mise, which is what
// makes removing a copy safe. A symlink is left alone: someone made it on
// purpose.
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
	var copies []string
	for _, name := range strings.Fields(out) {
		p := filepath.Join(localBin, name)
		if seen[p] {
			continue
		}
		seen[p] = true
		if fi, err := os.Lstat(p); err == nil && fi.Mode().IsRegular() {
			copies = append(copies, p)
		}
	}
	if len(copies) == 0 {
		return
	}
	sort.Strings(copies)
	if !fix {
		rep.Warn(fmt.Sprintf("%d file(s) in %s shadow mise's pinned CLIs for a process without mise activated (a GUI app, launchd, cron): %s (run: dotf doctor --fix)",
			len(copies), localBin, strings.Join(copies, ", ")))
		return
	}
	for _, p := range copies {
		if err := os.Remove(p); err != nil {
			rep.Warn("cannot remove " + p + ": " + err.Error())
			continue
		}
		rep.Fix("removed " + p + ", a copy that shadowed mise's pinned CLI")
	}
}
