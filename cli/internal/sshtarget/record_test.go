package sshtarget

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

var recordTime = time.Date(2026, 10, 17, 19, 0, 0, 0, time.UTC)

// recorded is a machine whose target keeps its record under the temp root.
func recorded(t *testing.T) (*machine, Target) {
	t.Helper()
	m := newMachine(t)
	m.paths.Record = filepath.Join(filepath.Dir(m.paths.SSHDConfig), "../../var/lib/dotf/workstation-target.json")
	tg := m.target()
	tg.Dotf = "0.75.0"
	tg.Now = func() time.Time { return recordTime }
	return m, tg
}

func loadRecord(t *testing.T, path string) Record {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var r Record
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatalf("record %s: %v\n%s", path, err, data)
	}
	return r
}

func applyPlan(t *testing.T, tg Target) Plan {
	t.Helper()
	p, err := tg.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if err := tg.Apply(p); err != nil {
		t.Fatal(err)
	}
	return p
}

// A first apply on a fresh machine records each resource as the step's own:
// the retirement may then remove exactly these.
func TestRecordWritesWhatTheFirstApplyCreated(t *testing.T) {
	m, tg := recorded(t)
	m.write(m.paths.HostKey, hostKey+"\n")
	p := applyPlan(t, tg)
	if p.Pending[len(p.Pending)-1] != resRecord {
		t.Fatalf("the record must be the last resource applied: %q", p.Pending)
	}
	r := loadRecord(t, m.paths.Record)
	if r.Version != RecordVersion || r.Dotf != "0.75.0" || !r.Written.Equal(recordTime) {
		t.Errorf("header = %d %q %v", r.Version, r.Dotf, r.Written)
	}
	for name, got := range map[string]*Prior{"key": r.Key, "drop-in": r.DropIn, "listener": r.Listener} {
		want := ""
		if name == "key" {
			want = testKey // the line it wrote, so a rotated repo key cannot hide it (#2799)
		}
		if got == nil || got.State != PriorCreated || got.Content != want {
			t.Errorf("%s = %+v, want created with %q", name, got, want)
		}
	}
	m.forbidRoot = true
	if p, err := tg.Inspect(); err != nil || !p.Converged() {
		t.Errorf("after the apply: %v pending %q, want converged", err, p.Pending)
	}
}

// A drop-in that was there with other content is kept in the record, so the
// retirement can put it back byte for byte.
func TestRecordKeepsTheDropInItReplaced(t *testing.T) {
	m, tg := recorded(t)
	m.write(m.paths.HostKey, hostKey+"\n")
	m.write(m.paths.DropIn, "PermitRootLogin no\n")
	applyPlan(t, tg)
	r := loadRecord(t, m.paths.Record)
	if r.DropIn == nil || r.DropIn.State != PriorReplaced || r.DropIn.Content != "PermitRootLogin no\n" {
		t.Errorf("drop-in = %+v, want replaced with the old content", r.DropIn)
	}
}

// A drop-in carrying the step's marker is the step's own, whichever build
// rendered it: recorded as created, never as content to restore.
func TestRecordTakesADropInWithTheMarkerAsTheStepsOwn(t *testing.T) {
	for name, prev := range map[string]string{
		"an earlier render": DropInMarker + " (#2356).\nPasswordAuthentication no\n",
		"this render":       "",
	} {
		t.Run(name, func(t *testing.T) {
			m, tg := recorded(t)
			m.write(m.paths.HostKey, hostKey+"\n")
			tg.Template = []byte(DropInMarker + " (#2356).\n" + template)
			if prev == "" {
				prev = string(tg.DropIn())
			}
			m.write(m.paths.DropIn, prev)
			applyPlan(t, tg)
			if r := loadRecord(t, m.paths.Record); r.DropIn == nil || r.DropIn.State != PriorCreated || r.DropIn.Content != "" {
				t.Errorf("drop-in = %+v, want created", r.DropIn)
			}
		})
	}
}

// The marker is how the record knows the step's file: the template in the
// checkout must keep opening with it.
func TestTheDropInTemplateOpensWithTheMarker(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", TemplateFile))
	if err != nil {
		t.Fatal(err)
	}
	if !ownDropIn(data) {
		t.Errorf("%s does not open with %q:\n%s", TemplateFile, DropInMarker, data)
	}
}

