package tools

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

// shadowFix is what an operator does about a pin that another config holds.
const shadowFix = "remove the pin there, or let that file's owner stop pinning a tool versions.conf owns"

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
// resolves them from a config other than the one the sync owns: cause names
// each such file and the version it pins, fix says what to do. Both are empty
// when no tool is shadowed, and also when mise cannot say: a failed or
// unreadable answer never claims a cause, so the caller keeps its plain
// message. Only the tools asked about are looked up, so a plan with nothing
// missing asks mise nothing (#3524).
func (s MiseSync) ShadowNote(missing []string) (cause, fix string) {
	run := s.Stdout
	if run == nil {
		run = s.Run
	}
	var found []shadow
	for _, name := range missing {
		if sh, ok := s.shadowOf(run, name); ok {
			found = append(found, sh)
		}
	}
	if len(found) == 0 {
		return "", ""
	}
	var files []string
	pins := map[string][]string{}
	for _, sh := range found {
		if _, seen := pins[sh.path]; !seen {
			files = append(files, sh.path)
		}
		pins[sh.path] = append(pins[sh.path], sh.tool+" "+sh.version)
	}
	parts := make([]string, len(files))
	for i, path := range files {
		parts[i] = path + " pins " + strings.Join(pins[path], ", ") + " and outranks " + s.ConfigPath()
	}
	return strings.Join(parts, "; "), shadowFix
}

// shadowOf asks mise which config the named tool runs from. It reports no
// shadow when the answer is an error, is not the array above, has no active
// entry, or names the sync's own file.
func (s MiseSync) shadowOf(run Runner, name string) (shadow, bool) {
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
func (s MiseSync) shadowSuffix(missing []string) string {
	cause, fix := s.ShadowNote(missing)
	if cause == "" {
		return ""
	}
	return "; " + cause + ": " + fix
}
