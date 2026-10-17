package schedule

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Marker heads every unit this package writes. A unit without it was written
// by something else, and Managed reads false for it.
const Marker = "# Managed by dotf converge: edits are overwritten."

// SystemdUnits renders job as a oneshot service and the timer that starts it.
// Persistent runs a slot missed while the machine was off at the next boot;
// RandomizedDelaySec spreads machines that share the same remote.
func SystemdUnits(job Job) (service, timer []byte) {
	// systemd expands %-specifiers in every one of these settings, so each %
	// the job carries is doubled wherever it reaches a unit; the argv and the
	// Environment values are also quoted where they need it.
	percent := strings.NewReplacer("%", "%%")
	argv := append([]string{job.Program}, job.Args...)
	for i, a := range argv {
		a = percent.Replace(a)
		if strings.ContainsAny(a, " \t\"\\") {
			a = `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(a) + `"`
		}
		argv[i] = a
	}
	description := percent.Replace(job.Description)
	quote := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "%", "%%")
	var env strings.Builder
	for _, v := range job.Env {
		env.WriteString(`Environment="` + quote.Replace(v.Name+"="+v.Value) + "\"\n")
	}
	service = []byte(Marker + `
[Unit]
Description=` + description + `

[Service]
Type=oneshot
` + env.String() + `ExecStart=` + strings.Join(argv, " ") + `
`)
	onCalendar := "daily"
	if c := job.Daily; c != nil {
		onCalendar = fmt.Sprintf("*-*-* %02d:%02d:00", c.Hour, c.Minute)
	}
	timer = []byte(Marker + `
[Unit]
Description=` + description + `

[Timer]
OnCalendar=` + onCalendar + `
Persistent=true
RandomizedDelaySec=300

[Install]
WantedBy=timers.target
`)
	return service, timer
}

// Systemd is the linux backend: a service and a timer in the user unit dir,
// the timer enabled and started. Run is `systemctl --user`.
type Systemd struct {
	Run Runner
	Dir string // ~/.config/systemd/user
}

type systemdState struct {
	exists, managed, stale, enabled, active bool
}

func (s Systemd) files(job Job) (service, timer string) {
	return filepath.Join(s.Dir, job.Names.Systemd+".service"), filepath.Join(s.Dir, job.Names.Systemd+".timer")
}

func (s Systemd) timerUnit(job Job) string { return job.Names.Systemd + ".timer" }

func (s Systemd) read(job Job) (systemdState, error) {
	var st systemdState
	svcPath, tmrPath := s.files(job)
	svc, tmr := SystemdUnits(job)
	for _, f := range []struct {
		path string
		want []byte
	}{{svcPath, svc}, {tmrPath, tmr}} {
		have, err := os.ReadFile(f.path)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			st.stale = true
			continue
		case err != nil:
			return st, err
		}
		st.exists = true
		st.managed = st.managed || bytes.HasPrefix(have, []byte(Marker+"\n"))
		st.stale = st.stale || !bytes.Equal(have, f.want)
	}
	// `show` answers for a unit the manager does not know (no UnitFileState,
	// ActiveState=inactive), so its error is a manager dotf cannot reach, never
	// an absent timer. is-enabled exits non-zero for both.
	out, err := s.Run("show", "--property=UnitFileState", "--property=ActiveState", s.timerUnit(job))
	if err != nil {
		return st, fmt.Errorf("systemctl --user show %s: %w", s.timerUnit(job), err)
	}
	for _, line := range strings.Split(out, "\n") {
		switch k, v, _ := strings.Cut(strings.TrimSpace(line), "="); k {
		case "UnitFileState":
			st.enabled = v == "enabled"
		case "ActiveState":
			st.active = v == "active"
		}
	}
	return st, nil
}

func (s Systemd) Inspect(job Job) (State, error) {
	st, err := s.read(job)
	if err != nil {
		return State{}, err
	}
	out := State{Present: st.exists || st.enabled, Managed: st.managed, Where: s.timerUnit(job) + " in " + s.Dir}
	if st.stale {
		out.Pending = append(out.Pending, "write "+job.Names.Systemd+".{service,timer}")
	}
	if st.stale || !st.enabled || !st.active {
		out.Pending = append(out.Pending, "enable --now "+s.timerUnit(job))
	}
	return out, nil
}

// Install writes the units, reloads the manager when they changed, and enables
// and starts the timer. `enable --now` is idempotent, so it runs whenever the
// timer is not both enabled and active.
func (s Systemd) Install(job Job) error {
	st, err := s.read(job)
	if err != nil {
		return err
	}
	if st.stale {
		if err := os.MkdirAll(s.Dir, 0o755); err != nil { //nolint:gosec // the systemd user dir is the user's own config
			return err
		}
		svcPath, tmrPath := s.files(job)
		svc, tmr := SystemdUnits(job)
		if err := writeAtomic(svcPath, svc, 0o644); err != nil {
			return err
		}
		if err := writeAtomic(tmrPath, tmr, 0o644); err != nil {
			return err
		}
		if _, err := s.Run("daemon-reload"); err != nil {
			return fmt.Errorf("systemctl --user daemon-reload: %w", err)
		}
	}
	if st.stale || !st.enabled || !st.active {
		if _, err := s.Run("enable", "--now", s.timerUnit(job)); err != nil {
			return fmt.Errorf("systemctl --user enable --now %s: %w", s.timerUnit(job), err)
		}
	}
	return nil
}

// Remove stops and disables the timer, deletes both units and reloads.
func (s Systemd) Remove(job Job) error {
	st, err := s.read(job)
	if err != nil {
		return err
	}
	if st.enabled || st.active {
		if _, err := s.Run("disable", "--now", s.timerUnit(job)); err != nil {
			return fmt.Errorf("systemctl --user disable --now %s: %w", s.timerUnit(job), err)
		}
	}
	svcPath, tmrPath := s.files(job)
	for _, p := range []string{tmrPath, svcPath} {
		if err := removeIfExists(p); err != nil {
			return err
		}
	}
	if _, err := s.Run("daemon-reload"); err != nil {
		return fmt.Errorf("systemctl --user daemon-reload: %w", err)
	}
	return nil
}
