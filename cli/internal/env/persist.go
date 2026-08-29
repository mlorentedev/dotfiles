package env

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// UserEnvReader reads the per-user PERSISTENT environment: what a process
// started with no profile at all — Copilot's `pwsh -NoProfile` tool calls, a
// Scheduled Task, Explorer — inherits. On Windows it is HKCU\Environment; the
// rc files' cascade (paths.sh / paths.ps1) never reaches those processes, which
// is why `dotf harness mirror` could not find the checkout from inside a
// Copilot tool call (CLI-058, #1324).
//
// The read-only callers (`--check`, doctor) take this and nothing wider, so a
// reader never has to carry a write it cannot honestly implement.
type UserEnvReader interface {
	// Get returns the stored value and whether the name exists.
	Get(name string) (string, bool, error)
}

// UserEnvStore is the reader plus the two writes Persist needs.
type UserEnvStore interface {
	UserEnvReader
	// Set writes name=value; a later Get must return it.
	Set(name, value string) error
	// Delete removes name. Deleting a name that does not exist SUCCEEDS: the
	// sweep is driven by the marker, and a marker can name what a hand edit
	// or an earlier failure already removed. The registry store and the test
	// fake both honour this; a store that errors on absence would make the
	// sweep fail on the second run.
	Delete(name string) error
}

// ErrUserEnvUnsupported is returned by NewUserEnvStore where no per-user
// persistent scope exists that a profile-less process reads (Linux, macOS):
// there the rc files already source paths.sh, and persisting is a no-op.
var ErrUserEnvUnsupported = errors.New("no per-user persistent environment scope on this OS")

// ManagedMarker is the store value naming every variable dotf persisted — the
// sorted names joined by ";". It is the ownership record the sweep is bounded
// to (CLI-065, #1363): a retired contract name is deleted only if the marker
// says dotf wrote it, so a variable the user set is never touched, whatever
// its name. Named so its owner is obvious in the Environment Variables dialog.
const ManagedMarker = "DOTF_MANAGED_ENV"

const markerSep = ";"

// PersistResult is one store name's outcome.
type PersistResult struct {
	Name    string
	Value   string
	Changed bool
	// Removed marks a retired name the sweep deleted; Value is empty then.
	Removed bool
}

