package tools

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// ShadowFix is what an operator does about a pin that another config holds.
const ShadowFix = "remove the pin there, or let that file's owner stop pinning a tool versions.conf owns"

// lsEntry is the part of one `mise ls --json <tool>` entry the shadow
// check reads. mise prints an array with one entry per installed version; the
// one it runs is active, and its source names the config that requested it
// (measured on mise 2026.11.6).
type lsEntry struct {
	Version          string `json:"version"`
	RequestedVersion string `json:"requested_version"`
	Active           bool   `json:"active"`
	Source           struct {
		Path string `json:"path"`
	} `json:"source"`
}

// shadow is a tool whose pin another mise config holds: mise resolves the tool
// from Path, at Version, not from the file the sync owns.
type shadow struct {
	tool, version, path string
}

// ShadowNote says why the named tools do not run at their pin when mise
// resolves them from a config other than the one the sync owns, at a version
// below the pin: cause names each such file and the version it pins, fix says
// what to do. Both are empty when no tool is shadowed, and also when mise
// cannot say: a failed or unreadable answer never claims a cause, and neither
// does a foreign file that resolves the pinned version or a newer one, so the
// caller keeps its plain message. Only the tools asked about are looked up, so
// a plan with nothing missing asks mise nothing (#3524).
//
// The caller asks only once the sync's own config is current: before that,
// mise has not been told the pins, and the file that runs the tool may rank
// below it.
func (s MiseSync) ShadowNote(pins []MiseTool, missing []string) (cause, fix string) {
	run := s.Stdout
	if run == nil {
		run = s.Run
	}
	var found []shadow
	for _, name := range missing {
		if sh, ok := s.shadowOf(run, pinOf(pins, name)); ok {
			found = append(found, sh)
		}
	}
	if len(found) == 0 {
		return "", ""
	}
	var files []string
	byFile := map[string][]string{}
	for _, sh := range found {
		if _, seen := byFile[sh.path]; !seen {
			files = append(files, sh.path)
		}
		byFile[sh.path] = append(byFile[sh.path], sh.tool+" "+sh.version)
	}
	parts := make([]string, len(files))
	for i, path := range files {
		parts[i] = path + " pins " + strings.Join(byFile[path], ", ") + " and outranks " + s.ConfigPath()
	}
	return strings.Join(parts, "; "), ShadowFix
}

// pinOf is the pin for name, or a pin with no version when none is declared.
func pinOf(pins []MiseTool, name string) MiseTool {
	for _, p := range pins {
		if p.Name == name {
			return p
		}
	}
	return MiseTool{Name: name}
}

// shadowOf asks mise which config the pinned tool runs from. It reports no
// shadow when the answer is an error, is not the array above, has no active
// entry, names the sync's own file, or resolves a version at or above the pin:
// such a file did not make the tool fail.
func (s MiseSync) shadowOf(run Runner, pin MiseTool) (shadow, bool) {
	name := pin.Name
	out, err := run("mise", "ls", "--json", name)
	if err != nil {
		return shadow{}, false
	}
	var entries []lsEntry
	if json.Unmarshal(out, &entries) != nil {
		return shadow{}, false
	}
	for _, e := range entries {
		if !e.Active || e.Source.Path == "" {
			continue
		}
		if samePath(e.Source.Path, s.ConfigPath()) {
			return shadow{}, false
		}
		if e.Version == "" || atLeast(e.Version, pin.Version) {
			return shadow{}, false
		}
		version := e.RequestedVersion
		if version == "" {
			version = e.Version
		}
		return shadow{tool: name, version: version, path: e.Source.Path}, true
	}
	return shadow{}, false
}

// samePath reports whether a and b name one file, comparing the physical
// paths when both exist: macOS reaches the same directory through a symlinked
// prefix (/var is /private/var), and a logical comparison would call dotfiles'
// own file a foreign one.
func samePath(a, b string) bool {
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	ra, erra := filepath.EvalSymlinks(a)
	rb, errb := filepath.EvalSymlinks(b)
	return erra == nil && errb == nil && ra == rb
}

// shadowSuffix is ShadowNote as the tail of the sync's failure, or "" when no
// cause is known.
func (s MiseSync) shadowSuffix(pins []MiseTool, missing []string) string {
	cause, fix := s.ShadowNote(pins, missing)
	if cause == "" {
		return ""
	}
	return "; " + cause + ": " + fix
}

// SourceDrift is a pinned tool that mise runs from a config other than the one
// the sync owns: Tool at Version, requested by the file at Path, where
// versions.conf says Pin.
type SourceDrift struct {
	Tool, Version, Pin, Path string
}

// SourceDrift lists every pin whose active entry comes from a foreign config,
// whichever way its version differs from the pin: a file that pins a newer
// version passes the sync's post-condition and ShadowNote never looks at it,
// so a go 1.27.4 and a golangci-lint 2.15.0 that no reviewed commit chose ran
// unseen (#3532). One `mise ls --current --json` answers for every pin. The
// caller runs it from the home directory, so a project's own mise.toml is not
// reported as drift. An error means mise could not say, which is not the same
// as "no drift"; a tool with no active entry or no source path is not claimed.
func (s MiseSync) SourceDrift(pins []MiseTool) ([]SourceDrift, error) {
	run := s.Stdout
	if run == nil {
		run = s.Run
	}
	out, err := run("mise", "ls", "--current", "--json")
	if err != nil {
		return nil, fmt.Errorf("mise ls --current --json: %w", err)
	}
	var byTool map[string][]lsEntry
	if err := json.Unmarshal(out, &byTool); err != nil {
		return nil, fmt.Errorf("mise ls --current --json: an answer this does not read: %w", err)
	}
	var drifts []SourceDrift
	for _, pin := range pins {
		for _, key := range lsKeys(pin.Name) {
			e, ok := activeEntry(byTool[key])
			if !ok {
				continue
			}
			if !samePath(e.Source.Path, s.ConfigPath()) {
				drifts = append(drifts, SourceDrift{Tool: pin.Name, Version: e.Version, Pin: pin.Version, Path: e.Source.Path})
			}
			break
		}
	}
	return drifts, nil
}

// activeEntry is the entry mise runs, when it names the config that asked for it.
func activeEntry(entries []lsEntry) (lsEntry, bool) {
	for _, e := range entries {
		if e.Active && e.Source.Path != "" {
			return e, true
		}
	}
	return lsEntry{}, false
}

// lsKeys are the names mise may list a pinned tool under: its short name, and
// the key the sync writes when that is not the short name (a Windows bats is
// npm:bats, a Windows direnv is github:direnv/direnv, an npm CLI is aliased).
func lsKeys(name string) []string {
	keys := []string{name}
	switch name {
	case "bats":
		keys = append(keys, "npm:bats")
	case "direnv":
		keys = append(keys, "github:direnv/direnv")
	}
	if backend, ok := aliasBackend(name); ok {
		keys = append(keys, backend)
	}
	return keys
}
