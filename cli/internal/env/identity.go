package env

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// identityLabel is what a machine id may be: a short label, free of whitespace,
// quotes and control characters. The id is a free label (ADR-032 §8): nothing
// is keyed on its value, only on whether one is declared, so the check keeps it
// readable in a log line and in a JSON file rather than policing a namespace.
var identityLabel = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// SetMachineIdentity declares who this machine is in machine.json and reports
// whether the file changed. It is the write side of LoadMachinePolicy.
//
// Declaring an identity is deliberate and never derived: a setup that guessed
// one from the hostname would defeat the denial an undeclared machine gets
// (ADR-032 §8), so the id comes from the caller. Re-declaring the same id is a
// no-op. A different id is refused unless replace is set: provisioning re-runs
// must never overwrite what an owner declared, and a rename is a decision.
// Every other section and key of the file is kept (lesson 402).
func SetMachineIdentity(path, id string, replace bool) (bool, error) {
	if !identityLabel.MatchString(id) {
		return false, fmt.Errorf("machine id %q is not a label: use 1 to 64 letters, digits, '.', '_' or '-'", id)
	}
	return editMachine(path, func(doc map[string]json.RawMessage) (bool, error) {
		machine := map[string]json.RawMessage{}
		if raw, ok := doc["machine"]; ok {
			if err := json.Unmarshal(raw, &machine); err != nil {
				return false, fmt.Errorf("parse machine.json machine block (%s): %w", path, err)
			}
			if machine == nil { // the block holds `null`
				machine = map[string]json.RawMessage{}
			}
		}
		var cur string
		if raw, ok := machine["id"]; ok {
			if err := json.Unmarshal(raw, &cur); err != nil {
				return false, fmt.Errorf("parse machine.json machine.id (%s): %w", path, err)
			}
		}
		switch {
		case cur == id:
			return false, nil
		case cur != "" && !replace:
			return false, fmt.Errorf("this machine already declares the id %q (%s)\n"+
				"a different id is a rename, so it is refused rather than overwritten; "+
				"re-run with --replace to rename it", cur, path)
		}
		raw, err := json.Marshal(id)
		if err != nil {
			return false, fmt.Errorf("marshal machine id: %w", err)
		}
		machine["id"] = raw
		return true, setSection(doc, "machine", machine)
	})
}
