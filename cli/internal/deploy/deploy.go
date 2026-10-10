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
	"io"
	"io/fs"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/fsmode"
	"github.com/mlorentedev/dotfiles/cli/internal/platform"
	"github.com/pelletier/go-toml/v2"
)

// ManifestRel is the declarative table of what gets deployed where, relative to
// the repo root. A manifest rather than code for the same reason packages.json,
// registry.yaml and env-contract.json are: adding a config should be an entry,
// not a function.
const ManifestRel = "ai/deploy.json"

// ManifestVersion is the newest schema this binary reads. It is bumped whenever
// a new field changes what an entry MEANS (2: `strategy` and `requires`,
// AI-039; 3: `paths`, AI-042; 4: `platforms`, #1843 B1), so a binary that
// predates the field refuses the manifest instead of deploying every entry the
// old way. The check is the version, not the field, because a field an old
// decoder ignores is invisible to it by construction.
const ManifestVersion = 4

// MinManifestVersion is the oldest schema this binary reads. A reader that adds
// a field reads the version before it too, which by construction carries none
// of it, so the reader ships in a release before the manifest moves: the dotf
// every machine has installed keeps reading the manifest in between (#1814).
const MinManifestVersion = 3

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
	// Paths renders {VAR} tokens inside the source's JSON string values in the
	// declared separator form ("native" | "slash") before the strategy runs
	// (AI-042, #1334). Declared per entry because which form a tool accepts is a
	// measured fact about the tool, not a property of the OS.
	Paths string `json:"paths"`
	// Platforms lists the GOOS values the entry applies on; absent means every
	// OS, and an OS that is not listed is skipped, not failed. The vocabulary
	// packages.json's platforms uses (ADR-045): a POSIX rc file is declared for
	// linux and darwin, because Git Bash on Windows would read it.
	Platforms []string `json:"platforms"`

	// dirMode is the mode for a directory the deploy creates to hold this
	// entry's file. ParseManifest derives it from every entry sharing the
	// directory (assignDirModes); zero means a Config built outside a manifest,
	// which falls back to its own mode.
	dirMode os.FileMode
}

// AppliesOn reports whether the entry is declared for goos.
func (c Config) AppliesOn(goos string) bool {
	return platform.Supports(c.Platforms, goos)
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
	// ModeFixed: the content was in sync but the declared mode was not on the
	// file — a 0600 whose ACL was still inherited (CLI-055) — and the deploy
	// applied it without rewriting the content. Changed is true alongside.
	ModeFixed bool
	// BackedUp is the path the destination's previous content was kept at, set
	// on the first deploy that replaces a file the machine already had.
	BackedUp string
}

// BackupSuffix names the one-time copy of a destination's previous content.
const BackupSuffix = ".pre-dotf"

// backupOnce keeps dst's current content at dst+BackupSuffix before the first
// replacement, so adopting a hand-made file (an rc file, a tool's own config)
// never loses it. An existing backup is never overwritten: it holds the
// machine's version, not one dotf wrote. A missing dst needs no backup.
//
// The backup takes the narrower of the file's own mode and the config's
// declared one: a 0644 settings file that held a credential must not leave a
// 0644 copy of it beside the 0600 file that replaces it.
//
// A symlink (the pre-ADR-012 leftover) is kept as a link: what the machine had
// there was a pointer, and the rename that installs the config replaces the
// pointer, not the file it named (#2054).
func backupOnce(dst string, declared os.FileMode) (string, error) {
	info, err := os.Lstat(dst)
	if err != nil {
		return "", nil //nolint:nilerr // nothing to keep: absent
	}
	backup := dst + BackupSuffix
	if _, err := os.Lstat(backup); err == nil {
		return "", nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return backupLink(dst, backup, declared)
	}
	if !info.Mode().IsRegular() {
		return "", nil // nothing to keep: a directory or device is not a config
	}
	return backupContent(dst, backup, info.Mode().Perm()&declared.Perm())
}

