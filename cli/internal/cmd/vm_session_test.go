package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/vm"
)

// noLimactl points the session verbs at a store whose limactl fails the test
// when it is called, and at an empty state directory: a refusal that regresses
// fails here and never boots, deletes or lists a real guest (AC9, lesson 489).
func noLimactl(t *testing.T) {
	t.Helper()
	noGuest(t)
	dir := filepath.Join(t.TempDir(), "sessions")
	old := linuxSessions
	t.Cleanup(func() { linuxSessions = old })
	linuxSessions = func() (vm.LinuxSessions, error) {
		return vm.LinuxSessions{
			Dir: dir,
			L: func(_ context.Context, _ io.Reader, _, _ io.Writer, args ...string) error {
				t.Errorf("a refusal reached the hypervisor: limactl %s", strings.Join(args, " "))
				return errors.New("must not run")
			},
			Now: time.Now,
		}, nil
	}
}

func runVerb(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	c := newVMLinuxCmd()
	c.SetArgs(args)
	var out, errb bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&errb)
	err = c.Execute()
	return out.String(), errb.String(), err
}

// AC9: every refusal of the session verbs, with the exit code ADR-055 decision
// 8 gives it, and the record on stdout under --json.
func TestVMVerbRefusals(t *testing.T) {
	for _, c := range []struct {
		name       string
		args       []string
		wantCode   int
		wantStatus string // "" when the refusal is a usage error with no record
	}{
		{"up with a bad name", []string{"up", "--session", "Bad Name"}, 2, ""},
		{"down with no name", []string{"down", "--json"}, 2, ""},
		{"down with a bad name", []string{"down", "--session", "../x", "--json"}, 2, ""},
		{"down on an unknown session", []string{"down", "--session", "ghost", "--json"}, 3, "no-session"},
		{"exec with no name", []string{"exec", "--json", "--", "true"}, 2, ""},
		{"exec with a bad name", []string{"exec", "--session", "../x", "--", "true"}, 2, ""},
		{"exec with nothing to run", []string{"exec", "--session", "ghost"}, 2, ""},
		{"exec with a program before --", []string{"exec", "--session", "ghost", "true"}, 2, ""},
		{"exec with a program and a script", []string{"exec", "--session", "ghost", "--script", "vm_exec.go", "--", "true"}, 2, ""},
		{"exec with a script that is not there", []string{"exec", "--session", "ghost", "--script", "no-such-file.sh"}, 2, ""},
		{"exec with a negative timeout", []string{"exec", "--session", "ghost", "--timeout", "-1s", "--", "true"}, 2, ""},
		{"exec with an unknown secret", []string{"exec", "--session", "ghost", "--secret", "no-such-secret", "--", "true"}, 2, ""},
		{"exec on an unknown session", []string{"exec", "--session", "ghost", "--json", "--", "true"}, 3, "no-session"},
	} {
		t.Run(c.name, func(t *testing.T) {
			noLimactl(t)
			stdout, _, err := runVerb(t, c.args...)
			if ExitCode(err) != c.wantCode {
				t.Fatalf("exit = %d (%v), want %d", ExitCode(err), err, c.wantCode)
			}
			if c.wantStatus == "" {
				if stdout != "" {
					t.Errorf("stdout = %q, want empty for a usage error", stdout)
				}
				return
			}
			var rec vm.Record
			if err := json.Unmarshal([]byte(stdout), &rec); err != nil || string(rec.Status) != c.wantStatus || rec.Version != 1 {
				t.Errorf("stdout = %q (%v), want one %s record", stdout, err, c.wantStatus)
			}
		})
	}
}

// With --json stdout is exactly the record; an empty list is [] (AC5).
func TestVMStatusWithNoSessionsIsAnEmptyArray(t *testing.T) {
	noGuest(t)
	dir := filepath.Join(t.TempDir(), "sessions")
	old := linuxSessions
	t.Cleanup(func() { linuxSessions = old })
	linuxSessions = func() (vm.LinuxSessions, error) {
		return vm.LinuxSessions{Dir: dir, Now: time.Now, L: func(_ context.Context, _ io.Reader, _, _ io.Writer, _ ...string) error { return nil }}, nil
	}
	stdout, _, err := runVerb(t, "status", "--json")
	if err != nil || stdout != "[]\n" {
		t.Errorf("status --json = %q, %v; want [] and exit 0", stdout, err)
	}
}

// Progress is on stderr and the session's name alone is on stdout (AC5).
func TestVMUpWritesProgressToStderrAndTheNameToStdout(t *testing.T) {
	noGuest(t)
	dir := filepath.Join(t.TempDir(), "sessions")
	old := linuxSessions
	t.Cleanup(func() { linuxSessions = old })
	linuxSessions = func() (vm.LinuxSessions, error) {
		return vm.LinuxSessions{Dir: dir, Now: time.Now, L: func(_ context.Context, _ io.Reader, _, _ io.Writer, _ ...string) error { return nil }}, nil
	}
	stdout, stderr, err := runVerb(t, "up", "--session", "proof")
	if err != nil || stdout != "proof\n" || !strings.Contains(stderr, "==> boot dotf-proof") {
		t.Errorf("up: stdout=%q stderr=%q err=%v", stdout, stderr, err)
	}
}
