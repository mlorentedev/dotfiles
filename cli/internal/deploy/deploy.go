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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// ManifestRel is the declarative table of what gets deployed where, relative to
// the repo root. A manifest rather than code for the same reason packages.json,
// registry.yaml and env-contract.json are: adding a config should be an entry,
// not a function.
const ManifestRel = "ai/deploy.json"

// ManifestVersion is the schema this binary reads. It is bumped whenever a new
// field changes what an entry MEANS (2: `strategy` and `requires`, AI-039), so a
// binary that predates the field refuses the manifest instead of deploying
// every entry the old way. The check is the version, not the field, because
// a field an old decoder ignores is invisible to it by construction.
const ManifestVersion = 2

// Strategies. Replace installs the source as the whole destination; merge
// writes the source's top-level keys into the destination's JSON object and
// leaves every other key alone (AI-039, #1322) — the shape a config needs when
// the tool that reads it also writes it, as Copilot's settings.json is.
const (
	StrategyReplace = "replace"
	StrategyMerge   = "merge"
)

// Config is one deployable agent configuration.
//
// Mode is DECLARED rather than inferred from whether the file looks
// secret-bearing: inferring it means guessing which configs hold credentials,
// and that guess has already been wrong once (#987, where a credential was
// written into a config nobody had classified as sensitive). Strategy is
// declared for the same reason: which files another program co-owns is a fact
// about the file, not something a deploy can detect.
type Config struct {
	Name     string `json:"name"`
	Src      string `json:"src"`      // repo-relative
	Dst      string `json:"dst"`      // may contain {VAR} tokens
	Render   bool   `json:"render"`   // run `secrets render` on the staged copy
	Mode     string `json:"mode"`     // octal, e.g. "0600"
	Strategy string `json:"strategy"` // "replace" (default) | "merge"
	// Requires names a command that must be on PATH for the entry to apply;
	// absent, the entry is skipped and said so. A config for a tool the box
	// does not carry is a file nobody reads and a doctor row no remedy clears
	// (#843), and the integration guard asserts ~/.copilot is never created
	// on a box without copilot (#1312).
	Requires string `json:"requires"`
}