// backupLink re-creates the link at backup, so `ls -l` shows where it pointed
// and restoring it is a rename. Where links cannot be created (Windows without
// the privilege) it keeps the content the link resolved to instead, narrowed
// like any other backup, and a dangling link there leaves nothing to keep: the
// file it named is untouched.
func backupLink(dst, backup string, declared os.FileMode) (string, error) {
	target, err := os.Readlink(dst)
	if err != nil {
		return "", fmt.Errorf("back up %s: %w", dst, err)
	}
	if err := symlink(target, backup); err == nil {
		return backup, nil
	}
	info, err := os.Stat(dst)
	if err != nil {
		return "", nil //nolint:nilerr // dangling: no content to keep
	}
	return backupContent(dst, backup, info.Mode().Perm()&declared.Perm())
}

// symlink is os.Symlink, swappable so the fallback above runs on every host.
var symlink = os.Symlink

func backupContent(dst, backup string, mode os.FileMode) (string, error) {
	data, err := os.ReadFile(dst) //nolint:gosec // a manifest-declared destination
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(backup, data, mode); err != nil {
		return "", fmt.Errorf("back up %s: %w", dst, err)
	}
	return backup, nil
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
	// Symlink: the destination is a link (the pre-ADR-012 leftover), so it is
	// Changed whatever it resolves to. Reading through it would compare the
	// link's target, and an alias of the right bytes is still not the regular
	// file a deploy installs (#2054).
	Symlink bool
	// ModeDrift: the content is in sync but the file does not carry the mode
	// the deploy would converge it to (ModeDrift), so `dotf deploy` reports
	// `would fix mode` and doctor reports drift. It is false whenever Changed
	// is true: a rewrite installs the converged mode anyway (installMode).
	ModeDrift bool
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
	m, err := decodeManifest(data)
	if err != nil {
		return nil, err
	}
	if m.Version < MinManifestVersion || m.Version > ManifestVersion {
		return nil, fmt.Errorf("deploy manifest version %d unsupported (this dotf reads %d to %d; update dotf, or the checkout, so they agree)", m.Version, MinManifestVersion, ManifestVersion)
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
		if err := validatePlatforms(c, m.Version); err != nil {
			return nil, err
		}
		if c.Paths != "" && c.Paths != PathsNative && c.Paths != PathsSlash {
			return nil, fmt.Errorf("config %q: unknown paths form %q (want %s or %s)", c.Name, c.Paths, PathsNative, PathsSlash)
		}
		// paths rewrites the source before it is installed or merged, so the
		// source's format is the one that must be JSON.
		if c.Paths != "" && isTOML(c.Src) {
			return nil, fmt.Errorf("config %q: paths rewrites JSON string values, and %s is TOML", c.Name, c.Src)
		}
		switch c.strategy() {
		case StrategyReplace:
		case StrategyMerge:
			if c.Render {
				return nil, fmt.Errorf("config %q: strategy merge cannot render (unsupported)", c.Name)
			}
			if isTOML(c.Src) != isTOML(c.Dst) {
				return nil, fmt.Errorf("config %q: merge reads %s in the format of %s, and their formats differ", c.Name, c.Src, c.Dst)
			}
		default:
			return nil, fmt.Errorf("config %q: unknown strategy %q (want %s or %s)", c.Name, c.Strategy, StrategyReplace, StrategyMerge)
		}
		seen[c.Name] = true
	}
	assignDirModes(m.Configs)
	return &m, nil
}