// ParseMarker splits a stored marker into names. An empty or missing marker
// is an empty set, not the one-element [""] strings.Split would give.
func ParseMarker(s string) []string {
	var out []string
	for _, part := range strings.Split(s, markerSep) {
		if name := strings.TrimSpace(part); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// MarkerValue renders the marker for vars: the names, sorted and deduplicated,
// so two runs over the same contract produce the same bytes.
func MarkerValue(vars []ResolvedVar) string {
	seen := make(map[string]bool, len(vars))
	names := make([]string, 0, len(vars))
	for _, v := range vars {
		if v.Name == "" || seen[v.Name] {
			continue
		}
		seen[v.Name] = true
		names = append(names, v.Name)
	}
	sort.Strings(names)
	return strings.Join(names, markerSep)
}

// Leftovers is the sweep set: the names marker lists that no variable in vars
// names. ONE definition for three callers — Persist deletes what it returns,
// `--check` prints it, doctor WARNs on it — so the two reports cannot drift
// from what the write actually removes.
//
// Names compare case-insensitively because registry value names are: a
// case-only rename in the contract (Foo → FOO) is a rewrite of the same value,
// not a leftover, and treating it as one would delete the value the same run
// just wrote. The marker's own name is never a leftover.
func Leftovers(marker []string, vars []ResolvedVar) []string {
	seen := make(map[string]bool, len(marker))
	var out []string
	for _, name := range marker {
		if name == "" || strings.EqualFold(name, ManagedMarker) {
			continue
		}
		key := strings.ToUpper(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		if !namesVar(vars, name) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func namesVar(vars []ResolvedVar, name string) bool {
	for _, v := range vars {
		if strings.EqualFold(v.Name, name) {
			return true
		}
	}
	return false
}

// Retired reads the marker through reader and returns the leftovers that are
// STILL IN THE STORE — the read-only half of the sweep, for `--check` and for
// doctor. A name the marker lists that a hand edit already removed is not
// "still persisted"; it is only a stale record, which MarkerStale reports.
func Retired(reader UserEnvReader, vars []ResolvedVar) ([]string, error) {
	cur, ok, err := reader.Get(ManagedMarker)
	if err != nil {
		return nil, fmt.Errorf("read persisted %s: %w", ManagedMarker, err)
	}
	if !ok {
		return nil, nil
	}
	var present []string
	for _, name := range Leftovers(ParseMarker(cur), vars) {
		if _, there, err := reader.Get(name); err != nil {
			return nil, fmt.Errorf("read persisted %s: %w", name, err)
		} else if there {
			present = append(present, name)
		}
	}
	return present, nil
}

// MarkerStale reports whether the ownership record differs from what the
// contract names now — the marker half of what Persist would write, so that
// `--check` stays the exact mirror of `persist`: a run that `--check` calls
// clean changes nothing, marker included.
func MarkerStale(reader UserEnvReader, vars []ResolvedVar) (bool, error) {
	cur, ok, err := reader.Get(ManagedMarker)
	if err != nil {
		return false, fmt.Errorf("read persisted %s: %w", ManagedMarker, err)
	}
	return !ok || cur != MarkerValue(vars), nil
}

// checkNames refuses a contract name the marker could not round-trip: a name
// holding the separator would be read back as two names, and the next run
// could delete two unrelated values. Contract names are identifiers by
// convention; this is where the convention becomes a guarantee.
func checkNames(vars []ResolvedVar) error {
	for _, v := range vars {
		if strings.Contains(v.Name, markerSep) {
			return fmt.Errorf("contract variable name %q contains the marker separator %q and cannot be persisted", v.Name, markerSep)
		}
	}
	return nil
}

// Persist brings the store in line with the contract, touching only what
// differs (idempotent: a second run changes nothing, marker included). It
// stops at the first store error and returns what it did so far.
//
// THE ORDER IS LOAD-BEARING: sweep, then write, then mark.
//
//  1. Delete every leftover the marker owns. Before the writes, because
//     registry names are case-insensitive: on a case-only rename the old
//     spelling is a leftover and the new one is a write, and write-then-delete
//     would delete the value it had just written.
//  2. Write every contract variable whose stored value differs.
//  3. Rewrite the marker, only if it differs from what the contract now names.
//
// With no marker in the store (a box that persisted before the marker existed)
// nothing is deleted: there is no record of what dotf wrote, and guessing from
// the registry would be the unbounded sweep this exists to avoid.
func Persist(vars []ResolvedVar, store UserEnvStore) ([]PersistResult, error) {
	out := make([]PersistResult, 0, len(vars)+1)
	if err := checkNames(vars); err != nil {
		return out, err
	}
	stored, hasMarker, err := store.Get(ManagedMarker)
	if err != nil {
		return out, fmt.Errorf("read persisted %s: %w", ManagedMarker, err)
	}
	var owned []string
	if hasMarker {
		owned = ParseMarker(stored)
	}
	for _, name := range Leftovers(owned, vars) {
		// A leftover a hand edit already removed is not deleted and not
		// reported as removed; the marker rewrite below retires its record.
		if _, there, err := store.Get(name); err != nil {
			return out, fmt.Errorf("read persisted %s: %w", name, err)
		} else if !there {
			continue
		}
		if err := store.Delete(name); err != nil {
			return out, fmt.Errorf("remove retired %s: %w", name, err)
		}
		out = append(out, PersistResult{Name: name, Removed: true})
	}
	for _, v := range vars {
		cur, ok, err := store.Get(v.Name)
		if err != nil {
			return out, fmt.Errorf("read persisted %s: %w", v.Name, err)
		}
		if ok && cur == v.Value {
			out = append(out, PersistResult{Name: v.Name, Value: v.Value})
			continue
		}
		if err := store.Set(v.Name, v.Value); err != nil {
			return out, fmt.Errorf("persist %s: %w", v.Name, err)
		}
		out = append(out, PersistResult{Name: v.Name, Value: v.Value, Changed: true})
	}
	want := MarkerValue(vars)
	if hasMarker && stored == want {
		out = append(out, PersistResult{Name: ManagedMarker, Value: want})
		return out, nil
	}
	if err := store.Set(ManagedMarker, want); err != nil {
		return out, fmt.Errorf("persist %s: %w", ManagedMarker, err)
	}
	out = append(out, PersistResult{Name: ManagedMarker, Value: want, Changed: true})
	return out, nil
}

// Drift lists the resolved variables whose persisted value is missing or
// different — the read-only half of the writes, for `--check` and for doctor.
// Retired names are the other half; Retired lists those.
func Drift(vars []ResolvedVar, reader UserEnvReader) ([]ResolvedVar, error) {
	var out []ResolvedVar
	for _, v := range vars {
		cur, ok, err := reader.Get(v.Name)
		if err != nil {
			return out, fmt.Errorf("read persisted %s: %w", v.Name, err)
		}
		if !ok || cur != v.Value {
			out = append(out, v)
		}
	}
	return out, nil
}

// ResolveVars loads the contract and the machine overrides and resolves every
// structural variable for goos/home — the same list `dotf env generate`
// renders, so the persisted scope and the rc files never disagree.
func ResolveVars(contractPath, machinePath, goos, home string) ([]ResolvedVar, error) {
	c, err := loadContract(contractPath)
	if err != nil {
		return nil, err
	}
	m, err := loadMachine(machinePath)
	if err != nil {
		return nil, err
	}
	return Resolve(c, m, goos, home), nil
}
