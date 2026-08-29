package secrets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/fsmode"
)

// envToken matches a {env:NAME} placeholder, NAME being an upper-snake env-var
// identifier. Same grammar the shell twin (substitute_env_placeholders) used.
var envToken = regexp.MustCompile(`\{env:([A-Z_][A-Z0-9_]*)\}`)

// Result reports what Render did, so the caller can surface precise operator
// diagnostics and decide whether to fail. A mapped placeholder that didn't resolve
// is split two ways (#612 A2): Missing (the secret is genuinely absent on this
// machine — expected during partial setup, quiet/info) vs Unresolved (a real
// failure — wrong key, locked vault, empty value, bw item/field typo — surfaced
// loudly with the specific error, and fatal under --strict).
type Result struct {
	Substituted int
	Unmapped    []string        // {env:VAR} with no registry entry — left intact (info)
	Missing     []string        // mapped but the secret is absent (age file missing) — left intact (info)
	Unresolved  []UnresolvedVar // mapped but a real failure — left intact, surfaced with its error
}

// UnresolvedVar pairs a placeholder var with the specific error that blocked it, so
// render no longer collapses every distinct failure into one opaque bucket.
type UnresolvedVar struct {
	Var string
	Err error
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
		value, err := resolve(loader, entry)
		if err != nil {
			// Absent (secret not provisioned here) stays quiet/non-fatal; any other
			// error is a real failure surfaced with its specific cause (#612 A2).
			if errors.Is(err, ErrSecretAbsent) {
				res.Missing = append(res.Missing, name)
			} else {
				res.Unresolved = append(res.Unresolved, UnresolvedVar{Var: name, Err: err})
			}
			continue
		}
		out = strings.ReplaceAll(out, "{env:"+name+"}", value)
		res.Substituted++
	}

	if res.Substituted == 0 {
		return res, nil // every placeholder unmapped/unresolved — no rewrite needed
	}
	if err := AtomicWrite(path, []byte(out)); err != nil {
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
		// Compare the union's own identity, not File: File is "" for every bw
		// entry, so a File comparison reported all of them as one source and this
		// guard could not fire for the backend 28 secrets now live on
		// (REFACTOR-012). Display the sources rather than the comparison key —
		// the pre-fix message printed "(, )" for a bw pair.
		if prev, dup := bySource[e.Var]; dup && prev.SourceID() != e.SourceID() {
			return nil, fmt.Errorf("render: registry exposes %q from two sources (%s, %s); the registry must map each var to one secret", e.Var, prev.SourceDisplay(), e.SourceDisplay())
		}
		bySource[e.Var] = e
	}
	return bySource, nil
}

// resolve decrypts a single env entry by reusing EnvFor (the one resolve+strip path
// shared with run/show). It returns the underlying error verbatim (no longer
// swallowed) so Render can classify absent vs misconfigured and surface the specific
// cause. EnvFor already rejects an empty value, so an empty secret arrives here as a
// real error, not a silent "".
func resolve(loader *Loader, e Entry) (string, error) {
	kv, err := loader.EnvFor([]Entry{e}, nil)
	if err != nil {
		return "", err
	}
	if len(kv) == 0 {
		return "", fmt.Errorf("no value produced for %q", e.Var)
	}
	_, value, _ := strings.Cut(kv[0], "=")
	return value, nil
}

// AtomicWrite stages content to a temp file in the target's directory, sets 0600,
// and renames it over the target. Shared by render (config files) and migrate (the
// registry cutover) — both want the secret-file 0600 default.
func AtomicWrite(path string, content []byte) error {
	return AtomicWriteMode(path, content, 0o600)
}

// AtomicWriteMode is AtomicWrite with caller-chosen permission bits. Same-dir temp
// keeps the rename on one filesystem; os.Rename replaces an existing file on both
// POSIX and Windows (MoveFileEx). The chmod is applied to the temp file before the
// rename, so the target never appears with looser-than-intended permissions.
func AtomicWriteMode(path string, content []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("atomic write: create temp for %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // no-op once the rename succeeds

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("atomic write: write temp for %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("atomic write: close temp for %s: %w", path, err)
	}
	// fsmode, not os.Chmod: on Windows a 0600 credential file gets an owner-only
	// DACL, which the rename below carries to its final name (CLI-055).
	if err := fsmode.Apply(tmpName, mode); err != nil {
		return fmt.Errorf("atomic write: chmod temp for %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("atomic write: replace %s: %w", path, err)
	}
	return nil
}