// Manifest is the parsed ai/deploy.json.
type Manifest struct {
	Comment []string `json:"$comment"` // documentation, kept so DisallowUnknownFields allows it
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

// Plan is what a deploy of a non-rendered config would do, computed without
// touching the destination or its directory. It is the read-only half of Deploy,
// and the half a diagnostic may call: a check that creates ~/.copilot/ while
// asking whether ~/.copilot/settings.json is in sync has answered a different
// question than it was asked.
type Plan struct {
	Dst     string
	Content []byte // what the destination holds after the deploy
	Changed bool
}

var (
	ErrNoSuchConfig = errors.New("no such config in the manifest")
	// ErrNeedsRender marks a config whose installed content is only known after
	// `secrets render` ran on a staged copy — Plan cannot answer for it.
	ErrNeedsRender = errors.New("rendered config cannot be planned without rendering")
	tokenRe        = regexp.MustCompile(`\{([A-Z_][A-Z0-9_]*)\}`)
)

// ParseManifest reads and validates the manifest. Validation is fail-fast and
// names the offending entry: a manifest error must not surface as a deploy that
// silently skipped something.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	// Unknown fields are refused, not ignored. encoding/json's default drops a
	// key it has no struct field for, which is how a binary predating
	// `strategy` and `requires` read the AI-039 manifest as "replace
	// everything" and would have wiped the box's own Copilot settings — the
	// exact loss the merge strategy exists to prevent. A manifest this binary
	// cannot fully read is one it must not act on.
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("parse deploy manifest: %w (a field this dotf does not know? rebuild or update dotf)", err)
	}
	if m.Version != ManifestVersion {
		return nil, fmt.Errorf("deploy manifest version %d unsupported (this dotf reads %d; update dotf, or the checkout, so they agree)", m.Version, ManifestVersion)
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
		switch c.strategy() {
		case StrategyReplace:
		case StrategyMerge:
			if c.Render {
				return nil, fmt.Errorf("config %q: strategy merge cannot render (unsupported)", c.Name)
			}
		default:
			return nil, fmt.Errorf("config %q: unknown strategy %q (want %s or %s)", c.Name, c.Strategy, StrategyReplace, StrategyMerge)
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

func (c Config) strategy() string {
	if c.Strategy == "" {
		return StrategyReplace
	}
	return c.Strategy
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

// PlanConfig computes what Deploy would install for a non-rendered config and
// whether the destination already holds it. Nothing under the destination is
// created or written; a rendered config returns ErrNeedsRender.
func PlanConfig(c Config, repoRoot, home string, resolve func(string) string) (Plan, error) {
	if c.Render {
		return Plan{}, fmt.Errorf("config %q: %w", c.Name, ErrNeedsRender)
	}
	srcData, dst, err := load(c, repoRoot, home, resolve)
	if err != nil {
		return Plan{}, err
	}
	p := Plan{Dst: dst}
	switch c.strategy() {
	case StrategyMerge:
		p.Content, p.Changed, err = mergeInto(dst, srcData)
		if err != nil {
			return Plan{}, fmt.Errorf("config %q: merge into %s: %w", c.Name, dst, err)
		}
	default:
		p.Content = srcData
		existing, readErr := os.ReadFile(dst) //nolint:gosec // manifest-declared destination
		p.Changed = readErr != nil || !bytes.Equal(existing, srcData)
	}
	return p, nil
}

// Deploy installs one config: plan (or stage and render), compare, install.
//
// The compare step is why this is not a copy. A deployed config that already
// matches must not be rewritten — rewriting churns mtime on every setup run,
// which makes "did this change?" unanswerable for the operator and for any
// check that watches the file. For a non-rendered config the compare comes
// BEFORE anything touches the destination directory, so an in-sync or dry-run
// deploy leaves the filesystem exactly as it found it.
func Deploy(c Config, repoRoot, home string, resolve func(string) string, render Renderer, dryRun bool) (Outcome, error) {
	out := Outcome{Name: c.Name, DryRun: dryRun}
	mode, err := c.FileMode()
	if err != nil {
		return out, fmt.Errorf("config %q: %w", c.Name, err)
	}

	if !c.Render {
		p, err := PlanConfig(c, repoRoot, home, resolve)
		if err != nil {
			return out, err
		}
		out.Dst = p.Dst
		if !p.Changed {
			return out, nil // in sync; Changed stays false and nothing is rewritten
		}
		out.Changed = true
		if dryRun {
			return out, nil
		}
		staged, err := stage(c, p.Dst, p.Content, mode)
		if err != nil {
			return out, err
		}
		defer func() { _ = os.Remove(staged) }() // no-op once renamed away
		return out, commit(c, staged, p.Dst, mode)
	}

	// A rendered config is only comparable after `secrets render` ran on a
	// staged copy, so this path stages first and compares second.
	srcData, dst, err := load(c, repoRoot, home, resolve)
	if err != nil {
		return out, err
	}
	out.Dst = dst
	staged, err := stage(c, dst, srcData, mode)
	if err != nil {
		return out, err
	}
	defer func() { _ = os.Remove(staged) }() // no-op once renamed away

	if render != nil {
		if err := render(staged); err != nil {
			return out, fmt.Errorf("config %q: render: %w", c.Name, err)
		}
	}
	stagedData, err := os.ReadFile(staged) //nolint:gosec // path this function just created
	if err != nil {
		return out, fmt.Errorf("config %q: re-read staged copy: %w", c.Name, err)
	}
	if existing, err := os.ReadFile(dst); err == nil && bytes.Equal(existing, stagedData) { //nolint:gosec // manifest-declared destination
		return out, nil // in sync; Changed stays false and nothing is rewritten
	}
	out.Changed = true
	if dryRun {
		return out, nil
	}
	return out, commit(c, staged, dst, mode)
}

// load reads the source and resolves the destination.
func load(c Config, repoRoot, home string, resolve func(string) string) ([]byte, string, error) {
	src := filepath.Join(repoRoot, filepath.FromSlash(c.Src))
	srcData, err := os.ReadFile(src) //nolint:gosec // repo-relative, manifest-declared
	if err != nil {
		return nil, "", fmt.Errorf("config %q: source %s: %w", c.Name, c.Src, err)
	}
	dst, err := ExpandDst(c.Dst, home, resolve)
	if err != nil {
		return nil, "", fmt.Errorf("config %q: %w", c.Name, err)
	}
	return srcData, dst, nil
}

// stage writes data to a temp file beside the destination, with the declared
// mode. Beside it, not /tmp: an atomic rename requires the same filesystem,
// and a cross-device rename is the failure that turns an install into a
// half-written config.
func stage(c Config, dst string, data []byte, mode os.FileMode) (string, error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", fmt.Errorf("config %q: destination directory: %w", c.Name, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".deploy-*")
	if err != nil {
		return "", fmt.Errorf("config %q: stage: %w", c.Name, err)
	}
	staged := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return staged, fmt.Errorf("config %q: stage: %w", c.Name, err)
	}
	if err := tmp.Close(); err != nil {
		return staged, fmt.Errorf("config %q: stage: %w", c.Name, err)
	}
	if err := os.Chmod(staged, mode); err != nil {
		return staged, fmt.Errorf("config %q: stage mode: %w", c.Name, err)
	}
	return staged, nil
}

// commit moves the staged file over the destination atomically.
func commit(c Config, staged, dst string, mode os.FileMode) error {
	if err := os.Rename(staged, dst); err != nil {
		return fmt.Errorf("config %q: install to %s: %w", c.Name, dst, err)
	}
	// Rename preserves the staged mode, but an existing destination replaced by
	// rename keeps the NEW inode's bits — so this is belt-and-braces for the
	// case that matters: a 0600 config must never end up 0644.
	if err := os.Chmod(dst, mode); err != nil {
		return fmt.Errorf("config %q: mode on %s: %w", c.Name, dst, err)
	}
	return nil
}

// mergeInto returns the destination's JSON object with the source's top-level
// keys written into it, and whether any of them differed. Equality is semantic
// (parsed values), never textual: Copilot rewrites its config.json with a
// `// User settings belong in settings.json` header, and a byte-compare would
// call that drift on every setup run. The header is dropped on read only — the
// merged file is plain JSON, and the tool that wants a header puts it back.
//
// A missing destination merges into an empty object; a destination that is not
// a JSON object is an error, because "merge" has no meaning for it and silently
// replacing it is the data loss this strategy exists to prevent.
func mergeInto(dst string, srcData []byte) ([]byte, bool, error) {
	var managed map[string]any
	if err := json.Unmarshal(srcData, &managed); err != nil {
		return nil, false, fmt.Errorf("source is not a JSON object: %w", err)
	}
	if managed == nil {
		return nil, false, errors.New("source is not a JSON object: null")
	}
	existing := map[string]any{}
	if raw, err := os.ReadFile(dst); err == nil { //nolint:gosec // manifest-declared destination
		if err := json.Unmarshal(stripLineComments(raw), &existing); err != nil {
			return nil, false, fmt.Errorf("destination is not a JSON object: %w", err)
		}
		// A JSON `null` unmarshals into a nil map without error; assigning
		// into it would panic. Reject it like any other non-object.
		if existing == nil {
			return nil, false, errors.New("destination is not a JSON object: null")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, false, err
	}
	changed := false
	for k, v := range managed {
		if cur, ok := existing[k]; ok && reflect.DeepEqual(cur, v) {
			continue
		}
		existing[k] = v
		changed = true
	}
	content, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return nil, false, err
	}
	return append(content, '\n'), changed, nil
}

// stripLineComments removes lines that are only a `//` comment — the header
// Copilot writes into config.json — so the rest parses as JSON. Nothing else
// is touched: a `//` inside a string value is not a comment and stays.
func stripLineComments(raw []byte) []byte {
	lines := bytes.Split(raw, []byte("\n"))
	kept := lines[:0]
	for _, ln := range lines {
		if bytes.HasPrefix(bytes.TrimSpace(ln), []byte("//")) {
			continue
		}
		kept = append(kept, ln)
	}
	return bytes.Join(kept, []byte("\n"))
}
