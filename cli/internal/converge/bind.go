package converge

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// recordsBind emits the hooks declared in the manifest's `agents.bind` into
// each harness's settings file (harness.Bind, the engine behind
// `dotf harness bind`). Every OS.
//
// Before this step the bind ran only from the setup scripts, so a converge-only
// machine (macOS, ADR-045) never repaired a hook another writer had removed or
// rewritten, and nothing reported it (#2232). It runs after configs-deploy,
// which seeds the settings files the hooks are merged into; on a machine where
// legacy-setup also binds, the second bind finds the hooks current.
type recordsBind struct {
	dotf func(home string) string // the binary the hooks name; harness.ResolveDotfPath in production
	has  func(string) bool        // is a command on PATH, for requires_command
}

func (recordsBind) Name() string        { return "records-bind" }
func (recordsBind) Platforms() []string { return nil }

// noResolver is the skip reason when no dotf path resolver is wired, as the
// other steps report a runner they were not given.
const noResolver = "no dotf path resolver is wired into this registry"

func (r recordsBind) run(env Env, dryRun bool) ([]harness.BindOutcome, error) {
	targets, err := harness.LoadBindTargets(env.RepoRoot)
	if err != nil {
		return nil, err
	}
	return harness.Bind(targets, harness.BindOptions{
		Home: env.Home, Binary: r.dotf(env.Home), GOOS: env.GOOS, DryRun: dryRun, Has: r.has,
	})
}

func (r recordsBind) Reconcile(env Env, dryRun bool) (Result, error) {
	if r.dotf == nil || r.has == nil {
		return Result{Skip: noResolver}, nil
	}
	// On an error Bind still returns what it wrote before failing, and the
	// runner records a result's changes alongside its error.
	outcomes, err := r.run(env, dryRun)
	changed := bindChanges(outcomes)
	return Result{Changes: len(changed), Detail: bindDetail(outcomes, changed, dryRun)}, err
}

// Probe re-plans: after an apply, no target may still need a write or a
// retirement.
func (r recordsBind) Probe(env Env) error {
	if r.dotf == nil || r.has == nil {
		return errors.New(noResolver)
	}
	outcomes, err := r.run(env, true)
	if err != nil {
		return err
	}
	if changed := bindChanges(outcomes); len(changed) > 0 {
		return fmt.Errorf("hooks still differ from the manifest after the bind: %s", strings.Join(changed, ", "))
	}
	return nil
}

// bindChanges names each write a run made or would make: a settings file, and
// each retired hook as file (id on event).
func bindChanges(outcomes []harness.BindOutcome) []string {
	var changed []string
	for _, o := range outcomes {
		if o.Changed {
			changed = append(changed, o.File)
		}
		for _, r := range o.Retired {
			changed = append(changed, fmt.Sprintf("%s (retire %s on %s)", r.File, r.ID, r.Event))
		}
	}
	return changed
}

// bindDetail counts the targets in sync, names what changed and names the
// targets skipped, so a harness that was not bound never reads as bound.
func bindDetail(outcomes []harness.BindOutcome, changed []string, dryRun bool) string {
	current := 0
	var skipped []string
	for _, o := range outcomes {
		switch {
		case o.Skip != "":
			skipped = append(skipped, fmt.Sprintf("%s (%s)", o.Agent, o.Skip))
		case !o.Changed && len(o.Retired) == 0:
			current++
		}
	}
	parts := []string{fmt.Sprintf("%d harness(es) in sync", current)}
	if len(changed) > 0 {
		verb := "written"
		if dryRun {
			verb = "to write"
		}
		parts = append(parts, fmt.Sprintf("%s: %s", verb, strings.Join(changed, ", ")))
	}
	if len(skipped) > 0 {
		parts = append(parts, "skipped: "+strings.Join(skipped, ", "))
	}
	return strings.Join(parts, "; ")
}
