package doctor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/deploy"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
)

// checkDeployManifest reports (AI-039, #1322) whether every entry of
// ai/deploy.json is installed as the manifest says. It asks the deploy package
// for its plan — the same compare `dotf deploy` runs, so the two can never
// disagree about what "in sync" means — and never for a deploy: PlanConfig
// touches nothing, which matters because the old flow staged before it
// compared and a diagnostic built on it would have created ~/.copilot/ while
// asking whether ~/.copilot/settings.json was in sync.
//
// Two kinds of entry are not compared, and the PASS line says how many: a
// rendered one (its installed content is only known after `secrets render`,
// which needs the daemon; doctor stays read-only) and one whose `requires`
// command is absent (deploy skips it too, and a row for a tool the box does
// not carry is a WARN no remedy can clear — #843). Drift is a WARN, not a
// FAIL: a tool that co-owns its file (Copilot's `/model` writes `model`) may
// legitimately have moved a managed key, and the remedy is one command.
func checkDeployManifest(sys *System, rep *Report) {
	rep.Section("Deployed agent configs (ai/deploy.json)")

	repo := resolveRepoDir(sys)
	if repo == "" {
		rep.Skip("repo not found — deploy manifest check skipped")
		return
	}
	raw, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(deploy.ManifestRel))) //nolint:gosec // repo-relative, fixed name
	if err != nil {
		rep.Warn("cannot read " + deploy.ManifestRel + ": " + err.Error())
		return
	}
	man, err := deploy.ParseManifest(raw)
	if err != nil {
		rep.Fail(err.Error())
		return
	}

	home := sys.home()
	inSync, drifted, notCompared := 0, 0, 0
	for _, c := range man.Configs {
		if c.Render || (c.Requires != "" && !sys.has(c.Requires)) {
			notCompared++
			continue
		}
		p, err := deploy.PlanConfig(c, repo, home, env.ResolvePath)
		if err != nil {
			rep.Warn(fmt.Sprintf("%s: %v (run: dotf deploy %s)", c.Name, err, c.Name))
			drifted++
			continue
		}
		if p.Changed {
			rep.Warn(fmt.Sprintf("drift: %s — %s is not what %s deploys (run: dotf deploy %s)", c.Name, p.Dst, c.Src, c.Name))
			drifted++
			continue
		}
		inSync++
	}
	if drifted == 0 {
		rep.Pass(fmt.Sprintf("%d deployed config(s) in sync with %s (%d not compared: rendered, or tool absent)", inSync, deploy.ManifestRel, notCompared))
	}
}
