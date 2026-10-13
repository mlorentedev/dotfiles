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
	return editMachine(machinePath, func(doc map[string]json.RawMessage) (bool, error) {
		paths := map[string]string{}
		if raw, ok := doc["paths"]; ok {
			if err := json.Unmarshal(raw, &paths); err != nil {
				return false, fmt.Errorf("parse machine.json paths (%s): %w", machinePath, err)
			}
			if paths == nil {
				paths = map[string]string{}
			}
		}
		if cur, ok := paths[key]; ok && cur == value {
			return false, nil // already set to this value — idempotent
		}
		paths[key] = value
		return true, setSection(doc, "paths", paths)
	})
}

// editMachine applies edit to machine.json as a document of top-level sections
// and writes the file back only when edit reports a change. The sections edit
// does not touch are kept as they were: machine.json holds sections that other
// readers own (machine, pools, roles), and a writer that round-trips the file
// through its own struct deletes every section that struct lacks (lesson 402).
// An absent file starts as an empty document; a malformed one is an error.
func editMachine(path string, edit func(doc map[string]json.RawMessage) (bool, error)) (bool, error) {
	doc := map[string]json.RawMessage{}
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(raw, &doc); err != nil {
			return false, fmt.Errorf("parse machine.json (%s): %w", path, err)
		}
		if doc == nil { // the file holds `null`
			doc = map[string]json.RawMessage{}
		}
	case !os.IsNotExist(err):
		return false, fmt.Errorf("read machine.json: %w", err)
	}
	changed, err := edit(doc)
	if err != nil || !changed {
		return false, err
	}
	if err := writeMachine(path, doc); err != nil {
		return false, err
	}
	return true, nil
}

// setSection stores v as the section name of doc.
func setSection(doc map[string]json.RawMessage, name string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal machine.json %s: %w", name, err)
	}
	doc[name] = raw
	return nil
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

// writeMachine serializes doc to path atomically: it writes a temp file in the
// same directory (so the rename is atomic on one filesystem) and renames it over
// path, so a concurrent reader never sees a half-written file. json.Marshal
// emits map keys sorted, keeping the file diff-friendly across runs.
func writeMachine(path string, doc map[string]json.RawMessage) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create machine.json dir: %w", err)
	}
	data, err := json.MarshalIndent(doc, "", "  ")
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
