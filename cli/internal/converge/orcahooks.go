package converge

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/orca"
)

// orcaHooks re-applies the DX-006 repair to the Copilot hooks Orca generates
// (CLI-093, #1953): the hook timeouts raised to the floor and the slow POST
// swapped. Orca regenerates both files on every install, so the repair has to
// run on every converge, as the bare `dotf deploy` already does; until this
// step existed only that deploy did (#1843 B10). It runs orca.TuneHooks, the
// implementation `dotf orca tune-hooks` and the deploy share. Every OS: the
// timeout floor waives itself where Orca registers bash hooks (#2252).
type orcaHooks struct {
	now func() time.Time // the backup stamp; time.Now when unset
}

func (orcaHooks) Name() string        { return "orca-hooks" }
func (orcaHooks) Platforms() []string { return nil }

// orcaHookState is one measurement of both files. A script whose POST line the
// swap does not know is not a change the step can make: it is named and left.
type orcaHookState struct {
	rep          *orca.HookTuneReport
	unrecognised bool
	config       string
	script       string
}

// todo counts the files the repair would change.
func (s orcaHookState) todo() int {
	n := 0
	if s.rep.ConfigDrift {
		n++
	}
	if s.rep.ScriptDrift && !s.unrecognised {
		n++
	}
	return n
}

func measureOrcaHooks(home string) (orcaHookState, error) {
	cfg, scr := orca.HookPaths(home)
	st := orcaHookState{config: cfg, script: scr}
	rep, err := orca.TuneHooks(cfg, scr, orca.DefaultHookTimeout, true, time.Now)
	if err != nil {
		return st, err
	}
	st.rep = rep
	if rep.ScriptDrift {
		content, err := os.ReadFile(scr) //nolint:gosec // the user's own hook file
		if err != nil {
			return st, fmt.Errorf("read %s: %w", scr, err)
		}
		_, ok := orca.TuneScript(content)
		st.unrecognised = !ok
	}
	return st, nil
}

func (o orcaHooks) Reconcile(env Env, dryRun bool) (Result, error) {
	st, err := measureOrcaHooks(env.Home)
	if err != nil {
		return Result{}, err
	}
	if st.rep.Nothing() {
		return Result{Skip: "Orca has generated no Copilot hooks for this user"}, nil
	}
	res := Result{Changes: st.todo()}
	var parts []string
	switch {
	case res.Changes == 0:
		parts = append(parts, "hooks tuned")
	case dryRun:
		parts = append(parts, fmt.Sprintf("%d hook file(s) to tune", res.Changes))
	default:
		now := o.now
		if now == nil {
			now = time.Now
		}
		rep, err := orca.TuneHooks(st.config, st.script, orca.DefaultHookTimeout, false, now)
		if err != nil {
			return res, err
		}
		parts = append(parts, fmt.Sprintf("%d hook file(s) tuned, backed up beside themselves; restart the Copilot CLI session to pick them up", rep.Changed))
	}
	if st.unrecognised {
		parts = append(parts, st.script+" has an unrecognised POST line: review it by hand")
	}
	res.Detail = strings.Join(parts, "; ")
	return res, nil
}

// Probe holds the post-condition: no file the repair can change still needs
// it. An unrecognised script was named and is not this step's to fail on.
func (orcaHooks) Probe(env Env) error {
	st, err := measureOrcaHooks(env.Home)
	if err != nil || st.rep.Nothing() {
		return err
	}
	if st.todo() > 0 {
		return fmt.Errorf("after the repair, %s or its script still needs it", st.config)
	}
	return nil
}
