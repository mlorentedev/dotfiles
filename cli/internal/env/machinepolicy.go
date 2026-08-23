package env

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// MachinePolicy is what machine.json says about WHO this machine is and which
// pools it forbids. It is the one exception to ADR-032 §7's "machine facts are
// probed, never stored": a probe can answer whether a pool is reachable, and
// cannot answer whether it is permitted.
//
// The zero value denies everything, deliberately. ADR-032 §8 requires that a
// machine whose identity cannot be established denies every non-local pool, so
// the unknown case degrades to "no cross-pool dispatch" rather than to "all
// pools allowed" — and a caller that forgets to populate this gets the safe
// answer rather than the permissive one.
type MachinePolicy struct {
	// Identified is false when machine.json declares no id. It is not an error
	// condition: an unconfigured machine is a normal state, it is simply one
	// that may not dispatch anywhere until it says who it is.
	Identified bool
	ID         string
	Deny       []string

	denied map[string]bool
}

// Denies reports whether this machine forbids dispatching to pool.
//
// An unidentified machine denies everything. That is the security-relevant
// default and the reason this type exists: a rebuilt corporate machine that has
// not restored machine.json probes successfully for personal pools, and the
// failure of allowing them would be silent and in the wrong direction.
func (p MachinePolicy) Denies(pool string) bool {
	if !p.Identified {
		return true
	}
	return p.denied[pool]
}

// ValidateDeny rejects a deny entry that names no declared pool.
//
// The typo fails dangerously: `claud` in the list leaves `claude` allowed on a
// machine whose whole intent was to forbid it, and nothing would ever say so.
// declared is the set of pool names the routing map declares.
func (p MachinePolicy) ValidateDeny(declared map[string]bool) error {
	var unknown []string
	for _, name := range p.Deny {
		if !declared[name] {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	known := make([]string, 0, len(declared))
	for name := range declared {
		known = append(known, name)
	}
	sort.Strings(known)
	return fmt.Errorf(
		"machine.json pools.deny names %s, which the routing map does not declare (it declares %s)\n"+
			"this is refused rather than ignored: a misspelled entry silently leaves the pool it meant "+
			"to forbid allowed, which is the direction that cannot be noticed",
		strings.Join(quoteAll(unknown), ", "), strings.Join(quoteAll(known), ", "))
}

func quoteAll(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = fmt.Sprintf("%q", s)
	}
	return out
}

// machinePolicyFile is the subset of machine.json this reads. It is a separate
// struct from `machine` because the two have different absence semantics: an
// absent `paths` is an empty override set, while an absent `machine` block is a
// machine that may not dispatch.
type machinePolicyFile struct {
	Machine struct {
		ID string `json:"id"`
	} `json:"machine"`
	Pools struct {
		Deny []string `json:"deny"`
	} `json:"pools"`
}

// LoadMachinePolicy reads the identity and denial policy from machine.json.
//
// An ABSENT file yields an unidentified policy and no error: not declaring an
// identity is a state, not a fault, and the denial that follows is the point. A
// MALFORMED file is an error: the file is there and cannot be trusted, and
// silently denying would send an operator hunting for a policy that is really a
// syntax error.
func LoadMachinePolicy(path string) (MachinePolicy, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return MachinePolicy{}, nil
		}
		return MachinePolicy{}, fmt.Errorf("read machine.json (%s): %w", path, err)
	}
	var f machinePolicyFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return MachinePolicy{}, fmt.Errorf("parse machine.json (%s): %w", path, err)
	}

	id := strings.TrimSpace(f.Machine.ID)
	p := MachinePolicy{Identified: id != "", ID: id, denied: map[string]bool{}}
	for _, name := range f.Pools.Deny {
		if n := strings.TrimSpace(name); n != "" {
			p.Deny = append(p.Deny, n)
			p.denied[n] = true
		}
	}
	return p, nil
}
