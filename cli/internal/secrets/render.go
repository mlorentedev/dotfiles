package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// envToken matches a {env:NAME} placeholder, NAME being an upper-snake env-var
// identifier. Same grammar the shell twin (substitute_env_placeholders) used.
var envToken = regexp.MustCompile(`\{env:([A-Z_][A-Z0-9_]*)\}`)

// Result reports what Render did, so the caller can surface the shell twin's
// operator diagnostics: which placeholders were substituted, left for the
// runtime resolver (Unmapped), or mapped-but-undecryptable (Unresolved).
type Result struct {
	Substituted int
	Unmapped    []string // {env:VAR} with no registry entry — left intact (info)
	Unresolved  []string // mapped but decrypt failed/empty — left intact (warning)
}

// Render rewrites the config at path in place, substituting every {env:VAR}
// placeholder whose VAR is a registry-exposed env secret with that secret's
// decrypted value. It is the Go port of the substitute_env_placeholders /
// Substitute-EnvPlaceholders twins (ADR-020 convergence over the registry SSOT).
//
// Behaviour parity with the twins:
//   - unmapped VAR (no registry entry, e.g. {env:HOME})  -> left intact (Unmapped)
//   - mapped VAR but the secret won't decrypt/is empty   -> left intact (Unresolved)
//   - resolved                                           -> substituted in place
//
// Setup must complete even when some secrets are absent, so an undecryptable
// secret is NOT fatal (unlike run's EnvFor, which fails fast): the placeholder
// falls through to opencode/pi's runtime resolver. The one fatal case render
// adds — which the twins lacked — is a non-deterministic registry: a VAR exposed
// by two distinct secrets has no single source, so render refuses it.
//
// The write is atomic (temp file in the same dir, then rename) at 0600, with no
// trailing-newline drift. A file with no placeholders is left untouched.
func Render(path string, reg *Registry, loader *Loader, home string) (Result, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("render: read %s: %w", path, err)
	}

	refs := referencedVars(content)
	if len(refs) == 0 {
		return Result{}, nil // nothing to substitute — leave the file as-is
	}

	bySource, err := envSourceMap(reg, home)
	if err != nil {
		return Result{}, err
	}

	out := string(content)
	res := Result{}
	for _, name := range refs {
		entry, mapped := bySource[name]
		if !mapped {
			res.Unmapped = append(res.Unmapped, name)
			continue
		}
		value, ok := resolve(loader, entry)
		if !ok {
			res.Unresolved = append(res.Unresolved, name)
			continue
		}
		out = strings.ReplaceAll(out, "{env:"+name+"}", value)
		res.Substituted++
	}

	if res.Substituted == 0 {
		return res, nil // every placeholder unmapped/unresolved — no rewrite needed
	}
	if err := atomicWrite(path, []byte(out)); err != nil {
		return res, err
	}
	return res, nil
}

// referencedVars returns the sorted, de-duplicated set of VAR names appearing as
// {env:VAR} in content. Sorted so Render's behaviour (and its diagnostics) is
// deterministic regardless of placeholder order in the file.
func referencedVars(content []byte) []string {
	seen := map[string]bool{}
	for _, m := range envToken.FindAllSubmatch(content, -1) {
		seen[string(m[1])] = true
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// envSourceMap flattens the registry to a VAR -> age entry map over env secrets
// only (file secrets are materialized by `run`, never substituted into configs).
// A VAR exposed by two distinct secrets is a non-deterministic registry and is
// rejected fail-fast — render needs exactly one source per var.
func envSourceMap(reg *Registry, home string) (map[string]Entry, error) {
	bySource := map[string]Entry{}
	for _, e := range reg.Entries(home) {
		if e.IsFile {
			continue
		}
		if prev, dup := bySource[e.Var]; dup && prev.File != e.File {
			return nil, fmt.Errorf("render: registry exposes %q from two sources (%s, %s); the registry must map each var to one secret", e.Var, prev.File, e.File)
		}
		bySource[e.Var] = e
	}
	return bySource, nil
}

// resolve decrypts a single env entry by reusing EnvFor (the one decrypt+strip
// path shared with run/show). EnvFor fails fast; here a failure is non-fatal —
// it maps to "unresolved, leave the placeholder intact" (twin parity).
func resolve(loader *Loader, e Entry) (string, bool) {
	kv, err := loader.EnvFor([]Entry{e}, nil)
	if err != nil || len(kv) == 0 {
		return "", false
	}
	_, value, _ := strings.Cut(kv[0], "=")
	if value == "" {
		return "", false
	}
	return value, true
}

// atomicWrite stages content to a temp file in the target's directory, sets 0600,
// and renames it over the target. Same-dir keeps the rename on one filesystem;
// os.Rename replaces an existing file on both POSIX and Windows (MoveFileEx).
func atomicWrite(path string, content []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".render-*.tmp")
	if err != nil {
		return fmt.Errorf("render: create temp for %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return fmt.Errorf("render: write temp for %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("render: close temp for %s: %w", path, err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("render: chmod temp for %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("render: replace %s: %w", path, err)
	}
	return nil
}
