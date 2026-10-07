package doctor

import (
	"fmt"
	"os"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// checkMiseTools checks that the CLIs versions.conf marks "# mise: cli" run at
// their pin through `mise which`: the plan `dotf tools sync --dry-run` prints,
// on every OS (ADR-044, PLAT-001c W2b). It reads the versions.conf the rest of
// doctor reads, checkout first, and runs mise from HOME as the sync does, so a
// project mise.toml in the working directory cannot answer for the machine.
func checkMiseTools(sys *System, cfg *Config, rep *Report) {
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
}
