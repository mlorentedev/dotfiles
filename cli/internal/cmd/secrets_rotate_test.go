package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// rotateRegistry: one bw-backed secret that declares a liveness probe, and one
// that does not — the two shapes rotate must treat differently.
const rotateRegistry = `
version: 1
secrets:
  - id: DOCKERHUB_TOKEN
    plane: app
    backend: bw
    bw: { item: dockerhub, field: PAT, folder: apps }
    expose: { env: DOCKERHUB_TOKEN }
  - id: BITACORA_PAT
    plane: app
    backend: bw
    bw: { item: github-bitacora-pat, field: api-token, folder: apps }
    expose: { env: BITACORA_PAT }
    validate: github-token
`

// fakeRW is a bw read+write pair over an in-memory field map, so rotation is
// exercised with no vault, no daemon and no network.
type fakeRW struct {
	fields   map[string]string
	setErr   error
	readErr  error
	setCalls int
}

func (f *fakeRW) Field(item, field string) (string, error) {
	if f.readErr != nil {
		return "", f.readErr
	}
	v, ok := f.fields[item+"/"+field]
	if !ok {
		return "", secrets.ErrBWFieldNotFound
	}
	return v, nil
}

func (f *fakeRW) SetField(item, field, value string) error {
	f.setCalls++
	if f.setErr != nil {
		return f.setErr
	}
	f.fields[item+"/"+field] = value
	return nil
}

func (f *fakeRW) CreateItem(item, field, value, folder string) error { return nil }
func (f *fakeRW) ResolveFolder(name string) (string, error)          { return "", nil }

type fakeSyncer struct {
	calls int
	err   error
}

func (f *fakeSyncer) Sync() error { f.calls++; return f.err }

// rotateHarness wires the package-level seams for one test and restores them.
func rotateHarness(t *testing.T, rw *fakeRW, sync *fakeSyncer) *bytes.Buffer {
	t.Helper()
	origReader, origWriter, origSync, origTerm := bwReader, bwWriter, bwSyncer, stdinIsTerminal
	t.Cleanup(func() { bwReader, bwWriter, bwSyncer, stdinIsTerminal = origReader, origWriter, origSync, origTerm })
	bwReader, bwWriter, bwSyncer = rw, rw, sync
	stdinIsTerminal = func() bool { return false } // read the value from stdin
	return &bytes.Buffer{}
}

// The happy path, and the assertion that distinguishes rotate from set: the daemon
// is synced and the value is re-read THROUGH the read path before success is claimed.
func TestRotate_WritesSyncsAndProvesTheChange(t *testing.T) {
	rw := &fakeRW{fields: map[string]string{"dockerhub/PAT": "old-token-value"}}
	sync := &fakeSyncer{}
	out := rotateHarness(t, rw, sync)

	cmd := newSecretsRotateCmd()
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetIn(strings.NewReader("brand-new-token-value"))
	useTempRegistry(t, rotateRegistry)

	if err := cmd.RunE(cmd, []string{"DOCKERHUB_TOKEN"}); err != nil {
		t.Fatalf("rotate: %v\n%s", err, out.String())
	}
	if sync.calls != 1 {
		t.Errorf("the daemon must be synced exactly once, got %d — without it every read serves the old value", sync.calls)
	}
	if rw.fields["dockerhub/PAT"] != "brand-new-token-value" {
		t.Errorf("the new value was not written")
	}
	s := out.String()
	if !strings.Contains(s, "rotated") || !strings.Contains(s, "->") {
		t.Errorf("output must report the before -> after fingerprint change\n%s", s)
	}
	// The value itself must never appear.
	if strings.Contains(s, "brand-new-token-value") || strings.Contains(s, "old-token-value") {
		t.Errorf("rotate printed a secret value\n%s", s)
	}
}

// Writing the same value back is the typo case. A liveness probe would pass and a
// bare `set` reports "unchanged" as success; for a rotation that is a failure,
// because the credential you meant to retire is still live.
func TestRotate_RefusesANoOp(t *testing.T) {
	rw := &fakeRW{fields: map[string]string{"dockerhub/PAT": "same-value"}}
	sync := &fakeSyncer{}
	out := rotateHarness(t, rw, sync)

	cmd := newSecretsRotateCmd()
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetIn(strings.NewReader("same-value"))
	useTempRegistry(t, rotateRegistry)

	err := cmd.RunE(cmd, []string{"DOCKERHUB_TOKEN"})
	if err == nil {
		t.Fatal("rotating to the identical value must fail — it is a typo, not a rotation")
	}
	if !strings.Contains(err.Error(), "not a rotation") {
		t.Errorf("the error must say why, got: %v", err)
	}
	if rw.setCalls != 0 {
		t.Errorf("nothing must be written on a no-op, got %d write(s)", rw.setCalls)
	}
}

