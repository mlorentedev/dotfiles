package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// BindOptions are the ambient inputs of one bind run: where the settings files
// live, which binary the hooks name, and what this machine has installed.
type BindOptions struct {
	Home string
	// Binary is the absolute dotf path, raw: each target renders its own token
	// because each harness owns its command parser.
	Binary string
	GOOS   string
	// Only binds a single agent; empty binds every declared target.
	Only   string
	DryRun bool
	// Has reports whether a command is on PATH, for requires_command. Required:
	// Bind refuses options without it.
	Has func(string) bool
}

// BindOutcome is what binding one target did, or under a dry run would do.
// Callers render it; nothing parses the text of a bind run.
type BindOutcome struct {
	Agent string
	File  string
	// Skip says why the target was not bound; empty when it was.
	Skip    string
	Changed bool
	Retired []RetiredHook
}

// Bind emits every selected target's hooks into its settings file and retires
// the hooks a target no longer emits. On an error it returns the outcomes of the
// targets before the failing one, and the failing one's too when it wrote
// before the error, so a caller can still report every change made.
//
// A target declaring `emit: false` is skipped visibly rather than forgotten, and
// one naming `requires_command` is skipped when that binary is absent: an
// uninstalled harness is not a failure.
func Bind(targets []BindTarget, o BindOptions) ([]BindOutcome, error) {
	if o.Has == nil {
		return nil, errors.New("harness.Bind: BindOptions.Has is required")
	}
	var out []BindOutcome
	for _, t := range targets {
		if o.Only != "" && t.Agent != o.Only {
			continue
		}
		res := BindOutcome{Agent: t.Agent, File: t.File}
		switch {
		case !t.Emits():
			res.Skip = fmt.Sprintf("declared emit:false (%s)", t.Format)
		case t.RequiresCommand != "" && !o.Has(t.RequiresCommand):
			res.Skip = t.RequiresCommand + " is not installed"
		default:
			token := HookBinaryTokenForTarget(o.Binary, o.GOOS, t.Format)
			changed, retired, err := bindOne(t, o.Home, token, o.DryRun)
			res.Changed, res.Retired = changed, retired
			if err != nil {
				// bindOne can fail after writing (a retirement that cannot be
				// read): that target's outcome is kept, since it changed a file.
				// One that failed before writing changed nothing, and an outcome
				// for it would read as "already current".
				if res.Changed || len(res.Retired) > 0 {
					out = append(out, res)
				}
				return out, fmt.Errorf("%s: %w", t.Agent, err)
			}
		}
		out = append(out, res)
	}
	return out, nil
}

// bindOne merges one target's hooks into its settings file, then retires the
// hooks the target says it no longer emits. It reports whether the file changed
// and which retirements took effect.
//
// The new home is written BEFORE the old entry is retired. The other order would
// leave a harness with no hook at all if the write failed halfway, and a gate that
// is briefly doubled is the smaller harm than one that is briefly absent.
func bindOne(t BindTarget, home, binary string, dryRun bool) (bool, []RetiredHook, error) {
	cmds, err := t.HookCommands(binary)
	if err != nil {
		return false, nil, err
	}
	path := filepath.Join(home, filepath.FromSlash(t.File))

	doc, err := readSettingsDoc(path)
	if err != nil {
		return false, nil, err
	}

	var (
		merged  map[string]any
		changed bool
	)
	// STRICT, with no default arm that falls back to claude's shape. That fallback
	// is exactly how a format this code did not know would be written into a file
	// of a different shape, and an unknown format is a refusal, not a guess.
	switch t.Format {
	case "command-hook", "":
		merged, changed, err = MergeHooks(doc, cmds)
	case NamedHooksFormat:
		merged, changed, err = MergeNamedHooks(doc, BindMarker, cmds)
	default:
		return false, nil, fmt.Errorf("unsupported bind format %q for %s", t.Format, t.Agent)
	}
	if err != nil {
		return false, nil, err
	}
	if changed && !dryRun {
		if err := writeSettingsAtomically(path, merged); err != nil {
			return false, nil, err
		}
	}

	var retired []RetiredHook
	for _, r := range t.Retire {
		did, err := retireOne(home, r, dryRun)
		if err != nil {
			return changed, retired, err
		}
		if did {
			retired = append(retired, r)
		}
	}
	return changed, retired, nil
}

