package env

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
)

// RoleSSHTarget makes a machine accept SSH from the mesh, and from nothing
// else (#2356). converge's workstation-target step runs only on a machine that
// declares it.
const RoleSSHTarget = "ssh-target"

// knownRoles are the roles a machine may declare. A role changes how a machine
// faces the network, so none is taken by default and none is taken by a typo:
// a name not listed here is an error, never a role nobody reads.
var knownRoles = []string{RoleSSHTarget}

// KnownRoles returns the roles a machine may declare.
func KnownRoles() []string { return slices.Clone(knownRoles) }

func checkRole(role string) error {
	if slices.Contains(knownRoles, role) {
		return nil
	}
	return fmt.Errorf("unknown machine role %q (known: %s)", role, strings.Join(knownRoles, ", "))
}

// LoadMachineRoles reads the roles machine.json declares. An absent file or
// section declares none, which is the state of every machine until its owner
// opts in. A malformed file or an unknown role is an error.
func LoadMachineRoles(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read machine.json (%s): %w", path, err)
	}
	var f struct {
		Roles []string `json:"roles"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse machine.json (%s): %w", path, err)
	}
	for _, r := range f.Roles {
		if err := checkRole(r); err != nil {
			return nil, fmt.Errorf("machine.json (%s): %w", path, err)
		}
	}
	return f.Roles, nil
}

// SetMachineRole adds role to machine.json when on is true, or removes it, and
// reports whether the file changed. Every other section is kept.
func SetMachineRole(path, role string, on bool) (bool, error) {
	if err := checkRole(role); err != nil {
		return false, err
	}
	return editMachine(path, func(doc map[string]json.RawMessage) (bool, error) {
		var roles []string
		if raw, ok := doc["roles"]; ok {
			if err := json.Unmarshal(raw, &roles); err != nil {
				return false, fmt.Errorf("parse machine.json roles (%s): %w", path, err)
			}
		}
		has := slices.Contains(roles, role)
		switch {
		case on == has:
			return false, nil
		case on:
			roles = append(roles, role)
		default:
			roles = slices.DeleteFunc(roles, func(r string) bool { return r == role })
		}
		if len(roles) == 0 {
			delete(doc, "roles")
			return true, nil
		}
		slices.Sort(roles)
		return true, setSection(doc, "roles", roles)
	})
}
