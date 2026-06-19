// Package env owns cross-machine path resolution and the generation of the
// per-machine path files (paths.sh / paths.ps1) that shells source and the
// session hooks read (ADR-025).
//
// The resolution cascade, per path key, is:
//
//  1. an explicit environment variable already set in the process  (highest)
//  2. ~/.config/dotfiles/machine.json  (the per-machine, gitignored override)
//  3. env-contract.json default for the current OS (runtime.GOOS)    (lowest)
//
// `env` deliberately does NOT import `doctor`: the dependency runs the other
// way (doctor imports env for its drift check), so the contract types are
// modelled here as a focused, read-only view rather than shared with doctor's
// richer Contract.
package env

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// contract is the minimal view of env-contract.json this package needs: the
// declared structural vars and their per-OS default. Fields doctor cares about
// (required, validation, ...) are intentionally omitted — JSON unmarshalling
// drops them.
type contract struct {
	EnvVars []contractVar `json:"env_vars"`
}

type contractVar struct {
	Name    string            `json:"name"`
	Default map[string]string `json:"default"`
}

// machine is the per-machine override file (~/.config/dotfiles/machine.json):
//
//	{ "paths": { "DOTFILES_REPO_DIR": "...", "VAULT_PATH": "..." } }
//
// Only the keys a machine overrides need to be present; everything else falls
// through to the contract default. An absent file is valid (no overrides).
type machine struct {
	Paths map[string]string `json:"paths"`
}

// loadContract reads and parses env-contract.json at path. A missing or
// malformed contract is a hard error — generation cannot run without it.
func loadContract(path string) (*contract, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read env-contract.json: %w", err)
	}
	var c contract
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse env-contract.json (%s): %w", path, err)
	}
	return &c, nil
}

// loadMachine reads ~/.config/dotfiles/machine.json. A missing file yields an
// empty (override-free) machine and no error; a malformed file is an error so
// a typo in the override never silently reverts to defaults.
func loadMachine(path string) (*machine, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &machine{Paths: map[string]string{}}, nil
		}
		return nil, fmt.Errorf("read machine.json: %w", err)
	}
	var m machine
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse machine.json (%s): %w", path, err)
	}
	if m.Paths == nil {
		m.Paths = map[string]string{}
	}
	return &m, nil
}

// Home resolves the user's home directory, preferring HOME (POSIX) then
// USERPROFILE (Windows) — matching the env-contract's OS-scoped vars and
// doctor's System.home().
func Home() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}

// DotfilesDir resolves DOTFILES_DIR, defaulting to <home>/.dotfiles — the
// deployed location where the generated path file lands.
func DotfilesDir(home string) string {
	if d := os.Getenv("DOTFILES_DIR"); d != "" {
		return d
	}
	return filepath.Join(home, ".dotfiles")
}

// MachinePath returns the per-machine override file location, honoring
// $XDG_CONFIG_HOME and falling back to <home>/.config/dotfiles/machine.json
// (Windows: %USERPROFILE%\.config\dotfiles\machine.json).
func MachinePath(home string) string {
	cfg := os.Getenv("XDG_CONFIG_HOME")
	if cfg == "" {
		cfg = filepath.Join(home, ".config")
	}
	return filepath.Join(cfg, "dotfiles", "machine.json")
}

// ResolveContractPath locates env-contract.json: first under DOTFILES_DIR (the
// deployed copy), then by walking up from the current directory (so the CLI
// works from inside a checkout). Returns "" when none is found.
func ResolveContractPath() string {
	home := Home()
	if p := filepath.Join(DotfilesDir(home), "env-contract.json"); fileExists(p) {
		return p
	}
	if cwd, err := os.Getwd(); err == nil {
		if p := walkUpFor(cwd, "env-contract.json"); p != "" {
			return p
		}
	}
	return ""
}

// walkUpFor walks up from start looking for a file named name, returning its
// full path or "".
func walkUpFor(start, name string) string {
	dir := start
	for {
		p := filepath.Join(dir, name)
		if fileExists(p) {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// ResolvePath resolves a single path key through the full cascade. Used by
// other Go callers (e.g. vault.ResolveVault) that want one value without
// sourcing the generated file. Returns "" when the key has no env value, no
// override, and no contract default for this OS.
func ResolvePath(name string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	contractPath := ResolveContractPath()
	if contractPath == "" {
		return ""
	}
	c, err := loadContract(contractPath)
	if err != nil {
		return ""
	}
	home := Home()
	m, err := loadMachine(MachinePath(home))
	if err != nil {
		return ""
	}
	for _, rv := range Resolve(c, m, runtime.GOOS, home) {
		if rv.Name == name {
			return rv.Value
		}
	}
	return ""
}