// retireOne removes one retired hook from its file, reporting whether it was
// there. An absent file is nothing to retire and is NOT created: retiring must
// never bring a file into being.
func retireOne(home string, r RetiredHook, dryRun bool) (bool, error) {
	path := filepath.Join(home, filepath.FromSlash(r.File))
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", path, err)
	}
	doc, err := readSettingsDoc(path)
	if err != nil {
		return false, err
	}
	pruned, changed := RetireHooks(doc, r.Event, r.ID)
	if !changed || dryRun {
		return changed, nil
	}
	if err := writeSettingsAtomically(path, pruned); err != nil {
		return false, err
	}
	return true, nil
}

// readSettingsDoc decodes a settings file. An absent or blank file is an empty
// document: there is nothing to preserve, so bootstrapping is fine.
func readSettingsDoc(path string) (map[string]any, error) {
	doc := map[string]any{}
	raw, readErr := os.ReadFile(path) // #nosec G304 -- path comes from the manifest's own declaration
	switch {
	case readErr == nil:
		if len(strings.TrimSpace(string(raw))) > 0 {
			if err := json.Unmarshal(raw, &doc); err != nil {
				// Refuse rather than bootstrap over it. A settings file that is
				// present but unparseable is a file someone is editing, and
				// replacing it loses their work - the opposite of the merge's
				// whole purpose.
				return nil, fmt.Errorf("%s is not valid JSON, refusing to overwrite it: %w", path, err)
			}
		}
	case os.IsNotExist(readErr):
		// Bootstrapping an absent file is fine: there is nothing to preserve.
	default:
		return nil, fmt.Errorf("read %s: %w", path, readErr)
	}
	return doc, nil
}

// writeSettingsAtomically renders doc and replaces path with it in one step.
//
// Temp + rename IN THE SAME DIRECTORY: a rename across filesystems is not
// atomic, and a half-written settings file is a harness that will not start.
func writeSettingsAtomically(path string, doc map[string]any) error {
	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	encoded = append(encoded, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".settings-*.json")
	if err != nil {
		return fmt.Errorf("create temp beside %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(encoded); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", tmpName, err)
	}
	// An existing file keeps its mode: a file we share with another tool (agy's
	// hooks.json is written by Orca too) is not ours to re-permission. A new one
	// starts private.
	mode := os.FileMode(0o600)
	if fi, statErr := os.Stat(path); statErr == nil {
		mode = fi.Mode().Perm()
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return fmt.Errorf("chmod %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename onto %s: %w", path, err)
	}
	return nil
}

// ResolveDotfPath picks the absolute binary path emitted into hook commands.
//
// It prefers ~/.local/bin/dotf — the path the installer writes and the one the
// session hooks already carried — over whatever happens to be on the PATH of the
// process running setup. Those differ exactly when it matters: a `go run` or a
// build-tree binary would otherwise be baked into a hook that outlives it.
//
// The `.exe` suffix is not cosmetic. setup-windows.ps1 emitted
// `"…\dotf.exe" mem session-start`, and an unmarked entry is adopted by its
// command signature (isOurs). A suffix-less path would still be adopted,
// but it would differ from the deployed line, so every Windows run would rewrite
// a hook that was already correct.
func ResolveDotfPath(home string) string {
	name := "dotf"
	if runtime.GOOS == "windows" {
		name = "dotf.exe"
	}
	installed := filepath.Join(home, ".local", "bin", name)
	if _, err := os.Stat(installed); err == nil {
		return installed
	}
	if p, err := exec.LookPath("dotf"); err == nil {
		abs, err := filepath.Abs(p)
		if err == nil {
			return abs
		}
		return p
	}
	return installed
}

// HookBinaryToken renders the binary path as it appears inside a hook command
// line, quoting it where the shell that runs the hook would otherwise split it.
//
// goos is a parameter rather than a read of runtime.GOOS so both branches are
// testable from either OS — the Windows leg of this behaviour cannot be
// exercised on the machine that develops it otherwise.
//
// Windows command-hook targets are quoted unconditionally, matching byte-for-byte
// what setup-windows.ps1 already deployed, because anything else fails to adopt
// that entry and duplicates it. Elsewhere the path is bare unless it contains a
// space, where quoting is the only correct shell rendering.
func HookBinaryToken(path, goos string) string {
	if goos == "windows" || strings.ContainsAny(path, " \t") {
		return `"` + path + `"`
	}
	return path
}

func HookBinaryTokenForTarget(path, goos, format string) string {
	// agy's Windows hook runner treats quotes around the first token as literal
	// executable-name characters. Its default dotf installation path has no spaces.
	if goos == "windows" && format == NamedHooksFormat &&
		!strings.ContainsAny(path, " \t") {
		return path
	}
	return HookBinaryToken(path, goos)
}