// assignDirModes gives each entry the mode for a directory the deploy creates
// to hold its file: private (0700) when any entry deploying into that
// directory is private, 0755 otherwise. It is a fact about the manifest, not
// about the entry being deployed, so the result no longer depends on which
// entry creates the directory first: `dotf deploy ssh-pubkey` (0644) alone used
// to create ~/.ssh 0755 beside a 0600 ssh config (#2053).
//
// Entries are grouped by the directory of their dst template, so one directory
// spelled through two tokens ({HOME}/.claude and {CLAUDE_CONFIG_DIR}) does not
// group. The manifest holds no such pair with mixed modes.
func assignDirModes(cs []Config) {
	private := map[string]bool{}
	for _, c := range cs {
		if c.private() {
			private[path.Dir(c.Dst)] = true
		}
	}
	for i := range cs {
		cs[i].dirMode = 0o755
		if private[path.Dir(cs[i].Dst)] {
			cs[i].dirMode = 0o700
		}
	}
}

// private reports whether the entry's file grants nothing to group or others.
// ParseManifest has already refused a mode that does not parse.
func (c Config) private() bool {
	mode, err := c.FileMode()
	return err == nil && mode&0o077 == 0
}

// PrivateDirs lists, expanded and sorted, the directories that hold a private
// file this manifest deploys: an entry whose own mode is private, that applies
// here (the caller's filter: platforms, requires), and whose destination file
// exists. Doctor reports one left open after a deploy (#2053, #2161). The
// existence test keeps that report true: an entry for another OS, or one never
// deployed, puts nothing private in its directory.
func (m *Manifest) PrivateDirs(home string, resolve func(string) string, applies func(Config) bool) ([]string, error) {
	return m.privateDirs(home, resolve, applies, true)
}

// PrivateDirsOf lists the directories the named entries put a private file in,
// whether or not the file is there yet. `dotf deploy` tightens these: it is
// about to write every one of those files, so a dry run on a fresh machine
// must predict the same tightening the real run makes (#2161).
func (m *Manifest) PrivateDirsOf(home string, resolve func(string) string, names map[string]bool) ([]string, error) {
	return m.privateDirs(home, resolve, func(c Config) bool { return names[c.Name] }, false)
}

func (m *Manifest) privateDirs(home string, resolve func(string) string, applies func(Config) bool, mustExist bool) ([]string, error) {
	seen := map[string]bool{}
	var dirs []string
	for _, c := range m.Configs {
		if !c.private() || !applies(c) {
			continue
		}
		dst, err := ExpandDst(c.Dst, home, resolve)
		if err != nil {
			return nil, fmt.Errorf("config %q: %w", c.Name, err)
		}
		if _, err := os.Stat(dst); mustExist && err != nil {
			continue
		}
		if dir := filepath.Dir(dst); !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

// TightenDir removes group and other access from dir, a directory that holds
// a private deployed file, and reports the mode it had and the one it now has.
// It is the one rule `dotf deploy` and `dotf doctor --fix` both apply, so the
// two cannot disagree on what "private enough" means (#2161). It only clears
// bits: the owner's bits stay exactly as they are, so a directory is never
// widened. A directory that grants nothing to group or others, and one that is
// absent, are not a change. A dry run reports the change without making it.
//
// Windows has no POSIX directory mode to clear. Stat reports 0777 for every
// directory there, so without the goos check this would report a tightening on
// every run and change nothing.
func TightenDir(dir, goos string, dryRun bool) (from, to os.FileMode, changed bool, err error) {
	if goos == "windows" {
		return 0, 0, false, nil
	}
	info, err := os.Stat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, 0, false, nil
	}
	if err != nil {
		return 0, 0, false, err
	}
	from = info.Mode().Perm()
	to = from &^ 0o077
	if to == from {
		return from, to, false, nil
	}
	if !dryRun {
		if err := os.Chmod(dir, to); err != nil {
			return from, to, false, err
		}
	}
	return from, to, true, nil
}

// validatePlatforms refuses a platforms list a released reader could not read
// (one in a version 3 manifest), one that names no OS, and one naming an OS
// that is not a GOOS: a typo such as "macos" would skip the entry everywhere.
func validatePlatforms(c *Config, version int) error {
	switch {
	case c.Platforms == nil:
		return nil
	case version < 4:
		return fmt.Errorf("config %q: platforms needs manifest version 4", c.Name)
	case len(c.Platforms) == 0:
		return fmt.Errorf("config %q: empty platforms (omit the field for every OS)", c.Name)
	}
	if p := platform.Unknown(c.Platforms); p != "" {
		return fmt.Errorf("config %q: unknown platform %q", c.Name, p)
	}
	return nil
}

// decodeManifest reads exactly one JSON document, strictly.
//
// Unknown fields are refused, not ignored: encoding/json's default drops a key
// it has no struct field for, which is how a binary predating `strategy` and
// `requires` read the AI-039 manifest as "replace everything" and would have
// wiped the box's own Copilot settings — the exact loss the merge strategy
// exists to prevent. Trailing data is refused too: Decode reads one value and
// stops, so `{...}{...}` would otherwise deploy the first document and hide the
// second. A manifest this binary cannot fully read is one it must not act on.
func decodeManifest(data []byte) (Manifest, error) {
	var m Manifest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return m, fmt.Errorf("parse deploy manifest: %w (a field this dotf does not know? rebuild or update dotf)", err)
	}
	if err := dec.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		return m, fmt.Errorf("parse deploy manifest: trailing data after the manifest object (dotf reads one JSON document; remove what follows it)")
	}
	return m, nil
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
	p := Plan{Dst: dst, Symlink: isSymlink(dst)}
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
	p.Changed = p.Changed || p.Symlink
	if !p.Changed {
		mode, err := c.FileMode()
		if err != nil {
			return Plan{}, fmt.Errorf("config %q: %w", c.Name, err)
		}
		if _, p.ModeDrift, err = ModeDrift(dst, mode); err != nil {
			return Plan{}, fmt.Errorf("config %q: mode on %s: %w", c.Name, dst, err)
		}
	}
	return p, nil
}

