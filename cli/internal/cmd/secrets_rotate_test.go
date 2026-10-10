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
    bw: { item: dockerhub, field: PAT, folder: Dotfiles/apps }
    expose: { env: DOCKERHUB_TOKEN }
  - id: BITACORA_PAT
    plane: app
    backend: bw
    bw: { item: github-bitacora-pat, field: api-token, folder: Dotfiles/apps }
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
func (f *fakeRW) MoveItem(item, folderID string) error               { return nil }
func (f *fakeRW) RemoveField(item, field string) error               { return nil }
func (f *fakeRW) DeleteItem(item string) error                       { panic("unexpected DeleteItem " + item) }

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

// AC4: an entry declaring `validate:` is probed with the NEW value after the
// read-back, and a probe that refuses it fails the rotation even though the write
// and the fingerprint change both succeeded.
func TestRotate_ProbesTheNewValueAndFailsWhenItDoesNotAuthenticate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		probe   error
		wantErr bool
		wantOut string
	}{
		{"live", nil, false, "verified live github token"},
		{"refused", errors.New("401 Bad credentials"), true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rw := &fakeRW{fields: map[string]string{"github-bitacora-pat/api-token": "old-pat"}}
			out := rotateHarness(t, rw, &fakeSyncer{})
			v := &fakeValidator{err: tc.probe}
			useGHTokenValidator(t, v)

			cmd := newSecretsRotateCmd()
			cmd.SetOut(out)
			cmd.SetErr(out)
			cmd.SetIn(strings.NewReader("new-pat"))
			useTempRegistry(t, rotateRegistry)

			err := cmd.RunE(cmd, []string{"BITACORA_PAT"})
			if len(v.calls) != 1 || v.calls[0] != "new-pat" {
				t.Fatalf("the probe must run once against the new value, got %d call(s)", len(v.calls))
			}
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), "does not authenticate") {
					t.Fatalf("a refused probe must fail the rotation, got %v\n%s", err, out.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("rotate: %v\n%s", err, out.String())
			}
			if !strings.Contains(out.String(), tc.wantOut) {
				t.Errorf("want %q in the output\n%s", tc.wantOut, out.String())
			}
		})
	}
}

// pushCIRegistry: a secret two repos consume in CI, and one no CI consumes.
const pushCIRegistry = `
version: 1
secrets:
  - id: DOCKERHUB_TOKEN
    plane: app
    backend: bw
    bw: { item: dockerhub, field: PAT, folder: Dotfiles/apps }
    expose: { env: DOCKERHUB_TOKEN }
    consumers: [ci:o/a, ci:o/b, local]
  - id: LOCAL_ONLY
    plane: app
    backend: bw
    bw: { item: local-only, field: key, folder: Dotfiles/apps }
    expose: { env: LOCAL_ONLY }
`

// runRotateWith rotates id to "new-value" with the given flags and returns the
// uploads the GitHub seam received.
func runRotateWith(t *testing.T, fields map[string]string, args ...string) (fakeSetter, string, error) {
	t.Helper()
	rw := &fakeRW{fields: fields}
	out := rotateHarness(t, rw, &fakeSyncer{})
	uploads := fakeSetter{}
	useGHSecretSetter(t, uploads)
	useTempRegistry(t, pushCIRegistry)

	cmd := newSecretsRotateCmd()
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetIn(strings.NewReader("new-value"))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return uploads, out.String(), err
}

// --push-ci hands the rotated value to every repo whose CI consumes the secret, so
// a rotation does not leave GitHub Actions on the retired credential.
func TestRotate_PushCIUploadsTheNewValueToEveryCIConsumer(t *testing.T) {
	uploads, out, err := runRotateWith(t, map[string]string{"dockerhub/PAT": "old-value"},
		"DOCKERHUB_TOKEN", "--push-ci")
	if err != nil {
		t.Fatalf("rotate --push-ci: %v\n%s", err, out)
	}
	for _, key := range []string{"o/a|DOCKERHUB_TOKEN", "o/b|DOCKERHUB_TOKEN"} {
		if uploads[key] != "new-value" {
			t.Errorf("%s: want the rotated value uploaded, got %q\n%s", key, uploads[key], out)
		}
	}
	if len(uploads) != 2 {
		t.Errorf("want exactly the two ci consumers, got %v", uploads)
	}
	if strings.Contains(out, "new-value") {
		t.Errorf("rotate --push-ci printed the secret value\n%s", out)
	}
}

// Without the flag nothing leaves the vault: pushing to CI is an explicit act.
func TestRotate_WithoutPushCIUploadsNothing(t *testing.T) {
	uploads, out, err := runRotateWith(t, map[string]string{"dockerhub/PAT": "old-value"}, "DOCKERHUB_TOKEN")
	if err != nil {
		t.Fatalf("rotate: %v\n%s", err, out)
	}
	if len(uploads) != 0 {
		t.Errorf("rotate without --push-ci uploaded %v", uploads)
	}
}

