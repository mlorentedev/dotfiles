package env

import (
	"errors"
	"fmt"
)

// UserEnvStore is the per-user PERSISTENT environment: what a process started
// with no profile at all — Copilot's `pwsh -NoProfile` tool calls, a Scheduled
// Task, Explorer — inherits. On Windows it is HKCU\Environment; the rc files'
// cascade (paths.sh / paths.ps1) never reaches those processes, which is why
// `dotf harness mirror` could not find the checkout from inside a Copilot tool
// call (CLI-058, #1324).
type UserEnvStore interface {
	// Get returns the stored value and whether the name exists.
	Get(name string) (string, bool, error)
	// Set writes name=value; a later Get must return it.
	Set(name, value string) error
}

// ErrUserEnvUnsupported is returned by NewUserEnvStore where no per-user
// persistent scope exists that a profile-less process reads (Linux, macOS):
// there the rc files already source paths.sh, and persisting is a no-op.
var ErrUserEnvUnsupported = errors.New("no per-user persistent environment scope on this OS")

// PersistResult is one contract variable's outcome.
type PersistResult struct {
	Name    string
	Value   string
	Changed bool
}

// Persist writes every resolved contract variable into store, touching only
// the ones whose stored value differs (idempotent: a second run changes
// nothing). It stops at the first store error and returns what it did so far.
func Persist(vars []ResolvedVar, store UserEnvStore) ([]PersistResult, error) {
	out := make([]PersistResult, 0, len(vars))
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
	return out, nil
}

// Drift lists the resolved variables whose persisted value is missing or
// different — the read-only half of Persist, for `--check` and for doctor.
func Drift(vars []ResolvedVar, store UserEnvStore) ([]ResolvedVar, error) {
	var out []ResolvedVar
	for _, v := range vars {
		cur, ok, err := store.Get(v.Name)
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
