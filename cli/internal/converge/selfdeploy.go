package converge

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/schedule"
	"github.com/mlorentedev/dotfiles/cli/internal/update"
)

// selfDeploy owns the trigger that runs `dotf update` daily (OPS-053, ADR-019):
// a launchd agent on darwin, a systemd user timer on linux, a Scheduled Task
// on windows, all rendered from update.Trigger. It is opt-in through the
// self-deploy machine role. Without the role it removes a trigger dotf wrote,
// and leaves one it did not write in place, naming the command that adopts it:
// a machine that opted in through a setup script before OPS-053 must not lose
// its self-deploy at the first converge after it.
type selfDeploy struct {
	scheduler func(Env) (schedule.Backend, error) // NativeScheduler in production
	trigger   func(Env) schedule.Job              // nil: update.Trigger for the run's OS and home
}

func (selfDeploy) Name() string        { return "self-deploy" }
func (selfDeploy) Platforms() []string { return []string{"darwin", "linux", "windows"} }

// NativeScheduler is the production scheduler, update.NativeBackend for the
// run's OS and home.
func NativeScheduler(uid int, elevated func() bool) func(Env) (schedule.Backend, error) {
	return func(e Env) (schedule.Backend, error) {
		return update.NativeBackend(e.GOOS, e.Home, os.Getenv, uid, elevated)
	}
}

const selfDeployOptIn = "dotf env role add " + env.RoleSelfDeploy

// selfDeployView is what the step read: the backend, the job, the installed
// trigger and the opt-in, or why the step does not apply here.
type selfDeployView struct {
	backend schedule.Backend
	job     schedule.Job
	state   schedule.State
	optedIn bool
	skip    string
}

func (r selfDeploy) inspect(e Env) (selfDeployView, error) {
	if r.scheduler == nil {
		return selfDeployView{skip: "no scheduler runner is wired into this registry"}, nil
	}
	// The role is read before the scheduler is reached: a machine that never
	// opted in must not fail on a scheduler dotf cannot reach, such as a
	// container with no systemd user bus (OPS-053 review round 1, F1).
	roles, err := env.LoadMachineRoles(env.MachinePath(e.Home))
	if err != nil {
		return selfDeployView{}, err
	}
	optedIn := slices.Contains(roles, env.RoleSelfDeploy)
	unreachable := func(err error) (selfDeployView, error) {
		if optedIn {
			return selfDeployView{}, err
		}
		return selfDeployView{skip: "the scheduler is unreachable and this machine is not opted in (" + selfDeployOptIn + "): " + err.Error()}, nil
	}
	b, err := r.scheduler(e)
	if errors.Is(err, schedule.ErrUnsupported) {
		return selfDeployView{skip: err.Error()}, nil
	}
	if err != nil {
		return unreachable(err)
	}
	job := update.Trigger(e.GOOS, e.Home, os.Getenv)
	if r.trigger != nil {
		job = r.trigger(e)
	}
	v := selfDeployView{backend: b, job: job, optedIn: optedIn}
	if v.state, err = b.Inspect(v.job); err != nil {
		return unreachable(err)
	}
	return v, nil
}

func (r selfDeploy) Reconcile(e Env, dryRun bool) (Result, error) {
	v, err := r.inspect(e)
	if v.skip != "" || err != nil {
		return Result{Skip: v.skip}, err
	}
	if !v.optedIn {
		return selfDeployOptedOut(v, dryRun)
	}
	if v.state.Current() {
		return Result{Detail: "dotf update " + selfDeployWhen(v.job) + " from " + v.state.Where}, nil
	}
	verb := "applied: "
	if dryRun {
		verb = "to apply: "
	}
	res := Result{Changes: len(v.state.Pending), Detail: verb + strings.Join(v.state.Pending, "; ")}
	if dryRun {
		return res, nil
	}
	return res, v.backend.Install(v.job)
}

func selfDeployOptedOut(v selfDeployView, dryRun bool) (Result, error) {
	st := v.state
	switch {
	case !st.Present:
		return Result{Skip: "this machine has not opted in (" + selfDeployOptIn + ")"}, nil
	case !st.Managed:
		return Result{Skip: st.Where + " was installed by a setup script, not by dotf, and is left as it is: " +
			selfDeployOptIn + " adopts it; with the role, `dotf env role rm " + env.RoleSelfDeploy + "` and a converge remove it"}, nil
	}
	res := Result{Changes: 1, Detail: "remove " + st.Where + " (the machine does not declare the " + env.RoleSelfDeploy + " role)"}
	if dryRun {
		return res, nil
	}
	return res, v.backend.Remove(v.job)
}

// Probe re-inspects: an opted-in machine has the trigger current and active,
// and one that opted out has no trigger dotf wrote.
func (r selfDeploy) Probe(e Env) error {
	v, err := r.inspect(e)
	st := v.state
	switch {
	case v.skip != "" || err != nil:
		return err
	case v.optedIn && !st.Current():
		return fmt.Errorf("the self-deploy trigger still differs: %s", strings.Join(st.Pending, "; "))
	case !v.optedIn && st.Present && st.Managed:
		return fmt.Errorf("%s is still installed although the machine does not declare the %s role", st.Where, env.RoleSelfDeploy)
	}
	return nil
}

// selfDeployWhen names the job's slot from the job itself, so the detail
// cannot drift from the trigger it reports (OPS-053 review round 1, F7).
func selfDeployWhen(job schedule.Job) string {
	if c := job.Daily; c != nil {
		return fmt.Sprintf("daily at %02d:%02d", c.Hour, c.Minute)
	}
	return "daily"
}
