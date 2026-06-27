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
	// Prefer the repo's contract when DOTFILES_REPO_DIR points at a real checkout.
	// On a dev machine the checkout is fresher than the deployed copy under
	// ~/.dotfiles, so this prevents the stale-deployed-copy drift where a relocated
	// repo keeps generating from an out-of-date ~/.dotfiles/env-contract.json. Read
	// with os.Getenv (not ResolvePath): ResolvePath calls back into here, so going
	// through the cascade would recurse. A non-existent path (stale value) just
	// falls through to the deployed copy below.
	if repo := os.Getenv("DOTFILES_REPO_DIR"); repo != "" {
		if p := filepath.Join(repo, "env-contract.json"); fileExists(p) {
			return p
		}
	}
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

// RepoDir resolves the dotfiles checkout root: DOTFILES_REPO_DIR when it points at
// a real directory, else walking up from the working directory for a .git entry (a
// file in a worktree, a directory in a normal clone — os.Stat matches both). Returns
// "" when neither locates a checkout. This is the shared "where is the checkout"
// seam the registry resolvers (ADR-030) build on.
func RepoDir() string {
	if r := os.Getenv("DOTFILES_REPO_DIR"); r != "" && isDir(r) {
		return r
	}
	if cwd, err := os.Getwd(); err == nil {
		if git := walkUpFor(cwd, ".git"); git != "" {
			return filepath.Dir(git)
		}
	}
	return ""
}

// ResolveRegistryPath locates secrets/registry.yaml for READS. It prefers the
// dotfiles checkout (the version-controlled SSOT, and fresher than the deployed
// copy on a dev machine), falling back to the deployed copy under DOTFILES_DIR when
// no checkout is found or the file is absent there. Mirrors ResolveContractPath and
// implements the read side of the registry source model (ADR-030, #635).
func ResolveRegistryPath() string {
	if root := RepoDir(); root != "" {
		if p := filepath.Join(root, "secrets", "registry.yaml"); fileExists(p) {
			return p
		}
	}
	return filepath.Join(DotfilesDir(Home()), "secrets", "registry.yaml")
}

// RepoRegistryPath returns secrets/registry.yaml inside the dotfiles checkout, or an
// error when no checkout is found. WRITERS (dotf secrets migrate) MUST use this: the
// registry is a version-controlled SSOT, so a mutation has to land in the checkout to
// be committed. Writing the deployed copy under ~/.dotfiles is silently reverted on
// the next redeploy and never reaches git — the durability bug behind #635. This
// fails loud rather than write a throwaway copy.
func RepoRegistryPath() (string, error) {
	root := RepoDir()
	if root == "" {
		return "", fmt.Errorf("no dotfiles checkout found (set DOTFILES_REPO_DIR or run from inside the repo) — refusing to write the registry SSOT to the deployed copy")
	}
	return filepath.Join(root, "secrets", "registry.yaml"), nil
}

// isDir reports whether p exists and is a directory.
func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
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