// Deploy installs one config: plan (or stage and render), compare, install.
//
// The compare step is why this is not a copy. A deployed config that already
// matches must not be rewritten — rewriting churns mtime on every setup run,
// which makes "did this change?" unanswerable for the operator and for any
// check that watches the file. For a non-rendered config the compare comes
// BEFORE anything touches the destination directory, so a dry run leaves the
// filesystem exactly as it found it, and an in-sync deploy rewrites nothing: at
// most it corrects the file's mode in place (ensureMode).
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
			// In sync by content; the mode may still be off (CLI-055: a 0600
			// deployed by an older binary keeps its inherited ACL forever
			// unless someone looks). Nothing is rewritten either way.
			return ensureMode(c, out, p.Dst, mode, dryRun)
		}
		out.Changed = true
		if dryRun {
			return out, nil
		}
		install := installMode(p.Dst, mode)
		staged, err := stage(c, p.Dst, p.Content, install)
		if err != nil {
			return out, err
		}
		defer func() { _ = os.Remove(staged) }() // no-op once renamed away
		if out.BackedUp, err = backupOnce(p.Dst, install); err != nil {
			return out, err
		}
		return out, commit(c, staged, p.Dst, install)
	}

	// A rendered config is only comparable after `secrets render` ran on a
	// staged copy, so this path stages first and compares second.
	srcData, dst, err := load(c, repoRoot, home, resolve)
	if err != nil {
		return out, err
	}
	out.Dst = dst
	// A plan only compares, and staging beside dst would create its directory;
	// it stages in a scratch directory instead and leaves the filesystem as it
	// found it.
	stageAt := dst
	if dryRun {
		scratch, err := os.MkdirTemp("", "dotf-plan-*")
		if err != nil {
			return out, fmt.Errorf("config %q: plan: %w", c.Name, err)
		}
		defer func() { _ = os.RemoveAll(scratch) }()
		stageAt = filepath.Join(scratch, filepath.Base(dst))
	}
	staged, err := stage(c, stageAt, srcData, mode)
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
	if existing, err := os.ReadFile(dst); err == nil && !isSymlink(dst) && bytes.Equal(existing, stagedData) { //nolint:gosec // manifest-declared destination
		return ensureMode(c, out, dst, mode, dryRun) // in sync by content; the mode may still be off
	}
	out.Changed = true
	if dryRun {
		return out, nil
	}
	install := installMode(dst, mode)
	if out.BackedUp, err = backupOnce(dst, install); err != nil {
		return out, err
	}
	return out, commit(c, staged, dst, install)
}

