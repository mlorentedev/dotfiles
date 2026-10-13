package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMachineRolesAbsentFileHasNone(t *testing.T) {
	roles, err := LoadMachineRoles(filepath.Join(t.TempDir(), "machine.json"))
	if err != nil || len(roles) != 0 {
		t.Fatalf("LoadMachineRoles(absent) = %v, %v; want none and no error", roles, err)
	}
}

func TestLoadMachineRolesRejectsAnUnknownRole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "machine.json")
	if err := os.WriteFile(path, []byte(`{"roles":["ssh-targt"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadMachineRoles(path); err == nil || !strings.Contains(err.Error(), "ssh-targt") {
		t.Fatalf("LoadMachineRoles(typo) error = %v, want one naming the unknown role", err)
	}
}

func TestLoadMachineRolesMalformedFileIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "machine.json")
	if err := os.WriteFile(path, []byte(`{"roles":`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadMachineRoles(path); err == nil {
		t.Fatal("LoadMachineRoles(malformed) = nil error, want one")
	}
}

func TestSetMachineRoleAddsOnceAndKeepsTheRest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg", "machine.json")
	changed, err := SetMachineRole(path, RoleSSHTarget, true)
	if err != nil || !changed {
		t.Fatalf("first add = %v, %v; want changed", changed, err)
	}
	if err := os.WriteFile(path, []byte(`{"machine":{"id":"m1"},"roles":["ssh-target"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err = SetMachineRole(path, RoleSSHTarget, true)
	if err != nil || changed {
		t.Fatalf("second add = %v, %v; want unchanged", changed, err)
	}
	changed, err = SetMachineRole(path, RoleSSHTarget, false)
	if err != nil || !changed {
		t.Fatalf("remove = %v, %v; want changed", changed, err)
	}
	roles, err := LoadMachineRoles(path)
	if err != nil || len(roles) != 0 {
		t.Fatalf("roles after remove = %v, %v; want none", roles, err)
	}
	if p, _ := LoadMachinePolicy(path); p.ID != "m1" {
		t.Errorf("machine id after a role write = %q, want m1", p.ID)
	}
	changed, err = SetMachineRole(path, RoleSSHTarget, false)
	if err != nil || changed {
		t.Fatalf("second remove = %v, %v; want unchanged", changed, err)
	}
}

func TestSetMachineRoleRejectsAnUnknownRole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "machine.json")
	if _, err := SetMachineRole(path, "ssh-targt", true); err == nil {
		t.Fatal("SetMachineRole(typo) = nil error, want one")
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("machine.json was written for a rejected role")
	}
}

func TestSetMachineRoleRemovesAnUnknownRole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "machine.json")
	if err := os.WriteFile(path, []byte(`{"roles":["ssh-targt"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadMachineRoles(path); err == nil || !strings.Contains(err.Error(), "dotf env role rm ssh-targt") {
		t.Fatalf("LoadMachineRoles = %v, want the way out named", err)
	}
	if changed, err := SetMachineRole(path, "ssh-targt", false); err != nil || !changed {
		t.Fatalf("remove the unknown role = %v, %v; want changed", changed, err)
	}
	if roles, err := LoadMachineRoles(path); err != nil || len(roles) != 0 {
		t.Fatalf("roles after the remove = %v, %v; want none", roles, err)
	}
}
