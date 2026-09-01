package doctor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// checkAgentPresence reports (HARNESS-092, #1326) whether each harness
// instructions file carries the persona roster the records render today. It
// renders the roster the same way `dotf harness presence` does and compares
// the sha the region's begin marker carries — never the text, which
// checkInstructionDrift deliberately strips — so it writes nothing and cannot
// disagree with the deploy about what "current" means.
//
// Measured 2026-08-27 on the Windows box: zero AGENT-PRESENCE regions in all
// four files, and every check green, because none looked. A target whose
// requires_command is absent is not a surface (the same gate deploy_instructions
// and checkInstructionDrift apply), and a file that does not exist is reported
// by the deploy checks, not here.
func checkAgentPresence(sys *System, rep *Report) {
	rep.Section("Agent presence (forced skills)")

	repo := resolveRepoDir(sys)
	if repo == "" {
		rep.Skip("repo not found — agent presence check skipped")
		return
	}
	manifest := filepath.Join(repo, filepath.FromSlash(harness.ManifestFile))
	recordDir, targets, err := harness.LoadPresence(manifest)
	if err != nil {
		rep.Warn("cannot read presence targets: " + err.Error())
		return
	}
	if len(targets) == 0 {
		rep.Skip("manifest declares no agents.presence targets")
		return
	}
	personas, err := harness.LoadPersonas(filepath.Join(repo, filepath.FromSlash(recordDir)))
	if err != nil {
		rep.Fail("agent records unreadable: " + err.Error())
		return
	}

	home := sys.home()
	current, drifted, skipped := 0, 0, 0
	for _, t := range targets {
		if t.RequiresCommand != "" && !sys.has(t.RequiresCommand) {
			skipped++
			continue
		}
		block := harness.BuildPresence(personas, t.Agent)
		file := filepath.Join(home, filepath.FromSlash(t.File))
		if block == "" {
			skipped++
			continue
		}
		if _, err := os.Stat(file); err != nil {
			skipped++ // the base file's absence is the deploy checks' finding
			continue
		}
		state, err := harness.PresenceStatus(file, block)
		if err != nil {
			rep.Warn(fmt.Sprintf("%s: %v", t.File, err))
			drifted++
			continue
		}
		switch state {
		case harness.PresenceCurrent:
			current++
		case harness.PresenceStale:
			rep.Warn(fmt.Sprintf("stale presence in %s (%s): the persona roster changed since it was injected (run: dotf harness presence)", t.File, t.Agent))
			drifted++
		default:
			rep.Warn(fmt.Sprintf("no presence region in %s (%s): personas' forced skills never reach this harness (run: dotf harness presence)", t.File, t.Agent))
			drifted++
		}
	}
	if drifted == 0 {
		rep.Pass(fmt.Sprintf("presence current in %d instructions file(s) (%d not compared: tool absent, file absent, or no persona targets it)", current, skipped))
	}
}
