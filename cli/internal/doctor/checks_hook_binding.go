package doctor

import (
	"fmt"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// checkHookBinding reports whether each harness settings file carries the hooks
// harness/manifest.json binds into it, by planning the same bind
// `dotf harness bind` and the records-bind converge step run, so the three
// cannot disagree about what "current" means. Under --fix it binds.
//
// Nothing reported this before #2232: another writer of ~/.claude/settings.json
// can remove or rewrite our hooks, and a machine without the gate hook runs
// every tool call ungated with no symptom at all.
func checkHookBinding(sys *System, rep *Report, fix bool) {
	rep.Section("Harness hook bindings")

	repo := resolveRepoDir(sys)
	if repo == "" {
		rep.Skip("repo not found — hook binding check skipped")
		return
	}
	targets, err := harness.LoadBindTargets(repo)
	if err != nil {
		rep.Fail("cannot read the bind targets: " + err.Error())
		return
	}
	home := sys.home()
	if home == "" {
		// A relative settings path would land in the working directory.
		rep.Skip("HOME is not set — hook binding check skipped")
		return
	}
	outcomes, err := harness.Bind(targets, harness.BindOptions{
		Home: home, Binary: harness.ResolveDotfPath(home), GOOS: sys.GOOS, DryRun: !fix, Has: sys.has,
	})
	for _, o := range outcomes {
		reportHookBinding(rep, o, fix)
	}
	if err != nil {
		rep.Fail("hook binding: " + err.Error())
	}
}

func reportHookBinding(rep *Report, o harness.BindOutcome, fix bool) {
	switch {
	case o.Skip != "":
		rep.Skip(fmt.Sprintf("%s: %s", o.Agent, o.Skip))
	case !o.Changed && len(o.Retired) == 0:
		rep.Pass(fmt.Sprintf("%s: hooks current in %s", o.Agent, o.File))
	case fix:
		msg := fmt.Sprintf("%s: hooks bound in %s", o.Agent, o.File)
		for _, r := range o.Retired {
			msg += fmt.Sprintf("; retired %s on %s from %s", r.ID, r.Event, r.File)
		}
		rep.Fix(msg)
	default:
		rep.Fail(fmt.Sprintf("%s: hooks in %s differ from harness/manifest.json — run `dotf doctor --fix`", o.Agent, o.File))
	}
}
