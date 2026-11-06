package converge

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
)

// fakeHKCU is HKCU\Environment: the variables and the Path value, with a
// count of the writes so a test can assert a second run writes nothing.
type fakeHKCU struct {
	vars   map[string]string
	path   string
	writes int
	err    error
}

func (f *fakeHKCU) Get(name string) (string, bool, error) {
	v, ok := f.vars[name]
	return v, ok, f.err
}

func (f *fakeHKCU) Set(name, value string) error {
	f.vars[name] = value
	f.writes++
	return nil
}

func (f *fakeHKCU) Delete(name string) error {
	delete(f.vars, name)
	f.writes++
	return nil
}

func (f *fakeHKCU) ReadPath() (string, error) { return f.path, f.err }

func (f *fakeHKCU) WritePath(v string) error {
	f.path = v
	f.writes++
	return nil
}

func windowsPersist(h *fakeHKCU, bins ...string) envPersist {
	return envPersist{
		userEnv:  func() (env.UserEnvStore, error) { return h, nil },
		userPath: func() (env.UserPathStore, error) { return h, nil },
		expand:   func(s string) string { return s },
		bins:     bins,
	}
}

func windowsPersistEnv(t *testing.T) Env {
	t.Helper()
	repo := t.TempDir()
	contract := `{"env_vars":[{"name":"VAULT_PATH","default":{"windows":"$HOME/Projects/knowledge"}}]}`
	if err := os.WriteFile(filepath.Join(repo, "env-contract.json"), []byte(contract), 0o600); err != nil {
		t.Fatal(err)
	}
	return Env{RepoRoot: repo, Home: t.TempDir(), GOOS: "windows"}
}

// The blocker-2 criterion (PLAT-001c, measured on an Arm64 guest 2026-11-05):
// a process with no profile gets mise's shims and ~/.local/bin from the user
// Path, ahead of the user's own entries, and the contract variables from the
// user scope. A plan writes nothing, an apply passes its probe, and a second
// run changes nothing.
func TestEnvPersist_WindowsPutsTheToolDirsOnTheUserPath(t *testing.T) {
	const shims, bin = `C:\u\AppData\Local\mise\shims`, `C:\u\.local\bin`
	h := &fakeHKCU{vars: map[string]string{}, path: `C:\u\AppData\Local\Microsoft\WindowsApps`}
	r := windowsPersist(h, shims, bin)
	e := windowsPersistEnv(t)

	if got := r.Platforms(); strings.Join(got, ",") != "darwin,windows" {
		t.Fatalf("Platforms = %v", got)
	}
	plan, err := r.Reconcile(e, true)
	if err != nil || plan.Changes == 0 || h.writes != 0 {
		t.Fatalf("plan = %+v, err %v, writes %d", plan, err, h.writes)
	}
	if err := r.Probe(e); err == nil {
		t.Fatal("the probe must fail before the apply")
	}
	if _, err := r.Reconcile(e, false); err != nil {
		t.Fatal(err)
	}
	if want := shims + ";" + bin + `;C:\u\AppData\Local\Microsoft\WindowsApps`; h.path != want {
		t.Fatalf("Path = %q, want %q", h.path, want)
	}
	if h.vars["VAULT_PATH"] == "" || h.vars[env.ManagedMarker] != "VAULT_PATH" || h.vars[env.ManagedPathMarker] != shims+";"+bin {
		t.Fatalf("user scope = %v", h.vars)
	}
	if err := r.Probe(e); err != nil {
		t.Fatalf("probe after apply: %v", err)
	}
	writes := h.writes
	again, err := r.Reconcile(e, false)
	if err != nil || again.Changes != 0 || h.writes != writes {
		t.Fatalf("second run = %+v, err %v, writes %d -> %d", again, err, writes, h.writes)
	}
}

func TestEnvPersist_WindowsStoreErrorFailsTheStep(t *testing.T) {
	h := &fakeHKCU{vars: map[string]string{}, err: errors.New("access denied")}
	if _, err := windowsPersist(h, `C:\bin`).Reconcile(windowsPersistEnv(t), false); err == nil || h.writes != 0 {
		t.Fatalf("err %v, writes %d", err, h.writes)
	}
}

func TestEnvPersist_WindowsSkippedWithoutAStore(t *testing.T) {
	res, err := envPersist{}.Reconcile(windowsPersistEnv(t), false)
	if err != nil || res.Skip == "" {
		t.Fatalf("no store must skip, got %+v, %v", res, err)
	}
}
