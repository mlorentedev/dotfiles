package doctor

import (
	"fmt"
	"os"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// checkModelPins reports routing pins in DEPLOYED files that the routing
// registry no longer resolves (HARNESS-067, #902).
//
// WHY THIS IS DOCTOR AND NOT CI. The repo half of this guard is a Go test, which
// runs anywhere. This half reads files on one machine — ~/.pi/agent/settings.json
// and its neighbours — and CI never sees a machine. Same split as the pi
// extension check next door: a check that cannot run where the defect lives is
// not a guard.
//
// THE DEFECT, measured 2026-08-26. The deployed pi settings carried
// `nan/deepseek-v4-flash-0731`, an id that no longer resolves — pi printed
// `Warning: No models match pattern "nan/deepseek-v4-flash-0731"` on EVERY start
// — plus three `openrouter/*` entries for a provider this repo retired in August
// 2026. Nobody noticed until an unrelated investigation read the file, because
// the repo cannot converge it: `ai/pi/settings.json` is seed-if-missing, pi owns
// the live copy (#754), and no check looked.
//
// IT NEVER WRITES. Repairing would mean a surgical merge into a file the tool
// rewrites at runtime, which is the same disposition question the hand-wired
// extension symlinks raised in #1243 — asked, not defaulted. There is
// deliberately no --fix here, and the check is not even given the flag.
//
// SEVERITY FOLLOWS CONSEQUENCE, not tidiness. A dead scalar routing pin
// (`defaultModel`) decides what a real session runs on, so it fails. A dead
// entry in a catalog array costs a warning line at startup and a stale picker
// row, so it warns. Reporting both identically would either cry wolf about a
// picker or under-report a broken default.
func checkModelPins(sys *System, cfg *Config, rep *Report) {
	rep.Section("Model pin drift")

	pins, err := harness.LoadModelPins(cfg.DotfilesDir)
	if err != nil {
		// C15: an unreadable registry is not an empty one. Both would produce
		// "no drift found"; they mean opposite things.
		rep.Fail(fmt.Sprintf("%v\n    This check reads the deployed copy; if the file exists in your checkout, re-run setup to mirror it.", err))
		return
	}
	m, err := harness.LoadModelMap(cfg.DotfilesDir)
	if err != nil {
		rep.Fail(fmt.Sprintf("routing registry unreadable, so no pin can be resolved: %v", err))
		return
	}
	qualified, bare := harness.DeclaredModels(m)

	sites := 0
	values := 0
	findings := 0

	for _, site := range pins.Sites {
		if site.Scope != "deployed" {
			continue
		}
		path := expandHome(sys, site.File)
		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				rep.Skip(fmt.Sprintf("%s not deployed on this machine", site.File))
				continue
			}
			rep.Warn(fmt.Sprintf("%s unreadable: %v", site.File, err))
			continue
		}
		sites++

		for _, p := range site.Pins {
			raw, err := harness.Extract(p, content)
			if err != nil {
				// A locator that stopped matching reports zero values, and zero
				// values look exactly like zero drift. Say which one it is.
				rep.Warn(fmt.Sprintf("%s: %v", site.File, err))
				continue
			}
			isCatalog := strings.HasSuffix(p.Locator, "[]")
			for _, v := range raw {
				values++
				switch harness.Check(p, v, qualified, bare) {
				case harness.VerdictOK:
				case harness.VerdictWrongPool:
					findings++
					rep.Warn(fmt.Sprintf("%s %s: %q is a model the map knows, but not under pool %q",
						site.File, p.ID, v, p.Pool))
				case harness.VerdictUnknown:
					if isCatalog {
						// A CATALOG entry the map does not route is normally
						// legitimate — that is #1244's whole point, and
						// `nan/gemma4` is a live NaN model nobody routes. So an
						// unrouted catalog id is not reported. Only two shapes
						// are, because only these two mean something is WRONG
						// rather than merely unrouted.
						if retired := retiredProvider(v); retired != "" {
							findings++
							rep.Warn(fmt.Sprintf("%s %s: %q names %q, a provider this repository retired\n    Catalog entry: a stale picker row, not a broken default.",
								site.File, p.ID, v, retired))
							continue
						}
						if base := staleSnapshotOf(v, p, bare); base != "" {
							findings++
							rep.Warn(fmt.Sprintf("%s %s: %q is a frozen snapshot of %q, and no longer resolves\n    Catalog entry: pi prints `No models match pattern` for it on every start.",
								site.File, p.ID, v, base))
						}
						continue
					}
					findings++
					msg := fmt.Sprintf("%s %s: %q resolves to nothing the routing registry declares", site.File, p.ID, v)
					if retired := retiredProvider(v); retired != "" {
						msg = fmt.Sprintf("%s %s: %q names %q, a provider this repository retired",
							site.File, p.ID, v, retired)
					}
					rep.Fail(msg + "\n    This decides what a real session runs on.")
				}
			}
		}
	}

	if sites == 0 {
		rep.Skip("no deployed pin sites present on this machine")
		return
	}
	if values == 0 {
		// Never render as "clean": a sweep that inspected nothing is the failure
		// this check exists to catch, one level up.
		rep.Fail("deployed pin sites were read but no pin was extracted — the locators have rotted, this is not a clean result")
		return
	}
	if findings == 0 {
		rep.Pass(fmt.Sprintf("%d deployed routing pins across %d files all resolve in the map", values, sites))
	}
}

// staleSnapshotOf reports the declared model a catalog id is a frozen snapshot
// of, or "" when it is not one.
//
// THE DISTINCTION THIS DRAWS, and why the naive version is wrong. A catalog id
// the map does not route is USUALLY fine: `nan/gemma4` is a live NaN model that
// nothing routes, exactly as `qwen3.8-flash` and `glm5.3-flash` are (#1244).
// Reporting every unrouted catalog entry would fire on a recorded decision — the
// failure this registry's own $comment warns against, and one that was written
// into the first version of this check before a real run caught it.
//
// `nan/deepseek-v4-flash-0731` is different in a way that is mechanical rather
// than a matter of taste: it is a declared id (`deepseek-v4-flash`) plus a date
// stamp. That makes it a pin to a frozen snapshot of a model that is still
// alive under its rolling name — a thing that goes stale by construction, and
// which pi rejects outright (`No models match pattern`). Only that shape, and a
// retired provider, are reported.
//
// Providers publish these as `-MMDD`, `-YYYYMMDD` or `-YYYY-MM-DD`; the suffix
// is required to be digits and separators so an ordinary versioned name like
// `qwen3.6-plus` can never match.
func staleSnapshotOf(raw string, p harness.Pin, bare map[string]bool) string {
	id := raw
	if p.Prefix != "" {
		id = strings.TrimPrefix(id, p.Prefix)
	}
	for base := range bare {
		suffix, ok := strings.CutPrefix(id, base+"-")
		if !ok || suffix == "" {
			continue
		}
		if strings.IndexFunc(suffix, func(r rune) bool {
			return (r < '0' || r > '9') && r != '-'
		}) >= 0 {
			continue
		}
		return base
	}
	return ""
}

// retiredProvider names a provider prefix this repository has retired, so the
// diagnostic says WHY an id cannot resolve rather than only that it cannot. The
// map's own $comment records both as decisions rather than oversights: openrouter
// was deleted upstream in August 2026, and codex is no longer used by this
// operator.
func retiredProvider(raw string) string {
	for _, p := range []string{"openrouter", "codex"} {
		if strings.HasPrefix(raw, p+"/") || strings.HasPrefix(raw, p+":") {
			return p
		}
	}
	return ""
}