// ensureMode is the in-sync path's last word: the content matches, so the only
// thing left to be wrong is the mode — on Windows, the ACL a 0600 file kept
// from its directory because an earlier binary could not express owner-only
// (CLI-055). A dry run reports the fix it would make; a real run makes it and
// reports it as ModeFixed, never as a content rewrite.
func ensureMode(c Config, out Outcome, dst string, mode os.FileMode, dryRun bool) (Outcome, error) {
	target, drift, err := ModeDrift(dst, mode)
	if err != nil {
		return out, fmt.Errorf("config %q: mode on %s: %w", c.Name, dst, err)
	}
	if !drift {
		return out, nil
	}
	out.Changed, out.ModeFixed = true, true
	if dryRun {
		return out, nil
	}
	if err := fsmode.Apply(dst, target); err != nil {
		return out, fmt.Errorf("config %q: mode on %s: %w", c.Name, dst, err)
	}
	return out, nil
}

// ModeDrift reports the mode an in-sync dst converges to, and whether it is
// not on the file yet. It is the one predicate behind both `dotf deploy`'s
// mode fix and doctor's drift line, so the two cannot disagree (#1664).
//
// The convergence only ever narrows what group and others get. The owner keeps
// exactly the declared bits, and group and others keep a declared bit only
// while the file still grants it: an operator who tightened a deployed 0644 to
// 0600 keeps 0600, and neither tool calls it drift. A bit the declaration does
// not grant is removed wherever it is. Widening a file someone deliberately
// narrowed, and reporting it only as `mode fixed`, was the defect.
//
// On Windows os.Stat reports only the read-only bit, so the target is the
// declared mode, and the tightening an operator can make there, an owner-only
// DACL, is kept by fsmode.Needs: it reads the DACL only for an owner-only mode.
func ModeDrift(dst string, declared os.FileMode) (os.FileMode, bool, error) {
	info, err := os.Stat(dst)
	if err != nil {
		return 0, false, err
	}
	target := narrowed(info.Mode(), declared)
	drift, err := fsmode.Needs(dst, target)
	return target, drift, err
}

// narrowed is the declared mode less the group and other bits the file no
// longer grants. See ModeDrift.
func narrowed(have, declared os.FileMode) os.FileMode {
	return declared.Perm()&0o700 | have.Perm()&declared.Perm()&0o077
}

// installMode is the mode a rewrite installs over dst: the same narrowing as
// the in-sync path, so new content does not undo an operator's tightening
// either. A destination that is absent, or not a regular file (the link
// #2054 replaces), has nothing to keep and gets the declared mode.
func installMode(dst string, declared os.FileMode) os.FileMode {
	info, err := os.Lstat(dst)
	if err != nil || !info.Mode().IsRegular() {
		return declared
	}
	return narrowed(info.Mode(), declared)
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
	if c.Paths != "" {
		if srcData, err = expandPaths(srcData, c.Paths, home, resolve); err != nil {
			return nil, "", fmt.Errorf("config %q: %w", c.Name, err)
		}
	}
	return srcData, dst, nil
}

// dirMode is the mode for a directory the deploy creates to hold a file of
// mode, for a Config that did not come from a manifest: private (0700) when
// the file grants nothing to group or others. A manifest entry uses the mode
// assignDirModes derived from every entry sharing the directory. A directory
// that exists is not created again; TightenDir narrows it afterwards.
func dirMode(mode os.FileMode) os.FileMode {
	if mode&0o077 == 0 {
		return 0o700
	}
	return 0o755
}