// A secret no CI consumes rotates normally and says there was nothing to push,
// rather than failing a rotation that succeeded.
func TestRotate_PushCIWithNoCIConsumerSaysSo(t *testing.T) {
	uploads, out, err := runRotateWith(t, map[string]string{"local-only/key": "old-value"},
		"LOCAL_ONLY", "--push-ci")
	if err != nil {
		t.Fatalf("rotate --push-ci: %v\n%s", err, out)
	}
	if len(uploads) != 0 || !strings.Contains(out, "no ci: consumer") {
		t.Errorf("want no upload and a note naming the absence, got %v\n%s", uploads, out)
	}
}

// --dry-run names the repos a real run would push to, and pushes nothing.
func TestRotate_DryRunPushCINamesTheReposAndUploadsNothing(t *testing.T) {
	uploads, out, err := runRotateWith(t, map[string]string{"dockerhub/PAT": "old-value"},
		"DOCKERHUB_TOKEN", "--push-ci", "--dry-run")
	if err != nil {
		t.Fatalf("rotate --dry-run --push-ci: %v\n%s", err, out)
	}
	if len(uploads) != 0 {
		t.Errorf("--dry-run uploaded %v", uploads)
	}
	if !strings.Contains(out, "o/a") || !strings.Contains(out, "o/b") {
		t.Errorf("--dry-run must name both repos it would push to\n%s", out)
	}
}

// rotateOn rotates DOCKERHUB_TOKEN to "new-value" against reg, uploading through
// setter, and returns the vault fake so a test can see whether anything was written.
func rotateOn(t *testing.T, reg string, setter secrets.GitHubSecretSetter, args ...string) (*fakeRW, string, error) {
	t.Helper()
	rw := &fakeRW{fields: map[string]string{"dockerhub/PAT": "old-value"}}
	out := rotateHarness(t, rw, &fakeSyncer{})
	useGHSecretSetter(t, setter)
	useTempRegistry(t, reg)

	cmd := newSecretsRotateCmd()
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetIn(strings.NewReader("new-value"))
	cmd.SetArgs(append([]string{"DOCKERHUB_TOKEN"}, args...))
	err := cmd.Execute()
	return rw, out.String(), err
}

// A ci: consumer that cannot be pushed is a registry bug known before any write. It
// fails before the rotation, so the vault and CI are never left on different values.
func TestRotate_PushCIRefusesAnUnpushableCIConsumerBeforeRotating(t *testing.T) {
	cases := map[string]struct{ consumers, env, want string }{
		"malformed slug":  {consumers: `[ci:o/a, "ci:", local]`, env: "DOCKERHUB_TOKEN", want: "invalid ci consumer"},
		"reserved by GHA": {consumers: `[ci:o/a, local]`, env: "GITHUB_PAT", want: "nothing GitHub Actions can store"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			reg := strings.Replace(pushCIRegistry, "[ci:o/a, ci:o/b, local]", tc.consumers, 1)
			reg = strings.Replace(reg, "expose: { env: DOCKERHUB_TOKEN }", "expose: { env: "+tc.env+" }", 1)
			uploads := fakeSetter{}
			rw, out, err := rotateOn(t, reg, uploads, "--push-ci")
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "nothing was rotated") {
				t.Fatalf("want a refusal containing %q before rotating, got %v\n%s", tc.want, err, out)
			}
			if got := rw.fields["dockerhub/PAT"]; got != "old-value" {
				t.Errorf("the vault was written (%q) although the push was known to be impossible", got)
			}
			if len(uploads) != 0 {
				t.Errorf("nothing may be uploaded, got %v", uploads)
			}
		})
	}
}

// failingSetter refuses uploads to one repo and records the rest.
type failingSetter struct {
	fakeSetter
	refuse string
}

func (f failingSetter) SetSecret(repo, name, value string) error {
	if repo == f.refuse {
		return errors.New("HTTP 403")
	}
	return f.fakeSetter.SetSecret(repo, name, value)
}

// One repo refusing the upload must not strand the others on the retired value,
// and the error must name every repo that still holds it.
func TestRotate_PushCIFailureOnOneRepoStillPushesTheOthers(t *testing.T) {
	setter := failingSetter{fakeSetter: fakeSetter{}, refuse: "o/a"}
	_, out, err := rotateOn(t, pushCIRegistry, setter, "--push-ci")
	if err == nil || !strings.Contains(err.Error(), "o/a still hold the old value") {
		t.Fatalf("want an error naming o/a as stale, got %v\n%s", err, out)
	}
	if setter.fakeSetter["o/b|DOCKERHUB_TOKEN"] != "new-value" {
		t.Errorf("o/b was not pushed after o/a failed: %v\n%s", setter.fakeSetter, out)
	}
}