// The case that motivated the fingerprint: the write succeeds but the read path
// still returns the old value (a stale cache, a write that landed elsewhere).
// A probe against the OLD credential would pass; the fingerprint catches it.
func TestRotate_FailsWhenTheReadPathStillServesTheOldValue(t *testing.T) {
	rw := &fakeRW{fields: map[string]string{"dockerhub/PAT": "old-token-value"}}
	// SetField silently does not take effect on the read path.
	rw.setErr = nil
	sync := &fakeSyncer{}
	out := rotateHarness(t, rw, sync)

	// Freeze the map after the write so the read-back returns the old value.
	origSet := rw.SetField
	_ = origSet
	stubborn := &stubbornRW{fakeRW: rw}
	bwReader, bwWriter = stubborn, stubborn

	cmd := newSecretsRotateCmd()
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetIn(strings.NewReader("brand-new-token-value"))
	useTempRegistry(t, rotateRegistry)

	err := cmd.RunE(cmd, []string{"DOCKERHUB_TOKEN"})
	if err == nil {
		t.Fatal("a write that does not reach the read path must fail the rotation")
	}
	if !strings.Contains(err.Error(), "still the old one") {
		t.Errorf("the error must name the stale-read case, got: %v", err)
	}
}

// stubbornRW accepts writes and never reflects them — the stale-read path.
type stubbornRW struct{ *fakeRW }

func (s *stubbornRW) SetField(item, field, value string) error { s.setCalls++; return nil }

// rotate never creates. Provisioning and replacing are different acts, and
// conflating them is how a locked vault becomes a duplicate item.
func TestRotate_RefusesToCreate(t *testing.T) {
	rw := &fakeRW{fields: map[string]string{}}
	sync := &fakeSyncer{}
	out := rotateHarness(t, rw, sync)

	cmd := newSecretsRotateCmd()
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetIn(strings.NewReader("whatever"))
	useTempRegistry(t, rotateRegistry)

	err := cmd.RunE(cmd, []string{"DOCKERHUB_TOKEN"})
	if err == nil {
		t.Fatal("rotating an absent field must fail, not create it")
	}
	if !strings.Contains(err.Error(), "never creates") {
		t.Errorf("the error must point at `set` for provisioning, got: %v", err)
	}
	if rw.setCalls != 0 {
		t.Errorf("nothing must be written, got %d write(s)", rw.setCalls)
	}
}

// --dry-run reports the current fingerprint and the probe that would run, and
// touches nothing — including stdin, so it never consumes a piped secret.
func TestRotate_DryRunWritesNothing(t *testing.T) {
	rw := &fakeRW{fields: map[string]string{"github-bitacora-pat/api-token": "old"}}
	sync := &fakeSyncer{}
	out := rotateHarness(t, rw, sync)

	cmd := newSecretsRotateCmd()
	cmd.SetOut(out)
	cmd.SetErr(out)
	useTempRegistry(t, rotateRegistry)
	if err := cmd.Flags().Set("dry-run", "true"); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	if err := cmd.RunE(cmd, []string{"BITACORA_PAT"}); err != nil {
		t.Fatalf("dry run: %v\n%s", err, out.String())
	}
	if rw.setCalls != 0 || sync.calls != 0 {
		t.Errorf("dry run must not write (%d) or sync (%d)", rw.setCalls, sync.calls)
	}
	s := out.String()
	for _, want := range []string{"would rotate", "fingerprint", "github-token"} {
		if !strings.Contains(s, want) {
			t.Errorf("dry run must report %q\n%s", want, s)
		}
	}
}

// A sync failure is a warning, not a failure: the write DID happen, and reporting
// it as a failed rotation would send the operator to re-write a value that is
// already stored.
func TestRotate_SyncFailureWarnsButDoesNotFail(t *testing.T) {
	rw := &fakeRW{fields: map[string]string{"dockerhub/PAT": "old-token-value"}}
	sync := &fakeSyncer{err: errors.New("daemon unreachable")}
	out := rotateHarness(t, rw, sync)

	cmd := newSecretsRotateCmd()
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetIn(strings.NewReader("brand-new-token-value"))
	useTempRegistry(t, rotateRegistry)

	if err := cmd.RunE(cmd, []string{"DOCKERHUB_TOKEN"}); err != nil {
		t.Fatalf("a sync failure must not fail a completed write: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "WARNING") {
		t.Errorf("the sync failure must be surfaced\n%s", out.String())
	}
}
