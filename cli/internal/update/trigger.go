package update

import (
	"path"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/schedule"
)

// TriggerNames are the self-deploy trigger's names. The systemd units and the
// task keep the names setup-linux.sh and setup-windows.ps1 gave them, so a
// machine that opted in before OPS-053 has its trigger rewritten in place when
// it declares the role.
var TriggerNames = schedule.Names{
	Launchd: "com.github.mlorentedev.dotf.self-deploy",
	Systemd: "dotfiles-selfupdate",
	Task:    "DotfilesSelfUpdate",
}

// Trigger is the job that runs `dotf update` once a day on goos for the user
// whose home is home. It names the installed dotf, the pinned release, never
// the running binary: a `go run` build lives in a temp dir that outlives
// nothing. Paths are spelled for goos, whatever OS renders them.
//
// getenv is the environment of the run that installs the job, and the job
// carries its XDG homes. A scheduler does not start jobs from the user's
// shell: without XDG_CONFIG_HOME the job would read a different machine.json,
// find no self-deploy role, and remove itself at its first converge; without
// XDG_STATE_HOME it would write its last-run record where doctor does not
// look, and every run would read as none.
func Trigger(goos, home string, getenv func(string) string) schedule.Job {
	job := schedule.Job{
		Names:       TriggerNames,
		Description: "dotfiles self-deploy: daily dotf update (fast-forward the checkout, then converge; ADR-019)",
		Program:     path.Join(home, ".local", "bin", "dotf"),
		Args:        []string{"update"},
		Daily:       &schedule.Clock{Hour: 3},
	}
	switch goos {
	case "windows":
		job.Program = strings.TrimRight(home, `\/`) + `\.local\bin\dotf.exe`
	case "darwin":
		// launchd hands a job /usr/bin:/bin:/usr/sbin:/sbin, which reaches
		// neither dotf nor Homebrew; converge puts the tool dirs on it itself.
		bin := path.Join(home, ".local", "bin")
		job.Env = []schedule.EnvVar{{Name: "PATH", Value: bin + ":/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"}}
		job.Log = path.Join(home, "Library", "Logs", "dotf-self-deploy.log")
	}
	for _, name := range []string{"XDG_CONFIG_HOME", "XDG_STATE_HOME"} {
		if v := getenv(name); v != "" {
			job.Env = append(job.Env, schedule.EnvVar{Name: name, Value: v})
		}
	}
	return job
}
