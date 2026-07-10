package env

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SetMachinePath merges one path override into machine.json — the write-side
// counterpart of the read-side ResolvePath / `dotf env path`. It loads the
// existing file (an absent file starts empty), validates key against the
// contract at contractPath (an undeclared key is rejected so a typo cannot
// silently create a dead override no resolver reads), sets paths[key]=value
// while preserving every other override, and writes the file back atomically
// (temp + rename in the same dir), creating the parent dir when absent. Returns
// changed=false when key was already set to value (an idempotent no-op, no
// write), true when the file was (re)written.
func SetMachinePath(contractPath, machinePath, key, value string) (bool, error) {
	c, err := loadContract(contractPath)
	if err != nil {
		return false, err
	}
	if !contractHasVar(c, key) {
		return false, fmt.Errorf("unknown path key %q: not declared in env-contract.json — check the spelling of the contract var", key)
	}
	m, err := loadMachine(machinePath)
	if err != nil {
		return false, err
	}
	if cur, ok := m.Paths[key]; ok && cur == value {
		return false, nil // already set to this value — idempotent
	}
	m.Paths[key] = value
	if err := writeMachine(machinePath, m); err != nil {
		return false, err
	}
	return true, nil
}

// contractHasVar reports whether name is a declared structural var in the
// contract — the allowlist SetMachinePath validates against.
func contractHasVar(c *contract, name string) bool {
	for _, v := range c.EnvVars {
		if v.Name == name {
			return true
		}
	}
	return false
}

// writeMachine serializes m to path atomically: it writes a temp file in the
// same directory (so the rename is atomic on one filesystem) and renames it over
// path, so a concurrent reader never sees a half-written file. json.Marshal
// emits map keys sorted, keeping the file diff-friendly across runs.
func writeMachine(path string, m *machine) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create machine.json dir: %w", err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal machine.json: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, ".machine-*.json")
	if err != nil {
		return fmt.Errorf("create temp machine.json: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // no-op once the rename below succeeds
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp machine.json: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp machine.json: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename machine.json into place: %w", err)
	}
	return nil
}
