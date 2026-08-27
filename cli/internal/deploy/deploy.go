// Package deploy installs agent configuration files from the checkout to their
// deployed locations.
//
// It exists because that behaviour was implemented twice — once in
// setup-linux.sh, once in setup-windows.ps1 — for each of three configs, in two
// languages, kept in step by hand. ADR-020 C7 leaves shell the thin bootstrap
// (detect OS/arch, fetch a binary, set PATH) and assigns user-facing tooling
// logic to Go; substituting secrets into a config and installing it atomically
// is the latter. The strangler-fig rule says a twin gets ported the next time it
// is touched, and #987 was that touch (CLI-039, #1023).
package deploy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ManifestRel is the declarative table of what gets deployed where, relative to
// the repo root. A manifest rather than code for the same reason packages.json,
// registry.yaml and env-contract.json are: adding a config should be an entry,
// not a function.
const ManifestRel = "ai/deploy.json"

// Config is one deployable agent configuration.
//
// Five fields, deliberately. Mode is DECLARED rather than inferred from whether
// the file looks secret-bearing: inferring it means guessing which configs hold
// credentials, and that guess has already been wrong once (#987, where a
// credential was written into a config nobody had classified as sensitive).
type Config struct {
	Name   string `json:"name"`
	Src    string `json:"src"`    // repo-relative
	Dst    string `json:"dst"`    // may contain {VAR} tokens
	Render bool   `json:"render"` // run `secrets render` on the staged copy
	Mode   string `json:"mode"`   // octal, e.g. "0600"
}

// Manifest is the parsed ai/deploy.json.
type Manifest struct {
	Version int      `json:"version"`
	Configs []Config `json:"configs"`
}

// Outcome is what happened to one config, so callers can report precisely
// instead of saying "done".
type Outcome struct {
	Name    string
	Dst     string
	Changed bool
	DryRun  bool
}

var (
	ErrNoSuchConfig = errors.New("no such config in the manifest")
	tokenRe         = regexp.MustCompile(`\{([A-Z_][A-Z0-9_]*)\}`)
)

// ParseManifest reads and validates the manifest. Validation is fail-fast and
// names the offending entry: a manifest error must not surface as a deploy that
// silently skipped something.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse deploy manifest: %w", err)
	}
	if m.Version != 1 {
		return nil, fmt.Errorf("deploy manifest version %d unsupported (want 1)", m.Version)
	}
	seen := map[string]bool{}
	for i := range m.Configs {
		c := &m.Configs[i]
		switch {
		case c.Name == "":
			return nil, fmt.Errorf("config #%d: empty name", i)
		case seen[c.Name]:
			return nil, fmt.Errorf("duplicate config name %q", c.Name)
		case c.Src == "":
			return nil, fmt.Errorf("config %q: empty src", c.Name)
		case c.Dst == "":
			return nil, fmt.Errorf("config %q: empty dst", c.Name)
		}
		if _, err := c.FileMode(); err != nil {
			return nil, fmt.Errorf("config %q: %w", c.Name, err)
		}
		seen[c.Name] = true
	}
	return &m, nil
}

// FileMode parses the declared octal mode, defaulting to 0644 when unset. A
// config that carries a credential declares 0600 explicitly.
func (c Config) FileMode() (os.FileMode, error) {
	if c.Mode == "" {
		return 0o644, nil
	}
	n, err := strconv.ParseUint(c.Mode, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("mode %q is not octal", c.Mode)
	}
	return os.FileMode(n), nil
}

// Lookup finds a config by name.
func (m *Manifest) Lookup(name string) *Config {
	for i := range m.Configs {
		if m.Configs[i].Name == name {
			return &m.Configs[i]
		}
	}
	return nil
}

