package env

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeTestContract drops a minimal contract with two declared vars for the
// SetMachinePath validation path.
func writeTestContract(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "env-contract.json")
	body := `{"env_vars":[` +
		`{"name":"DOTFILES_REPO_DIR","default":{"linux":"$HOME/Projects/dotfiles"}},` +
		`{"name":"VAULT_PATH","default":{"linux":"$HOME/Projects/knowledge"}}]}`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// readPaths parses machine.json back into its paths map for assertions.
func readPaths(t *testing.T, path string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read machine.json: %v", err)
	}
	var m machine
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("parse machine.json: %v", err)
	}
	return m.Paths
}

func TestSetMachinePathCreatesFileAndSetsKey(t *testing.T) {
	dir := t.TempDir()
	contract := writeTestContract(t, dir)
	machinePath := filepath.Join(dir, "cfg", "machine.json") // parent dir absent on purpose

	changed, err := SetMachinePath(contract, machinePath, "DOTFILES_REPO_DIR", "/home/me/repo")
	if err != nil {
		t.Fatalf("SetMachinePath: %v", err)
	}
	if !changed {
		t.Error("changed = false, want true (file created)")
	}
	if got := readPaths(t, machinePath)["DOTFILES_REPO_DIR"]; got != "/home/me/repo" {
		t.Errorf("DOTFILES_REPO_DIR = %q, want /home/me/repo", got)
	}
}

func TestSetMachinePathPreservesOtherKeys(t *testing.T) {
	dir := t.TempDir()
	contract := writeTestContract(t, dir)
	machinePath := filepath.Join(dir, "machine.json")
	// Pre-existing override the seed must NOT clobber.
	if err := os.WriteFile(machinePath, []byte(`{"paths":{"VAULT_PATH":"/data/vault"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := SetMachinePath(contract, machinePath, "DOTFILES_REPO_DIR", "/home/me/repo"); err != nil {
		t.Fatalf("SetMachinePath: %v", err)
	}

	got := readPaths(t, machinePath)
	if got["VAULT_PATH"] != "/data/vault" {
		t.Errorf("VAULT_PATH = %q, want preserved /data/vault", got["VAULT_PATH"])
	}
	if got["DOTFILES_REPO_DIR"] != "/home/me/repo" {
		t.Errorf("DOTFILES_REPO_DIR = %q, want /home/me/repo", got["DOTFILES_REPO_DIR"])
	}
}

func TestSetMachinePathIdempotent(t *testing.T) {
	dir := t.TempDir()
	contract := writeTestContract(t, dir)
	machinePath := filepath.Join(dir, "machine.json")

	if _, err := SetMachinePath(contract, machinePath, "DOTFILES_REPO_DIR", "/home/me/repo"); err != nil {
		t.Fatal(err)
	}
	changed, err := SetMachinePath(contract, machinePath, "DOTFILES_REPO_DIR", "/home/me/repo")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("changed = true on identical re-set, want false (idempotent no-op)")
	}
}

func TestSetMachinePathUnknownKeyFails(t *testing.T) {
	dir := t.TempDir()
	contract := writeTestContract(t, dir)
	machinePath := filepath.Join(dir, "machine.json")

	changed, err := SetMachinePath(contract, machinePath, "DOTFILES_REPODIR" /* typo */, "/home/me/repo")
	if err == nil {
		t.Fatal("SetMachinePath with an undeclared key = nil error, want fail-loud")
	}
	if changed {
		t.Error("changed = true on a rejected key, want false")
	}
	if _, statErr := os.Stat(machinePath); statErr == nil {
		t.Error("machine.json was written despite the key being rejected")
	}
}
