package doctor

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"time"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/schedule"
	"github.com/mlorentedev/dotfiles/cli/internal/update"
)

const (
	selfDeployRemedy = "`dotf converge --only self-deploy`"
	selfDeployOptIn  = "dotf env role add " + envpkg.RoleSelfDeploy
	// selfDeployStale is how old the last run may be before the daily trigger
	// reads as not running: three missed nights, so one powered-off night or a
	// laptop asleep over a weekend is not a warning.
	selfDeployStale = 72 * time.Hour
)

// checkSelfDeploy (OPS-053 AC7) reports the opt-in self-deploy: whether the
// machine declares the role, whether the trigger the converge step owns is
// installed as rendered, and what the last `dotf update` did. It inspects the
// step's own scheduler, read-only, and reads the record update writes.
func checkSelfDeploy(sys *System, rep *Report) {
	if sys.Scheduler == nil {
		return
	}
	backend, err := sys.Scheduler(sys.GOOS, sys.home())
	if errors.Is(err, schedule.ErrUnsupported) {
		return // no scheduler on this OS: the step does not run here either
	}
	rep.Section("Self-deploy")
	if err != nil {
		rep.Warn("cannot reach the scheduler: " + err.Error())
		return
	}
	home := sys.home()
	roles, err := envpkg.LoadMachineRoles(envpkg.MachinePath(home))
	if err != nil {
		rep.Warn(err.Error())
		return
	}
	optedIn := slices.Contains(roles, envpkg.RoleSelfDeploy)
	st, err := backend.Inspect(update.Trigger(sys.GOOS, home, sys.Getenv))
	switch {
	case err != nil:
		rep.Warn("cannot read the self-deploy trigger: " + err.Error())
		return
	case !optedIn && !st.Present:
		rep.Skip("this machine has not opted in (" + selfDeployOptIn + ")")
		return
	case !optedIn && !st.Managed:
		rep.Warn(st.Where + " was installed by a setup script, not by dotf, and nothing keeps it current: " +
			selfDeployOptIn + " and " + selfDeployRemedy + " adopt it")
		return
	case !optedIn:
		rep.Warn(st.Where + " is installed but the machine does not declare the " + envpkg.RoleSelfDeploy +
			" role; a converge removes it (" + selfDeployRemedy + ")")
		return
	case !st.Current():
		rep.Warn(st.Where + " differs from the declared trigger: " + strings.Join(st.Pending, "; ") + "; run " + selfDeployRemedy)
		return
	}
	checkSelfDeployRun(sys, home, st.Where, rep)
}

// checkSelfDeployRun reports the last `dotf update` an opted-in machine ran.
func checkSelfDeployRun(sys *System, home, where string, rep *Report) {
	state, err := envpkg.StateDirFor(sys.Getenv, home)
	if err != nil {
		rep.Warn(err.Error())
		return
	}
	r, err := update.ReadRecord(state)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		rep.Warn(where + " is installed, but no run recorded yet: the first runs at 03:00, or run `dotf update`")
		return
	case err != nil:
		rep.Warn("cannot read the last self-deploy run: " + err.Error())
		return
	}
	age := sys.Now().Sub(r.At)
	msg, _, _ := strings.Cut(r.Message, "\n")
	last := fmt.Sprintf("last `dotf update` %s (%s)", ago(age), r.Status)
	switch {
	case r.Failed():
		rep.Fail(last + " failed: " + r.Error + " — " + msg)
	case age > selfDeployStale:
		rep.Warn(last + ": the daily trigger has not run since; check " + where)
	case r.Stuck():
		rep.Warn(last + ": " + msg + " — every run skips the same way until the checkout is fixed")
	default:
		rep.Pass("daily from " + where + "; " + last + ": " + msg)
	}
}

// ago renders an age in hours under two days, in days past that.
func ago(d time.Duration) string {
	if d < 48*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%d days ago", int(d.Hours()/24))
}
