package vm

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// listLima answers `limactl list --json` from the instances it holds, with
// Lima 2.2.1's real shape (testdata), and records every other call.
type listLima struct {
	instances map[string]string // name -> status as Lima prints it
	calls     []string
	failStart bool
	failDel   bool
}

func (f *listLima) run(_ io.Reader, stdout, _ io.Writer, args ...string) error {
	f.calls = append(f.calls, strings.Join(args, " "))
	switch args[0] {
	case "list":
		// Lima 2.2.1 prints one JSON object per instance, nothing when none.
		b, err := os.ReadFile(filepath.Join("testdata", "limactl-list-running.json"))
		if err != nil {
			return err
		}
		for name, status := range f.instances {
			line := strings.Replace(string(b), `"name":"dotf-measure"`, `"name":"`+name+`"`, 1)
			line = strings.Replace(line, `"status":"Running"`, `"status":"`+status+`"`, 1)
			_, _ = io.WriteString(stdout, strings.TrimSpace(line)+"\n")
		}
	case "start":
		if f.failStart {
			return errors.New("exit status 1")
		}
		for i, a := range args {
			if a == "--name" {
				f.instances[args[i+1]] = "Running"
			}
		}
	case "delete":
		if f.failDel {
			return errors.New("exit status 1")
		}
		delete(f.instances, args[len(args)-1])
	}
	return nil
}

func newSessions(t *testing.T, f *listLima) LinuxSessions {
	t.Helper()
	return LinuxSessions{
		Dir: filepath.Join(t.TempDir(), "sessions"),
		L:   f.run,
		Now: func() time.Time { return time.Unix(1700000000, 0) },
	}
}

