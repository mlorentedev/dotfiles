package converge

import (
	"errors"
	"fmt"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// Registry is the ordered list of reconcilers a run drives. The order extends
// ADR-041 decision 4: records come before anything that reads them (ADR-045
// decision 4). Later reconcilers are appended by the rows that port them.
func Registry() []Reconciler {
	return []Reconciler{recordsMirror{}}
}

// recordsMirror mirrors harness/ and every manifest target from the checkout
// into the deploy dir (harness.Mirror). It is the first step of the records
// phase: the instruction files and the hook bindings read what it mirrors.
type recordsMirror struct{}

func (recordsMirror) Name() string        { return "records-mirror" }
func (recordsMirror) Platforms() []string { return nil }

func (recordsMirror) Reconcile(env Env, dryRun bool) (Result, error) {
	mirror, verb := harness.Mirror, "updated"
	if dryRun {
		mirror, verb = harness.PlanMirror, "to write"
	}
	res, err := mirror(env.RepoRoot, env.DeployDir)
	if errors.Is(err, harness.ErrCheckoutIsDeployDir) {
		return Result{Detail: "the checkout is the deploy dir; nothing to mirror"}, nil
	}
	if err != nil {
		return Result{}, err
	}
	return Result{
		Changes: res.Updated,
		Detail: fmt.Sprintf("harness/ + %d target(s) → %s (%d %s, %d unchanged)",
			len(res.Targets), env.DeployDir, res.Updated, verb, res.Unchanged),
	}, nil
}

// Probe re-plans the mirror: after an apply, nothing may still differ.
func (recordsMirror) Probe(env Env) error {
	res, err := harness.PlanMirror(env.RepoRoot, env.DeployDir)
	if errors.Is(err, harness.ErrCheckoutIsDeployDir) {
		return nil
	}
	if err != nil {
		return err
	}
	if res.Updated > 0 {
		return fmt.Errorf("%d file(s) in %s still differ from the checkout", res.Updated, env.DeployDir)
	}
	return nil
}
