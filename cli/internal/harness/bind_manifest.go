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
	// Format is the emission kind. Only "command-hook" is emitted today;
	// "ts-extension" needs a generated-code template and is declared with
	// emit:false so the gap is visible rather than remembered.
	Format string `json:"format"`
	// Matcher records whether this harness's groups carry a `matcher` key.
	// Claude's do; agy's — measured against ~/.gemini/settings.json — do not.
	Matcher bool `json:"matcher"`
	// Emit absent means true. A pointer distinguishes "not declared" from
	// "declared false", so a target that forgets the key emits rather than
	// silently doing nothing.
	Emit *bool `json:"emit"`
	// RequiresCommand names a binary that must be on PATH; absent means always.
	RequiresCommand string            `json:"requires_command"`
	Events          map[string]string `json:"events"`
	EmitHooks       []EmitHook        `json:"emit_hooks"`
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

// LoadBindTargets reads `agents.bind` from the manifest.
//
// An absent or empty `bind` is an ERROR rather than an empty slice, on C15: a
// caller that asked for the bind targets and got none cannot tell "this repo
// declares no binding" from "the manifest moved and nobody noticed", and the
// two would produce the same silence — a setup run that writes no hooks and
// reports success.
func LoadBindTargets(root string) ([]BindTarget, error) {
	path := filepath.Join(root, filepath.FromSlash(ManifestFile))
	raw, err := os.ReadFile(path) // #nosec G304 -- path is the manifest under the caller's root
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", ManifestFile, err)
	}
	var doc struct {
		Agents struct {
			Bind []BindTarget `json:"bind"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", ManifestFile, err)
	}
	if len(doc.Agents.Bind) == 0 {
		return nil, fmt.Errorf("%s declares no agents.bind targets", ManifestFile)
	}
	return doc.Agents.Bind, nil
}
