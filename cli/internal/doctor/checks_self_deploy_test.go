package doctor

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
	"time"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/schedule"
	"github.com/mlorentedev/dotfiles/cli/internal/update"
)

// inspectOnly is a scheduler doctor may only read: an install or a removal
// fails the test.
type inspectOnly struct {
	t     *testing.T
	state schedule.State
	err   error
}

func (b inspectOnly) Inspect(schedule.Job) (schedule.State, error) { return b.state, b.err }
func (b inspectOnly) Install(schedule.Job) error {
	b.t.Fatal("doctor installed the trigger; it must only read")
	return nil
}
func (b inspectOnly) Remove(schedule.Job) error {
	b.t.Fatal("doctor removed the trigger; it must only read")
	return nil
}

// AC7 (OPS-053): doctor's Self-deploy section reports the opt-in, the trigger
// the converge step owns, and the last `dotf update`, one row per verdict.
func TestCheckSelfDeploy_ByState(t *testing.T) {
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	where := "launchd agent com.github.mlorentedev.dotf.self-deploy"
	current := schedule.State{Present: true, Managed: true, Where: where}
	run := func(ago time.Duration, status string, err error) *update.Record {
		r := update.NewRecord(now.Add(-ago), update.Outcome{Status: status, Message: status + " message"}, err)
		return &r
	}
	cases := []struct {
		name    string
		goos    string
		optIn   bool
		state   schedule.State
		inspect error
		record  *update.Record
		want    Status
		needle  string
	}{
		{name: "not opted in, no trigger → SKIP naming the opt-in", optIn: false, state: schedule.State{Where: where}, want: StatusSkip, needle: "dotf env role add self-deploy"},
		{name: "not opted in, a trigger dotf did not write → WARN naming the adoption", state: schedule.State{Present: true, Where: where}, want: StatusWarn, needle: "not by dotf"},
		{name: "not opted in, a trigger dotf wrote → WARN naming the removal", state: schedule.State{Present: true, Managed: true, Where: where}, want: StatusWarn, needle: "converge removes it"},
		{name: "opted in, no trigger → WARN naming the converge", optIn: true, state: schedule.State{Where: where, Pending: []string{"write the agent"}}, want: StatusWarn, needle: "dotf converge --only self-deploy"},
		{name: "opted in, an unmarked trigger → WARN naming the converge", optIn: true, state: schedule.State{Present: true, Where: where, Pending: []string{"adopt"}}, want: StatusWarn, needle: "dotf converge --only self-deploy"},
		{name: "scheduler unreadable → WARN", optIn: true, inspect: errors.New("launchctl print: exit 5"), want: StatusWarn, needle: "launchctl print: exit 5"},
		{name: "not opted in, scheduler unreadable → SKIP naming the opt-in", inspect: errors.New("no user bus"), want: StatusSkip, needle: "dotf env role add self-deploy"},
		{name: "opted in, no run recorded → WARN naming the first run", optIn: true, state: current, want: StatusWarn, needle: "no run recorded yet"},
		{name: "last run failed → FAIL with its error", optIn: true, state: current, record: run(time.Hour, "converge-failed", errors.New("converge: exit 1")), want: StatusFail, needle: "converge: exit 1"},
		{name: "last run older than three days → WARN", optIn: true, state: current, record: run(4*24*time.Hour, "updated", nil), want: StatusWarn, needle: "4 days ago"},
		{name: "last run skipped on a dirty checkout → WARN", optIn: true, state: current, record: run(time.Hour, "dirty", nil), want: StatusWarn, needle: "dirty message"},
		{name: "last run offline → PASS", optIn: true, state: current, record: run(time.Hour, "offline", nil), want: StatusPass, needle: "offline message"},
		{name: "last run updated → PASS", optIn: true, state: current, record: run(time.Hour, "updated", nil), want: StatusPass, needle: "updated message"},
		{name: "linux, last run current → PASS", goos: "linux", optIn: true, state: current, record: run(time.Hour, "current", nil), want: StatusPass, needle: "current message"},
		{name: "windows, opted in, no trigger → WARN", goos: "windows", optIn: true, state: schedule.State{Where: `task \DotfilesSelfUpdate`, Pending: []string{"register"}}, want: StatusWarn, needle: "dotf converge --only self-deploy"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			config, state := filepath.Join(home, ".config"), filepath.Join(home, "state")
			t.Setenv("XDG_CONFIG_HOME", config)
			if tc.optIn {
				if _, err := envpkg.SetMachineRole(envpkg.MachinePath(home), envpkg.RoleSelfDeploy, true); err != nil {
					t.Fatal(err)
				}
			}
			if tc.record != nil {
				if err := update.WriteRecord(filepath.Join(state, "dotfiles"), *tc.record); err != nil {
					t.Fatal(err)
				}
			}
			s := newSys(map[string]string{"HOME": home, "XDG_CONFIG_HOME": config, "XDG_STATE_HOME": state}, nil, nil)
			s.GOOS = "darwin"
			if tc.goos != "" {
				s.GOOS = tc.goos
			}
			s.Now = func() time.Time { return now }
			s.Scheduler = func(goos, h string) (schedule.Backend, error) {
				if goos != s.GOOS || h != home {
					t.Fatalf("scheduler for %s %s, want %s %s", goos, h, s.GOOS, home)
				}
				return inspectOnly{t: t, state: tc.state, err: tc.inspect}, nil
			}
			var buf bytes.Buffer
			checkSelfDeploy(s, capture(&buf))
			if got := statusOfLine(buf.String(), tc.needle); got != tc.want {
				t.Fatalf("line mentioning %q: status %q, want %q\n%s", tc.needle, tagOf(got), tagOf(tc.want), buf.String())
			}
		})
	}
}