// ExpandDst resolves {VAR} tokens in a destination.
//
// resolve is the env seam (env.ResolvePath in production), so destinations go
// through the same cascade every other path in this repo uses (ADR-025) rather
// than being hardcoded per OS — which is precisely what the two shell copies did
// differently. An unresolvable token is an error, never an empty segment: a path
// silently becoming "/models.json" is how a deploy lands somewhere nobody looks.
func ExpandDst(dst, home string, resolve func(string) string) (string, error) {
	var bad []string
	out := tokenRe.ReplaceAllStringFunc(dst, func(tok string) string {
		name := tok[1 : len(tok)-1]
		if name == "HOME" {
			return home
		}
		if v := resolve(name); v != "" {
			return v
		}
		bad = append(bad, name)
		return tok
	})
	if len(bad) > 0 {
		return "", fmt.Errorf("destination %q: unresolvable path variable(s) %s", dst, strings.Join(bad, ", "))
	}
	// The manifest spells destinations with "/" on every OS; the resolved
	// {HOME} is native. Normalise so the result is a path in the OS's own form
	// rather than `C:\Users\u/.pi/agent/models.json` — accepted by the syscall,
	// but never equal to the filepath.Join'ed path a check compares it against
	// (CLI-054/#1301).
	return filepath.Clean(filepath.FromSlash(out)), nil
}

// Renderer materialises {env:VAR} placeholders in a staged file. The seam exists
// so this package CALLS the existing implementation rather than growing a second
// one — a second substitution implementation is the defect this port removes,
// not a thing to reintroduce in Go.
type Renderer func(path string) error

// Deploy installs one config: stage, render, compare, install.
//
// The compare step is why this is not a copy. A deployed config that already
// matches must not be rewritten — rewriting churns mtime on every setup run,
// which makes "did this change?" unanswerable for the operator and for any
// check that watches the file.
func Deploy(c Config, repoRoot, home string, resolve func(string) string, render Renderer, dryRun bool) (Outcome, error) {
	out := Outcome{Name: c.Name, DryRun: dryRun}

	src := filepath.Join(repoRoot, filepath.FromSlash(c.Src))
	srcData, err := os.ReadFile(src) //nolint:gosec // repo-relative, manifest-declared
	if err != nil {
		return out, fmt.Errorf("config %q: source %s: %w", c.Name, c.Src, err)
	}

	dst, err := ExpandDst(c.Dst, home, resolve)
	if err != nil {
		return out, fmt.Errorf("config %q: %w", c.Name, err)
	}
	out.Dst = dst

	mode, err := c.FileMode()
	if err != nil {
		return out, fmt.Errorf("config %q: %w", c.Name, err)
	}

	// Stage in the destination directory, not /tmp: an atomic rename requires
	// the same filesystem, and a cross-device rename is the failure that turns
	// an install into a half-written config.
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return out, fmt.Errorf("config %q: destination directory: %w", c.Name, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".deploy-*")
	if err != nil {
		return out, fmt.Errorf("config %q: stage: %w", c.Name, err)
	}
	staged := tmp.Name()
	defer func() { _ = os.Remove(staged) }() // no-op once renamed away

	if _, err := tmp.Write(srcData); err != nil {
		_ = tmp.Close()
		return out, fmt.Errorf("config %q: stage: %w", c.Name, err)
	}
	if err := tmp.Close(); err != nil {
		return out, fmt.Errorf("config %q: stage: %w", c.Name, err)
	}
	if err := os.Chmod(staged, mode); err != nil {
		return out, fmt.Errorf("config %q: stage mode: %w", c.Name, err)
	}

	if c.Render && render != nil {
		if err := render(staged); err != nil {
			return out, fmt.Errorf("config %q: render: %w", c.Name, err)
		}
	}

	stagedData, err := os.ReadFile(staged) //nolint:gosec // path this function just created
	if err != nil {
		return out, fmt.Errorf("config %q: re-read staged copy: %w", c.Name, err)
	}
	if existing, err := os.ReadFile(dst); err == nil && string(existing) == string(stagedData) { //nolint:gosec // manifest-declared destination
		return out, nil // in sync; Changed stays false and nothing is rewritten
	}

	out.Changed = true
	if dryRun {
		return out, nil
	}
	if err := os.Rename(staged, dst); err != nil {
		return out, fmt.Errorf("config %q: install to %s: %w", c.Name, dst, err)
	}
	// Rename preserves the staged mode, but an existing destination replaced by
	// rename keeps the NEW inode's bits — so this is belt-and-braces for the
	// case that matters: a 0600 config must never end up 0644.
	if err := os.Chmod(dst, mode); err != nil {
		return out, fmt.Errorf("config %q: mode on %s: %w", c.Name, dst, err)
	}
	return out, nil
}