// stage writes data to a temp file beside the destination, with the declared
// mode. Beside it, not /tmp: an atomic rename requires the same filesystem,
// and a cross-device rename is the failure that turns an install into a
// half-written config.
func stage(c Config, dst string, data []byte, mode os.FileMode) (string, error) {
	dm := c.dirMode
	if dm == 0 {
		dm = dirMode(mode)
	}
	if err := os.MkdirAll(filepath.Dir(dst), dm); err != nil {
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
	if err := fsmode.Apply(staged, mode); err != nil {
		return staged, fmt.Errorf("config %q: stage mode: %w", c.Name, err)
	}
	return staged, nil
}

// isSymlink reports whether path is itself a link, without following it.
func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

// commit moves the staged file over the destination atomically.
func commit(c Config, staged, dst string, mode os.FileMode) error {
	if err := os.Rename(staged, dst); err != nil {
		return fmt.Errorf("config %q: install to %s: %w", c.Name, dst, err)
	}
	// Rename keeps the staged inode's bits, and stage() set them, but a
	// rendered config's staged copy has been through `secrets render` since,
	// which writes it anew: the bits on it are the renderer's, not the
	// manifest's. This is the call that makes the installed mode the declared
	// one on that path; a 0600 config must never end up 0644. On Windows
	// fsmode re-applies the owner-only DACL for the same reason (CLI-055).
	if err := fsmode.Apply(dst, mode); err != nil {
		return fmt.Errorf("config %q: mode on %s: %w", c.Name, dst, err)
	}
	return nil
}

// mergeFormat is how a merge reads and writes one file format. The merge itself
// (deepMerge) is the same for every format: a document is a map of keys, and
// the repo owns the keys its source names.
type mergeFormat struct {
	what         string // "a JSON object", in "source is not a JSON object"
	decodeSource func([]byte) (map[string]any, error)
	decodeDest   func([]byte) (map[string]any, error)
	encode       func(map[string]any) ([]byte, error)
}

// jsonMerge drops `//` header lines from the destination on read only:
// Copilot rewrites its config.json with a `// User settings belong in
// settings.json` header. The merged file is plain JSON, and the tool that wants
// a header puts it back.
var jsonMerge = mergeFormat{
	what:         "a JSON object",
	decodeSource: decodeJSONMap,
	decodeDest:   func(raw []byte) (map[string]any, error) { return decodeJSONMap(stripLineComments(raw)) },
	encode:       func(m map[string]any) ([]byte, error) { return encodeJSON(m) },
}

// tomlMerge is for a TOML config the tool also writes: herdr saves onboarding
// and its Settings screen into config.toml (#2013 H4). The merged file is
// re-encoded, so comments in the destination do not survive a merge that
// changes a value; an in-sync merge writes nothing and keeps them.
var tomlMerge = mergeFormat{
	what:         "a TOML table",
	decodeSource: decodeTOMLTable,
	decodeDest:   decodeTOMLTable,
	encode:       encodeTOML,
}

// mergeFormatFor chooses by the destination's extension: .toml is TOML, and
// anything else is JSON, the only format merge read before #2013 H4.
func mergeFormatFor(dst string) mergeFormat {
	if isTOML(dst) {
		return tomlMerge
	}
	return jsonMerge
}

func isTOML(p string) bool { return strings.EqualFold(path.Ext(filepath.ToSlash(p)), ".toml") }

// decodeJSONMap is decodeJSONObject for a merge: a JSON `null` unmarshals into
// a nil map without error, and assigning into it would panic, so it is refused
// like any other non-object.
func decodeJSONMap(raw []byte) (map[string]any, error) {
	m, err := decodeJSONObject(raw)
	if err == nil && m == nil {
		return nil, errors.New("null")
	}
	return m, err
}

// decodeTOMLTable reads a TOML document. An empty one is an empty table, which
// mergeInto refuses as a source.
func decodeTOMLTable(raw []byte) (map[string]any, error) {
	m := map[string]any{}
	if err := toml.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func encodeTOML(m map[string]any) ([]byte, error) { return toml.Marshal(m) }

// mergeInto returns the destination's document with the source's top-level
// keys written into it, and whether any of them differed. Equality is semantic
// (parsed values), never textual: a tool that rewrites its config in its own
// layout (a header, key order, inline tables) must not read as drift on every
// setup run.
//
// A missing destination merges into an empty document; a destination that does
// not parse as one is an error, because "merge" has no meaning for it and
// silently replacing it is the data loss this strategy exists to prevent.
// managesAValue reports whether m holds a value anywhere below it: anything
// that is not a table counts, an empty list included.
func managesAValue(m map[string]any) bool {
	for _, v := range m {
		sub, ok := v.(map[string]any)
		if !ok || managesAValue(sub) {
			return true
		}
	}
	return false
}

func mergeInto(dst string, srcData []byte) ([]byte, bool, error) {
	f := mergeFormatFor(dst)
	managed, err := f.decodeSource(srcData)
	if err != nil {
		return nil, false, fmt.Errorf("source is not %s: %w", f.what, err)
	}
	if !managesAValue(managed) {
		// The repo owns the values its source names, so a source that names
		// none owns nothing: every run would succeed and change nothing, and an
		// emptied config would read as in sync forever. A table with no values
		// (a TOML header left alone, `{"keys": {}}`) names none either.
		return nil, false, errors.New("source manages no key: a merge source must name at least one value")
	}
	existing := map[string]any{}
	if raw, err := os.ReadFile(dst); err == nil { //nolint:gosec // manifest-declared destination
		if existing, err = f.decodeDest(raw); err != nil {
			return nil, false, fmt.Errorf("destination is not %s: %w", f.what, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, false, err
	}
	merged, changed := deepMerge(existing, managed)
	content, err := f.encode(merged)
	if err != nil {
		return nil, false, err
	}
	return content, changed, nil
}

func deepMerge(existing, managed map[string]any) (map[string]any, bool) {
	changed := false
	for key, managedValue := range managed {
		existingValue, exists := existing[key]
		if !exists {
			existing[key] = managedValue
			changed = true
			continue
		}
		switch value := managedValue.(type) {
		case map[string]any:
			if object, ok := existingValue.(map[string]any); ok {
				merged, didChange := deepMerge(object, value)
				existing[key] = merged
				changed = changed || didChange
				continue
			}
		case []any:
			if list, ok := existingValue.([]any); ok {
				merged, didChange := unionLists(list, value)
				existing[key] = merged
				changed = changed || didChange
				continue
			}
		}
		if jsonEqual(existingValue, managedValue) {
			continue
		}
		existing[key] = managedValue
		changed = true
	}
	return existing, changed
}

func unionLists(existing, managed []any) ([]any, bool) {
	out := existing
	changed := false
	for _, managedValue := range managed {
		found := false
		for _, existingValue := range existing {
			if jsonEqual(existingValue, managedValue) {
				found = true
				break
			}
		}
		if !found {
			out = append(out, managedValue)
			changed = true
		}
	}
	return out, changed
}

func jsonEqual(a, b any) bool {
	switch value := a.(type) {
	case json.Number:
		other, ok := b.(json.Number)
		if !ok {
			return false
		}
		left, leftOK := new(big.Rat).SetString(value.String())
		right, rightOK := new(big.Rat).SetString(other.String())
		return leftOK && rightOK && left.Cmp(right) == 0
	case map[string]any:
		other, ok := b.(map[string]any)
		if !ok || len(value) != len(other) {
			return false
		}
		for key, child := range value {
			otherChild, exists := other[key]
			if !exists || !jsonEqual(child, otherChild) {
				return false
			}
		}
		return true
	case []any:
		other, ok := b.([]any)
		if !ok || len(value) != len(other) {
			return false
		}
		for index, child := range value {
			if !jsonEqual(child, other[index]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(a, b)
	}
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
