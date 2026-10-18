package sshtarget

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const otherKey = `from="10.0.0.0/8" ssh-ed25519 AAAAOther other@host`

// retiring is a machine the target was applied to: sshd loaded, the drop-in
// and the key in place, and a record whose entries the test sets.
func retiring(t *testing.T, key, dropIn, listener *Prior) *machine {
	t.Helper()
	m := newMachine(t)
	m.paths.Record = filepath.Join(filepath.Dir(m.paths.SSHDConfig), "..", "..", "var", "db", "dotf", "workstation-target.json")
	m.loaded = true
	m.write(m.paths.DropIn, "AllowUsers manu@100.64.0.0/10\n")
	m.write(m.paths.AuthorizedKeys, otherKey+"\n"+testKey+"\n")
	data, err := json.Marshal(Record{Version: RecordVersion, Key: key, DropIn: dropIn, Listener: listener})
	if err != nil {
		t.Fatal(err)
	}
	m.write(m.paths.Record, string(data))
	return m
}

var (
	created = &Prior{State: PriorCreated}
	unknown = &Prior{State: PriorUnknown}
)

func inspectRetire(t *testing.T, m *machine, tg Target) Retirement {
	t.Helper()
	m.forbidRoot = true
	r, err := tg.InspectRetire()
	m.forbidRoot = false
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func gone(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	return errors.Is(err, os.ErrNotExist)
}

// Retiring undoes only what the record says the step created, listener off
// and persisted first, the record last; a second run changes nothing.
func TestRetireUndoesWhatTheStepCreated(t *testing.T) {
	m := retiring(t, created, created, created)
	tg := m.target()
	r := inspectRetire(t, m, tg)
	if got, want := strings.Join(r.Pending, ","), "sshd service,sshd drop-in,authorized key,state record"; got != want {
		t.Fatalf("pending = %s, want %s", got, want)
	}
	if err := tg.Retire(r); err != nil {
		t.Fatal(err)
	}
	off, persisted := m.index("launchctl bootout"), m.index("launchctl disable")
	drop, rec := m.index("rm -f "+m.paths.DropIn), m.index("rm -f "+m.paths.Record)
	if off < 0 || persisted < off || drop < persisted || rec < drop {
		t.Fatalf("want the listener off and persisted, then the drop-in, then the record: %q", m.rootCalls)
	}
	if m.loaded || !gone(t, m.paths.DropIn) || !gone(t, m.paths.Record) {
		t.Errorf("after Retire: sshd loaded %v, drop-in gone %v, record gone %v", m.loaded, gone(t, m.paths.DropIn), gone(t, m.paths.Record))
	}
	if keys, _ := os.ReadFile(m.paths.AuthorizedKeys); string(keys) != otherKey+"\n" {
		t.Errorf("authorized_keys = %q, want only the owner's other line, byte for byte", keys)
	}
	if again := inspectRetire(t, m, tg); !again.Converged() {
		t.Errorf("a second run plans %v", again.Pending)
	}
}

func TestRetireRestoresTheDropInItReplaced(t *testing.T) {
	m := retiring(t, created, &Prior{State: PriorReplaced, Content: "PasswordAuthentication no\n"}, created)
	tg := m.target()
	if err := tg.Retire(inspectRetire(t, m, tg)); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(m.paths.DropIn); string(got) != "PasswordAuthentication no\n" {
		t.Errorf("drop-in = %q, want the one the step replaced", got)
	}
}

// An unknown entry may be the owner's: it is left and named, and the record,
// which can tell nothing more about it, goes.
func TestRetireLeavesWhatItCannotTellIsItsOwn(t *testing.T) {
	m := retiring(t, unknown, unknown, unknown)
	tg := m.target()
	r := inspectRetire(t, m, tg)
	if got := strings.Join(r.Pending, ","); got != "state record" {
		t.Fatalf("pending = %s, want the record alone", got)
	}
	for _, res := range []string{"sshd service", "sshd drop-in", "authorized key"} {
		if !slices.ContainsFunc(r.Notes, func(n string) bool { return strings.HasPrefix(n, res) }) {
			t.Errorf("notes %q do not name %s", r.Notes, res)
		}
	}
	if err := tg.Retire(r); err != nil {
		t.Fatal(err)
	}
	if !m.loaded || gone(t, m.paths.DropIn) || !strings.Contains(fileText(t, m.paths.AuthorizedKeys), testKey) {
		t.Errorf("an unknown entry was undone: %q", m.rootCalls)
	}
}

// Without its drop-in, an sshd the step did not start would run on its stock
// config, passwords included (#2388): the drop-in is held, and the record with
// it, until sshd is off.
func TestRetireHoldsTheDropInWhileAListenerItDidNotStartRuns(t *testing.T) {
	for name, prior := range map[string]*Prior{"created": created, "replaced": {State: PriorReplaced, Content: "X\n"}} {
		m := retiring(t, unknown, prior, unknown)
		tg := m.target()
		r := inspectRetire(t, m, tg)
		if !r.Converged() || !slices.ContainsFunc(r.Notes, func(n string) bool { return strings.Contains(n, "held while sshd listens") }) {
			t.Fatalf("%s: pending %v, notes %q; want nothing pending and the drop-in held", name, r.Pending, r.Notes)
		}
		if err := tg.Retire(r); err != nil || len(m.rootCalls) != 0 {
			t.Fatalf("%s: Retire = %v, ran %q; want nothing done", name, err, m.rootCalls)
		}
		m.loaded = false // the owner turned sshd off
		if got := strings.Join(inspectRetire(t, m, tg).Pending, ","); got != "sshd drop-in,state record" {
			t.Errorf("%s: with sshd off, pending = %s", name, got)
		}
	}
}

// Stopping sshd from a session it carries would cut the session: refused
// before anything changes, and refused when sudo hid the session.
func TestRetireRefusesToStopTheListenerItRunsOver(t *testing.T) {
	for _, tc := range []struct {
		session string
		sudoed  bool
	}{{"100.64.0.9 51234 100.64.0.15 22", false}, {"", true}} {
		m := retiring(t, created, created, created)
		tg := m.target()
		tg.Session, tg.Sudoed = tc.session, tc.sudoed
		if err := tg.Retire(inspectRetire(t, m, tg)); err == nil || !strings.Contains(err.Error(), "at the machine") {
			t.Errorf("session %q, sudo %v: Retire = %v, want a refusal", tc.session, tc.sudoed, err)
		}
		if len(m.rootCalls) != 0 || gone(t, m.paths.Record) {
			t.Errorf("session %q: the refusal ran %q", tc.session, m.rootCalls)
		}
	}
	m := retiring(t, unknown, unknown, unknown)
	tg := m.target()
	tg.Session = "100.64.0.9 51234 100.64.0.15 22"
	if err := tg.Retire(inspectRetire(t, m, tg)); err != nil {
		t.Errorf("with no listener to stop, Retire over ssh = %v, want it to proceed", err)
	}
}

// A run that fails half way keeps the record, so the next converge resumes
// from what is left instead of forgetting it.
func TestRetireKeepsTheRecordWhenAStepFails(t *testing.T) {
	m := retiring(t, created, created, created)
	m.failRoot = "rm -f " + m.paths.DropIn
	tg := m.target()
	if err := tg.Retire(inspectRetire(t, m, tg)); err == nil || !strings.Contains(err.Error(), "resumes") {
		t.Fatalf("Retire = %v, want the failure and that the next run resumes", err)
	}
	if gone(t, m.paths.Record) {
		t.Error("a failed retirement removed the record")
	}
	m.failRoot = ""
	if got := strings.Join(inspectRetire(t, m, tg).Pending, ","); got != "sshd drop-in,authorized key,state record" {
		t.Errorf("the next run plans %s", got)
	}
}

func TestRetireWithoutARecordIsNothing(t *testing.T) {
	m := newMachine(t)
	m.paths.Record = filepath.Join(t.TempDir(), "workstation-target.json")
	r := inspectRetire(t, m, m.target())
	if r.Present || !r.Converged() {
		t.Errorf("present %v, pending %v; a machine with no record has nothing to retire", r.Present, r.Pending)
	}
}

func fileText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
