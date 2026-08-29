package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
)

// One structural variable with a default for both CI legs; the resolved value
// is whatever the OS and the temp HOME make it — the tests never hardcode it.
const sweepContract = `{"env_vars":[
  {"name":"DOTFILES_REPO_DIR","required":false,"default":{"linux":"$HOME/Projects/dotfiles","windows":"$env:USERPROFILE\\Projects\\dotfiles"},"validation":"path_exists"}
]}`

type memUserEnv struct {
	values  map[string]string
	deletes []string
}

func (m *memUserEnv) Get(name string) (string, bool, error) {
	v, ok := m.values[name]
	return v, ok, nil
}

func (m *memUserEnv) Set(name, value string) error {
	m.values[name] = value
	return nil
}

func (m *memUserEnv) Delete(name string) error {
	delete(m.values, name)
	m.deletes = append(m.deletes, name)
	return nil
}

// sweepFixture points the command at a temp checkout holding the contract and
// at a temp HOME, the way stdout_contract_test.go does, so nothing it resolves
// comes from the machine running the test.
func sweepFixture(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "env-contract.json"), []byte(sweepContract), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTFILES_REPO_DIR", dir)
	t.Setenv("DOTFILES_DIR", dir)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
}

func runEnvPersist(t *testing.T, store func() (env.UserEnvStore, error), args ...string) (string, error) {
	t.Helper()
	prev := userEnvStore
	userEnvStore = store
	t.Cleanup(func() { userEnvStore = prev })
	var out bytes.Buffer
	cmd := newEnvPersistCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// AC3 (CLI-065, #1363): --check names a retired name and exits non-zero
// while it is persisted; persist reports the removal; --check is then clean.
func TestEnvPersist_CheckAndSweepOfARetiredName(t *testing.T) {
	sweepFixture(t)
	store := &memUserEnv{values: map[string]string{"FOREIGN": "theirs"}}
	seam := func() (env.UserEnvStore, error) { return store, nil }

	// Run 1 persists the contract and records what it wrote.
	out, err := runEnvPersist(t, seam)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	marker := store.values[env.ManagedMarker]
	if marker != "DOTFILES_REPO_DIR" {
		t.Fatalf("marker after run 1 = %q", marker)
	}

	// An earlier contract had also persisted OLD_NAME; the store still holds it
	// and the marker still lists it — the state a retired variable leaves.
	store.values["OLD_NAME"] = "x"
	store.values[env.ManagedMarker] = marker + ";OLD_NAME"

	out, err = runEnvPersist(t, seam, "--check")
	if err == nil || !strings.Contains(err.Error(), "retired") {
		t.Fatalf("--check must fail naming the retired count, got err=%v\n%s", err, out)
	}
	if !strings.Contains(out, "retired: OLD_NAME") {
		t.Fatalf("--check must name the retired variable:\n%s", out)
	}

	out, err = runEnvPersist(t, seam)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "removed OLD_NAME") || !strings.Contains(out, "1 removed") {
		t.Fatalf("persist must report the removal:\n%s", out)
	}
	if _, ok := store.values["OLD_NAME"]; ok {
		t.Fatal("OLD_NAME still persisted after the sweep")
	}
	if store.values["FOREIGN"] != "theirs" {
		t.Fatal("a name dotf never wrote was touched")
	}
	if store.values[env.ManagedMarker] != "DOTFILES_REPO_DIR" {
		t.Fatalf("marker not rewritten: %q", store.values[env.ManagedMarker])
	}

	out, err = runEnvPersist(t, seam, "--check")
	if err != nil {
		t.Fatalf("--check after the sweep must pass, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "ok:") {
		t.Fatalf("clean --check must say ok:\n%s", out)
	}
}

// AC5: where the OS has no per-user persistent scope the command is a no-op
// that says so, and never reaches the store — the sweep included.
func TestEnvPersist_UnsupportedScopeIsANoOp(t *testing.T) {
	sweepFixture(t)
	out, err := runEnvPersist(t, func() (env.UserEnvStore, error) { return nil, env.ErrUserEnvUnsupported })
	if err != nil {
		t.Fatalf("unsupported scope must not be an error, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "nothing to persist") {
		t.Fatalf("no-op must be reported:\n%s", out)
	}
}