// A machine converged before the record existed (the macmini and msi under
// 0.74.0) gets one: the only change is the record, and what nothing can tell
// any more is written as unknown, never guessed.
func TestRecordMarksWhatAConvergedMachineAlreadyHadUnknown(t *testing.T) {
	m, tg := recorded(t)
	m.write(m.paths.HostKey, hostKey+"\n")
	record := m.paths.Record
	tg.Paths.Record = ""
	applyPlan(t, tg)
	tg.Paths.Record = record

	p, err := tg.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(p.Pending, []string{resRecord}) {
		t.Fatalf("pending = %q, want only the record", p.Pending)
	}
	m.rootCalls = nil
	if err := tg.Apply(p); err != nil {
		t.Fatal(err)
	}
	for _, c := range m.rootCalls {
		if !strings.HasPrefix(c, "install ") {
			t.Errorf("recording ran %q as root; it may only write the record", c)
		}
	}
	r := loadRecord(t, record)
	for name, got := range map[string]*Prior{"key": r.Key, "drop-in": r.DropIn, "listener": r.Listener} {
		if got == nil || got.State != PriorUnknown {
			t.Errorf("%s = %+v, want unknown", name, got)
		}
	}
}

// An entry is written once: a later apply that changes the resource again
// leaves what the first one found.
func TestRecordEntriesAreWrittenOnce(t *testing.T) {
	m, tg := recorded(t)
	m.write(m.paths.HostKey, hostKey+"\n")
	applyPlan(t, tg)
	m.write(m.paths.DropIn, "edited by hand\n")
	tg.Now = func() time.Time { return recordTime.Add(time.Hour) }
	p := applyPlan(t, tg)
	if slices.Contains(p.Pending, resRecord) {
		t.Errorf("pending = %q: a complete record is not rewritten", p.Pending)
	}
	if r := loadRecord(t, m.paths.Record); r.DropIn.State != PriorCreated || !r.Written.Equal(recordTime) {
		t.Errorf("record = %+v, want the first apply's", r)
	}
}

// A failed proof undoes the record with the rest of the run.
func TestRecordIsUndoneWithAFailedApply(t *testing.T) {
	m, tg := recorded(t)
	m.write(m.paths.HostKey, hostKey+"\n")
	m.accepts = func(string) bool { return false }
	p, err := tg.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if err := tg.Apply(p); err == nil {
		t.Fatal("Apply succeeded with sshd refusing the mesh")
	}
	wrote := slices.ContainsFunc(m.rootCalls, func(c string) bool {
		return strings.HasPrefix(c, "install -m") && strings.HasSuffix(c, " "+m.paths.Record)
	})
	if !wrote {
		t.Fatalf("the apply never wrote the record, so its undo is untested: %q", m.rootCalls)
	}
	if _, err := os.Stat(m.paths.Record); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the record survived the undo: %v", err)
	}
}

// An entry added later keeps the stamp of the build that first wrote the
// record; only the time moves.
func TestRecordKeepsTheBuildThatFirstWroteIt(t *testing.T) {
	m, tg := recorded(t)
	m.write(m.paths.HostKey, hostKey+"\n")
	m.write(m.paths.Record, `{"version": 1, "dotf": "0.74.9", "written": "2026-10-01T00:00:00Z", "authorized_key": {"state": "created"}, "sshd_drop_in": {"state": "created"}}`)
	applyPlan(t, tg)
	r := loadRecord(t, m.paths.Record)
	if r.Dotf != "0.74.9" || !r.Written.Equal(recordTime) || r.Listener == nil {
		t.Errorf("record = %+v, want 0.74.9 kept, the time moved and the listener added", r)
	}
}

// A record this build cannot read is refused, never overwritten: it may be the
// only account of what the machine had before the role.
func TestInspectRefusesARecordItCannotRead(t *testing.T) {
	for name, data := range map[string]string{
		"malformed": "{",
		"newer":     `{"version": 99}`,
	} {
		t.Run(name, func(t *testing.T) {
			m, tg := recorded(t)
			m.write(m.paths.Record, data)
			if _, err := tg.Inspect(); err == nil || !strings.Contains(err.Error(), m.paths.Record) {
				t.Errorf("Inspect = %v, want a refusal naming the record", err)
			}
		})
	}
}

func TestRecordPathPerOS(t *testing.T) {
	for goos, want := range map[string]string{
		"linux":  "/var/lib/dotf/workstation-target.json",
		"darwin": "/var/db/dotf/workstation-target.json",
	} {
		if got := PathsFor(goos, "/home/manu").Record; got != want {
			t.Errorf("%s: Record = %q, want %q", goos, got, want)
		}
	}
	if got := WindowsPaths(`C:\ProgramData`, `C:\Users\manu`).Record; got != `C:\ProgramData\dotf\workstation-target.json` {
		t.Errorf("windows: Record = %q", got)
	}
}
