package env

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
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

// machine.json carries sections other readers own: the identity and pool policy
// (LoadMachinePolicy) and the roles (LoadMachineRoles). A path write that drops
// them leaves a machine unidentified, which denies every pool, and takes away
// the roles it opted into, with nothing reporting either.
func TestSetMachinePathKeepsSectionsItDoesNotOwn(t *testing.T) {
	dir := t.TempDir()
	contract := writeTestContract(t, dir)
	machinePath := filepath.Join(dir, "machine.json")
	before := `{"machine":{"id":"macmini"},"pools":{"deny":["claude"]},"roles":["ssh-target"],"paths":{}}`
	if err := os.WriteFile(machinePath, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := SetMachinePath(contract, machinePath, "VAULT_PATH", "/data/vault"); err != nil {
		t.Fatalf("SetMachinePath: %v", err)
	}

	policy, err := LoadMachinePolicy(machinePath)
	if err != nil {
		t.Fatal(err)
	}
	if policy.ID != "macmini" || !policy.Denies("claude") {
		t.Errorf("policy after a path write = %+v, want id macmini denying claude", policy)
	}
	roles, err := LoadMachineRoles(machinePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 1 || roles[0] != RoleSSHTarget {
		t.Errorf("roles after a path write = %v, want [%s]", roles, RoleSSHTarget)
	}
}

// Two dotf processes can edit machine.json at once (`dotf env set` and `dotf
// env role add`, or a converge). Each edit reads, changes one section and
// renames a temp file over the file; without a lock held across all three,
// the later rename drops the earlier edit and both report success (#2363).
func TestEditMachineKeepsEveryConcurrentEdit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dotfiles", "machine.json")
	const writers = 8
	errs := make(chan error, writers)
	for i := 0; i < writers; i++ {
		key := fmt.Sprintf("k%d", i)
		go func() {
			_, err := editMachine(path, func(doc map[string]json.RawMessage) (bool, error) {
				sec := map[string]bool{}
				if raw, ok := doc["test"]; ok {
					if err := json.Unmarshal(raw, &sec); err != nil {
						return false, err
					}
				}
				time.Sleep(5 * time.Millisecond) // widen the read-to-rename window
				sec[key] = true
				return true, setSection(doc, "test", sec)
			})
			errs <- err
		}()
	}
	for i := 0; i < writers; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct{ Test map[string]bool }
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Test) != writers {
		t.Fatalf("machine.json kept %d of %d concurrent edits: %v", len(doc.Test), writers, doc.Test)
	}
}
