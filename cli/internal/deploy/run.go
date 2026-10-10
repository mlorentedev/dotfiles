package deploy

import (
	"errors"
	"fmt"
	"os"
)

// ErrRenderIncomplete is what a strict Renderer returns when the secret store
// left a placeholder unresolved (locked, absent, or unreachable). Run skips the
// entry instead of installing a copy that holds less than the installed one: a
// run nobody watches, such as converge after `dotf update`, must not trade a
// resolved secret for its placeholder.
var ErrRenderIncomplete = errors.New("secrets left unresolved")

// RunOptions is what one pass over the manifest needs from its caller.
type RunOptions struct {
	RepoRoot, Home string
	GOOS           string
	Resolve        func(string) string // {VAR} in a destination, through `dotf env`
	Render         Renderer            // nil leaves {env:VAR} placeholders as they are
	Available      func(string) bool   // is a command on PATH, for `requires`
	DryRun         bool
}

// Step is one entry's line in a run.
type Step struct {
	Name    string
	Skipped string // why the entry did not run here; empty when it ran
	// Kept marks a skip caused by an incomplete render: the entry applies
	// here, and the installed file stays until the secret store answers.
	Kept bool
	// Absent names the `requires` command missing from PATH, when that is why
	// the entry did not run: it applies to this OS, unlike a platform skip.
	Absent  string
	Outcome Outcome
}

// Tightened is a directory holding a private deployed file whose mode the run
// narrowed (or, under a plan, would narrow).
type Tightened struct {
	Dir      string
	From, To os.FileMode
}

// RunResult is a whole pass, in manifest order.
type RunResult struct {
	Steps     []Step
	Tightened []Tightened
}

// Changed names the entries the run deployed or fixed the mode of, or under a
// plan would.
func (r RunResult) Changed() []string {
	var out []string
	for _, s := range r.Steps {
		if s.Skipped == "" && s.Outcome.Changed {
			out = append(out, s.Name)
		}
	}
	return out
}

// Run deploys targets, a subset of man in manifest order, then narrows each
// directory that holds a private file the run deployed. It is the one loop
// behind `dotf deploy` and converge's configs reconciler, so the two cannot
// disagree about which entries apply on this machine.
func Run(man *Manifest, targets []Config, o RunOptions) (RunResult, error) {
	var res RunResult
	deployed := map[string]bool{}
	for _, c := range targets {
		step := Step{Name: c.Name}
		switch {
		case !c.AppliesOn(o.GOOS):
			step.Skipped = "not for " + o.GOOS
		case c.Requires != "" && (o.Available == nil || !o.Available(c.Requires)):
			step.Skipped, step.Absent = c.Requires+" not installed", c.Requires
		default:
			out, err := Deploy(c, o.RepoRoot, o.Home, o.Resolve, o.Render, o.DryRun)
			if errors.Is(err, ErrRenderIncomplete) {
				step.Skipped, step.Kept = fmt.Sprintf("kept the installed file: %v", err), true
				break
			}
			if err != nil {
				return res, err
			}
			step.Outcome = out
			deployed[c.Name] = true
		}
		res.Steps = append(res.Steps, step)
	}
	dirs, err := man.PrivateDirsOf(o.Home, o.Resolve, deployed)
	if err != nil {
		return res, err
	}
	for _, dir := range dirs {
		from, to, changed, err := TightenDir(dir, o.GOOS, o.DryRun)
		if err != nil {
			return res, fmt.Errorf("tightening %s, which holds a private deployed file: %w", dir, err)
		}
		if changed {
			res.Tightened = append(res.Tightened, Tightened{Dir: dir, From: from, To: to})
		}
	}
	return res, nil
}