// Off the three schedulers, or with none wired, there is no section.
func TestCheckSelfDeploy_NoSectionWithoutAScheduler(t *testing.T) {
	s := newSys(map[string]string{"HOME": t.TempDir()}, nil, nil)
	s.GOOS = "darwin"
	var buf bytes.Buffer
	checkSelfDeploy(s, capture(&buf))
	s.GOOS = "freebsd"
	s.Scheduler = func(string, string) (schedule.Backend, error) { return nil, schedule.ErrUnsupported }
	checkSelfDeploy(s, capture(&buf))
	if buf.Len() != 0 {
		t.Errorf("want no section, got:\n%s", buf.String())
	}
}

// A machine.json that does not parse is said, not read as "not opted in".
func TestCheckSelfDeploy_UnreadableRolesWarn(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, ".config")
	t.Setenv("XDG_CONFIG_HOME", config)
	writeFile(t, envpkg.MachinePath(home), "{not json")
	s := newSys(map[string]string{"HOME": home, "XDG_CONFIG_HOME": config}, nil, nil)
	s.GOOS = "darwin"
	s.Scheduler = func(string, string) (schedule.Backend, error) {
		return inspectOnly{t: t, state: schedule.State{Where: "x"}}, nil
	}
	var buf bytes.Buffer
	checkSelfDeploy(s, capture(&buf))
	if got := statusOfLine(buf.String(), "machine.json"); got != StatusWarn {
		t.Fatalf("status %q, want WARN\n%s", tagOf(got), buf.String())
	}
}

// A scheduler that exists but cannot be reached is said once the machine opted
// in; only an OS with none has no section.
func TestCheckSelfDeploy_AnUnreachableSchedulerWarns(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, ".config")
	t.Setenv("XDG_CONFIG_HOME", config)
	if _, err := envpkg.SetMachineRole(envpkg.MachinePath(home), envpkg.RoleSelfDeploy, true); err != nil {
		t.Fatal(err)
	}
	s := newSys(map[string]string{"HOME": home, "XDG_CONFIG_HOME": config}, nil, nil)
	s.GOOS = "linux"
	s.Scheduler = func(string, string) (schedule.Backend, error) { return nil, errors.New("no user bus") }
	var buf bytes.Buffer
	checkSelfDeploy(s, capture(&buf))
	if got := statusOfLine(buf.String(), "no user bus"); got != StatusWarn {
		t.Fatalf("status %q, want WARN\n%s", tagOf(got), buf.String())
	}
}
