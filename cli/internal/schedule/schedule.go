// Package schedule renders a job a machine runs on a timetable into the native
// scheduler of each OS, and installs, inspects and removes it there: a launchd
// agent on darwin, a systemd user timer on linux, a Scheduled Task on windows
// (OPS-053). One Job declares the command once; each backend owns only how its
// scheduler spells it. It is the seed of the scheduling table of #2013 D5.
//
// The package imports no other dotf package, so env can render its login agent
// through it without an import cycle. Every scheduler CLI is reached through an
// injected Runner, so the backends are tested on any OS.
package schedule

import (
	"errors"
	"fmt"
	"os/exec"
)

// Runner runs one scheduler CLI (launchctl, systemctl --user, schtasks) with
// args and returns its stdout. A non-nil error means the command failed.
type Runner func(args ...string) (string, error)

// Exec is the Runner of the real name, with prefix before every call's args
// (`systemctl --user`). stderr is left out: callers read stdout as the answer.
func Exec(name string, prefix ...string) Runner {
	return func(args ...string) (string, error) {
		out, err := exec.Command(name, append(append([]string{}, prefix...), args...)...).Output() //nolint:gosec // a fixed scheduler CLI, args built by dotf
		return string(out), err
	}
}

// Names are what each scheduler calls the job. They differ because two of them
// predate this package: a trigger a setup script installed keeps its name, so
// adopting it rewrites it in place instead of leaving a twin.
type Names struct {
	Launchd string // the agent's label, also its plist's base name
	Systemd string // the unit stem: <stem>.service and <stem>.timer
	Task    string // the Scheduled Task's name
}

// Job is one command run on a timetable.
type Job struct {
	Names       Names
	Description string
	Program     string   // absolute path, in the target OS's spelling
	Args        []string // the program's arguments
	// Daily runs the job once a day at this local time; nil runs it on no
	// timetable (a login agent).
	Daily *Clock
	// RunAtLoad runs the job when launchd loads it, which is every login.
	// launchd only: a timer job leaves it false, or loading it would run it.
	RunAtLoad bool
	// Env is set in the job's environment: launchd's EnvironmentVariables
	// (the job otherwise gets a bare PATH) and systemd's Environment= lines.
	// The Task Scheduler has no per-task environment, so a task gets the
	// user's persistent one and Env is not rendered there (#2696).
	Env []EnvVar
	// Log is where launchd sends the job's stdout and stderr. launchd only:
	// systemd writes the journal, and the Task Scheduler keeps the result.
	Log string
}

// Clock is a local time of day.
type Clock struct{ Hour, Minute int }

// EnvVar is one variable of a job's environment, kept in order so the
// rendered definition is stable.
type EnvVar struct{ Name, Value string }

// State is what a backend found for a job.
type State struct {
	// Present: a trigger exists under the job's name.
	Present bool
	// Managed: it carries the marker this package writes, so dotf may replace
	// or remove it. A trigger without one was installed by something else (a
	// setup script before OPS-053) and is never removed by a caller that did
	// not opt in.
	Managed bool
	// Pending lists what Install would do; empty means the trigger is current.
	Pending []string
	// Where names the trigger for a report: a path, a unit or a task name.
	Where string
}

// Current reports whether the installed trigger is the rendered job, active.
func (s State) Current() bool { return s.Present && len(s.Pending) == 0 }

// Backend is one OS's scheduler.
type Backend interface {
	// Inspect reads the installed trigger without changing anything.
	Inspect(job Job) (State, error)
	// Install makes the trigger the rendered job and active. It reads the
	// installed state itself, so it changes only what differs.
	Install(job Job) error
	// Remove deletes the trigger and stops it.
	Remove(job Job) error
}

// ErrUnsupported is returned by For for an OS with no backend.
var ErrUnsupported = errors.New("no scheduler backend for this OS")

// Runners carries the scheduler CLIs and what each backend needs to address
// its scheduler. A backend whose runner is nil is not available.
type Runners struct {
	Launchctl Runner
	UID       int    // names the gui/<UID> domain the launchd agent loads into
	AgentsDir string // ~/Library/LaunchAgents

	Systemctl Runner // runs `systemctl --user <args>`
	UnitDir   string // the systemd user unit dir

	Schtasks Runner
	TaskUser string      // the account the task runs as
	Elevated func() bool // whether the caller can register an S4U task
	TempDir  string      // where the task XML is staged for schtasks
}

// For returns goos's backend, or ErrUnsupported when the OS has none or its
// runner is not wired.
func For(goos string, r Runners) (Backend, error) {
	switch {
	case goos == "darwin" && r.Launchctl != nil:
		return Launchd{Run: r.Launchctl, UID: r.UID, Dir: r.AgentsDir}, nil
	case goos == "linux" && r.Systemctl != nil:
		return Systemd{Run: r.Systemctl, Dir: r.UnitDir}, nil
	case goos == "windows" && r.Schtasks != nil:
		return Schtasks{Run: r.Schtasks, User: r.TaskUser, Elevated: r.Elevated, TempDir: r.TempDir}, nil
	}
	return nil, fmt.Errorf("%s: %w", goos, ErrUnsupported)
}