func TestValidSessionName(t *testing.T) {
	for _, ok := range []string{"a", "linux-1700000000", "sshd-proof", "x9"} {
		if err := ValidSessionName(ok); err != nil {
			t.Errorf("%q refused: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "-x", "A", "a b", "a/b", "../x", ".", strings.Repeat("a", 40)} {
		if ValidSessionName(bad) == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestUpCreatesTheSessionAndNamesIt(t *testing.T) {
	f := &listLima{instances: map[string]string{}}
	s := newSessions(t, f)
	rec := s.Up("", io.Discard, io.Discard)
	if rec.Status != StatusOK || rec.Session != "linux-1700000000" || rec.OS != "linux" || rec.ProcessCode() != 0 {
		t.Fatalf("Up = %+v, want ok under the default name", rec)
	}
	if f.instances["dotf-linux-1700000000"] != "Running" {
		t.Errorf("instances = %v, want dotf-linux-1700000000 running", f.instances)
	}
	if _, err := os.Stat(rec.StateDir); err != nil || rec.StateDir != filepath.Join(s.Dir, "linux-1700000000") {
		t.Errorf("state_dir = %q (%v), want the session's directory", rec.StateDir, err)
	}
}

func TestUpOnAHeldNameIsBusy(t *testing.T) {
	f := &listLima{instances: map[string]string{}}
	s := newSessions(t, f)
	s.Up("held", io.Discard, io.Discard)
	rec := s.Up("held", io.Discard, io.Discard)
	if rec.Status != StatusBusy || rec.ProcessCode() != 2 || !strings.Contains(rec.Stderr, "down --session held") {
		t.Errorf("second Up = %+v, want busy, exit 2, naming down", rec)
	}
}

// An instance dotf did not create under that name is not a session to take
// over, and not one to delete either.
func TestUpRefusesAnInstanceItDidNotCreate(t *testing.T) {
	f := &listLima{instances: map[string]string{"dotf-mine": "Running"}}
	rec := newSessions(t, f).Up("mine", io.Discard, io.Discard)
	if rec.Status != StatusBusy {
		t.Errorf("Up = %+v, want busy", rec)
	}
	for _, c := range f.calls {
		if strings.HasPrefix(c, "start") || strings.HasPrefix(c, "delete") {
			t.Errorf("a refused Up ran limactl %s", c)
		}
	}
}

func TestUpOnAGoneGuestIsStale(t *testing.T) {
	f := &listLima{instances: map[string]string{}}
	s := newSessions(t, f)
	s.Up("gone", io.Discard, io.Discard)
	delete(f.instances, "dotf-gone")
	rec := s.Up("gone", io.Discard, io.Discard)
	if rec.Status != StatusStale || rec.ProcessCode() != 2 || !strings.Contains(rec.Stderr, "down --session gone") {
		t.Errorf("Up = %+v, want stale, exit 2, naming down", rec)
	}
}

func TestUpThatFailsLeavesNothingBehind(t *testing.T) {
	f := &listLima{instances: map[string]string{}, failStart: true}
	s := newSessions(t, f)
	rec := s.Up("bad", io.Discard, io.Discard)
	if rec.Status != StatusFailed || rec.ProcessCode() != 1 {
		t.Errorf("Up = %+v, want failed", rec)
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "bad")); err == nil {
		t.Error("a failed Up left its state behind")
	}
	if !strings.Contains(strings.Join(f.calls, "\n"), "delete --force dotf-bad") {
		t.Errorf("calls = %q, want the half-started instance deleted", f.calls)
	}
}

func TestDownRemovesTheInstanceAndTheState(t *testing.T) {
	f := &listLima{instances: map[string]string{}}
	s := newSessions(t, f)
	s.Up("x", io.Discard, io.Discard)
	rec := s.Down("x", io.Discard)
	if rec.Status != StatusOK || rec.ProcessCode() != 0 {
		t.Fatalf("Down = %+v", rec)
	}
	if len(f.instances) != 0 {
		t.Errorf("instances = %v, want none", f.instances)
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "x")); err == nil {
		t.Error("state outlived down")
	}
}

func TestDownClearsAStaleSession(t *testing.T) {
	f := &listLima{instances: map[string]string{}}
	s := newSessions(t, f)
	s.Up("x", io.Discard, io.Discard)
	delete(f.instances, "dotf-x")
	if rec := s.Down("x", io.Discard); rec.Status != StatusOK {
		t.Errorf("Down on a stale session = %+v, want ok", rec)
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "x")); err == nil {
		t.Error("stale state survived down")
	}
}

func TestDownOnAnUnknownNameIsNoSession(t *testing.T) {
	f := &listLima{instances: map[string]string{"dotf-theirs": "Running"}}
	rec := newSessions(t, f).Down("theirs", io.Discard)
	if rec.Status != StatusNoSession || rec.ProcessCode() != 3 {
		t.Errorf("Down = %+v, want no-session, exit 3", rec)
	}
	if len(f.instances) != 1 {
		t.Error("down deleted an instance that is not a session")
	}
}

func TestDownThatFailsKeepsTheState(t *testing.T) {
	f := &listLima{instances: map[string]string{}}
	s := newSessions(t, f)
	s.Up("x", io.Discard, io.Discard)
	f.failDel = true
	if rec := s.Down("x", io.Discard); rec.Status != StatusFailed {
		t.Errorf("Down = %+v, want failed", rec)
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "x")); err != nil {
		t.Error("state removed although the instance was not")
	}
}

// status lists what the hypervisor answers: a running instance is ok, a
// record without one is stale, and a Lima instance dotf did not create is not
// listed (ADR-055 decision 2).
func TestListReportsWhatTheHypervisorAnswers(t *testing.T) {
	f := &listLima{instances: map[string]string{"dotf-theirs": "Running"}}
	s := newSessions(t, f)
	s.Up("a", io.Discard, io.Discard)
	s.Up("b", io.Discard, io.Discard)
	delete(f.instances, "dotf-b")
	got, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Session != "a" || got[0].Status != StatusOK || got[1].Session != "b" || got[1].Status != StatusStale {
		t.Errorf("List = %+v, want a ok, b stale", got)
	}
}

func TestListWithNoSessionsIsEmptyNotNil(t *testing.T) {
	got, err := newSessions(t, &listLima{instances: map[string]string{}}).List()
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("List = %#v, %v; want an empty non-nil slice (JSON [])", got, err)
	}
}

// A stopped instance is not there to serve: it is stale, and down clears it.
func TestListTreatsAStoppedInstanceAsStale(t *testing.T) {
	f := &listLima{instances: map[string]string{}}
	s := newSessions(t, f)
	s.Up("a", io.Discard, io.Discard)
	f.instances["dotf-a"] = "Stopped"
	got, _ := s.List()
	if len(got) != 1 || got[0].Status != StatusStale {
		t.Errorf("List = %+v, want a stale", got)
	}
}
