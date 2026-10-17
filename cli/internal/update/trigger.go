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
// configHome is the XDG_CONFIG_HOME of the run that installs the job. A
// scheduler does not start jobs from the user's shell, so without it the job
// would read a different machine.json, find no self-deploy role, and remove
// itself at its first converge.
func Trigger(goos, home, configHome string) schedule.Job {
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
	if configHome != "" {
		job.Env = append(job.Env, schedule.EnvVar{Name: "XDG_CONFIG_HOME", Value: configHome})
	}
	return job
}
