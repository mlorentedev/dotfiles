package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// EmitHook is one hook a bind target emits: which event, running what.
//
// `Command` is a `dotf` SUBCOMMAND SUFFIX, never a full command line, and the
// split is deliberate. The manifest owns WHERE a hook goes — the event names
// were measured against each installed harness rather than taken from ADR-027,
// whose render kinds were already stale in 2 of 5. The binary's absolute path is
// resolved at emit time, because it is a property of the machine being set up
// and not of the declaration: a checked-in absolute path is the ADR-025
// violation this repo has a lint for.
type EmitHook struct {
	// ID names this hook's purpose. It is the identity MergeHooks matches on,
	// so two hooks sharing an event must not share an ID or the second evicts
	// the first — a defect found by test, recorded on HookCommand.ID.
	ID string `json:"id"`
	// Event is the harness's native event name.
	Event string `json:"event"`
	// Command is the dotf subcommand and its flags, e.g. "mem session-start".
	Command string `json:"command"`
	// Timeout in seconds; omitted from the emitted hook when zero.
	Timeout int `json:"timeout"`
}

// BindTarget is one `agents.bind` entry: a harness, the settings file it reads,
// and the hooks this repository owns inside it.
type BindTarget struct {
	Agent string `json:"agent"`
	File  string `json:"file"`
	// Format is the emission kind. "command-hook" is claude's shape (a `hooks`
	// key of events); "hooks-json" is a document of NAMED hooks, agy's; and
	// "ts-extension" needs a generated-code template and is declared with
	// emit:false so the gap is visible rather than remembered.
	Format string `json:"format"`
	// Matcher records whether this harness's groups carry a `matcher` key.
	// Claude's do, and so do agy's hooks.json groups for the tool events.
	Matcher bool `json:"matcher"`
	// Emit absent means true. A pointer distinguishes "not declared" from
	// "declared false", so a target that forgets the key emits rather than
	// silently doing nothing.
	Emit *bool `json:"emit"`
	// RequiresCommand names a binary that must be on PATH; absent means always.
	RequiresCommand string            `json:"requires_command"`
	Events          map[string]string `json:"events"`
	EmitHooks       []EmitHook        `json:"emit_hooks"`
	// Retire lists hooks this repository USED to emit and no longer does, so a
	// move to another file does not leave the old entry behind. Each is removed
	// from its file by marker, never by position.
	Retire []RetiredHook `json:"retire"`
}

// RetiredHook names one hook to remove: the settings file it sat in, the event,
// and the ID it was emitted under.
type RetiredHook struct {
	File  string `json:"file"`
	Event string `json:"event"`
	ID    string `json:"id"`
}

// Emits reports whether this target is emitted at all.
func (t BindTarget) Emits() bool { return t.Emit == nil || *t.Emit }

// HookCommands renders this target's declaration into what MergeHooks folds in.
//
// dotfPath is the absolute path to the binary. It is absolute for the same
// reason the session hooks already were (#531): a harness runs a hook with the
// profile PATH it happened to inherit, and ~/.local/bin is not always on it. A
// hook that cannot find its binary fails in a way nothing reports.
func (t BindTarget) HookCommands(dotfPath string) ([]HookCommand, error) {
	if dotfPath == "" {
		return nil, fmt.Errorf("bind target %q: no dotf path to emit", t.Agent)
	}
	out := make([]HookCommand, 0, len(t.EmitHooks))
	for i, h := range t.EmitHooks {
		if h.ID == "" || h.Event == "" || h.Command == "" {
			return nil, fmt.Errorf("bind target %q: emit_hooks[%d] needs id, event and command", t.Agent, i)
		}
		out = append(out, HookCommand{
			Event:      h.Event,
			Command:    dotfPath + " " + h.Command,
			UseMatcher: t.Matcher,
			ID:         h.ID,
			Timeout:    h.Timeout,
		})
	}
	return out, nil
}

// LoadBindTargets reads `agents.bind` and `agents.bind_named` from the manifest.
//
// THE TWO KEYS EXIST BECAUSE THE MANIFEST AND THE BINARY SHIP SEPARATELY. Setup
// mirrors the manifest the moment it merges; a released binary arrives on its own
// schedule. A binary from before a format existed reads `bind`, ignores every key
// it does not know, and treats each target in `bind` as claude's shape. A target
// in a new format placed there is therefore emitted WRONG by every older binary
// rather than skipped: the agy target, in `bind`, would have made dotf 0.57.0 write
// a top-level `hooks` key with a `_managed` sidecar into agy's hooks.json (a file
// agy decodes as protojson and shares with Orca). Measured 2026-09-24 against a
// copy of the real file.
//
// So a format an older binary does not know lives under `bind_named`, which such a
// binary never reads, and its effect is nothing, the status quo. This is enforced
// here, where the manifest loads, so a target cannot drift into the wrong key.
//
// An absent or empty `bind` is an ERROR rather than an empty slice, on C15: a
// caller that asked for the bind targets and got none cannot tell "this repo
// declares no binding" from "the manifest moved and nobody noticed", and the
// two would produce the same silence - a setup run that writes no hooks and
// reports success.
func LoadBindTargets(root string) ([]BindTarget, error) {
	path := filepath.Join(root, filepath.FromSlash(ManifestFile))
	raw, err := os.ReadFile(path) // #nosec G304 -- path is the manifest under the caller's root
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", ManifestFile, err)
	}
	var doc struct {
		Agents struct {
			Bind      []BindTarget `json:"bind"`
			BindNamed []BindTarget `json:"bind_named"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", ManifestFile, err)
	}
	if len(doc.Agents.Bind) == 0 {
		return nil, fmt.Errorf("%s declares no agents.bind targets", ManifestFile)
	}
	for _, t := range doc.Agents.Bind {
		if t.Format == NamedHooksFormat {
			return nil, fmt.Errorf("%s: agents.bind target %q uses format %q, which older binaries do not know; "+
				"declare it under agents.bind_named, which they never read", ManifestFile, t.Agent, t.Format)
		}
	}
	for _, t := range doc.Agents.BindNamed {
		if t.Format != NamedHooksFormat {
			return nil, fmt.Errorf("%s: agents.bind_named target %q has format %q, want %q",
				ManifestFile, t.Agent, t.Format, NamedHooksFormat)
		}
	}
	return append(doc.Agents.Bind, doc.Agents.BindNamed...), nil
}
