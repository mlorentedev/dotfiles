package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ServersRel is the MCP server list, relative to the repo root. Both setup
// twins read the same file; opencode and agy mirror it.
const ServersRel = "mcp-servers.json"

// Server is one `claude mcp add` registration. Args is the argv after `--`,
// split on whitespace with no quoting, exactly as both twins split it.
//
// PrerequisiteCommand is read so the strict decoder accepts the file, and is
// deliberately never run: it installs a tool, which is not a registration and
// is not idempotent under --upgrade (#1993).
type Server struct {
	Name                string `json:"name"`
	Transport           string `json:"transport"`
	Args                string `json:"args"`
	PrerequisiteBinary  string `json:"prerequisite_binary"`
	PrerequisiteCommand string `json:"prerequisite_command"`
}

// LoadServers reads and validates the server list, strictly.
func LoadServers(path string) ([]Server, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // repo-relative, fixed name
	if err != nil {
		return nil, err
	}
	var doc struct {
		Comment string   `json:"_comment"`
		History string   `json:"_history"`
		Servers []Server `json:"servers"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", ServersRel, err)
	}
	seen := map[string]bool{}
	for i, s := range doc.Servers {
		switch {
		case s.Name == "":
			return nil, fmt.Errorf("%s: server #%d: empty name", ServersRel, i)
		case seen[s.Name]:
			return nil, fmt.Errorf("%s: duplicate server %q", ServersRel, s.Name)
		case s.Transport != "stdio" && s.Transport != "http":
			return nil, fmt.Errorf("%s: server %q: transport %q (want stdio or http)", ServersRel, s.Name, s.Transport)
		case len(strings.Fields(s.Args)) == 0:
			return nil, fmt.Errorf("%s: server %q: empty args", ServersRel, s.Name)
		}
		seen[s.Name] = true
	}
	return doc.Servers, nil
}

// Skip is a server left unregistered because its prerequisite binary is not
// on PATH.
type Skip struct{ Name, Missing string }

// Drift is a server registered with something other than what
// mcp-servers.json declares: Was is what `claude mcp get` reported, Now the
// declaration, both as "<transport> <argv or URL>".
type Drift struct{ Name, Was, Now string }

// Untouched is a registered server Register left as it is, and why.
type Untouched struct{ Name, Why string }

// MCPReport is what one Register found and did.
type MCPReport struct {
	Present, Added, Failed []string
	// Migrated are the drifted registrations replaced (or, in a dry run, that
	// would be). Each is in Added too, once its re-add landed.
	Migrated []Drift
	// RolledBack are the drifted servers whose re-add failed after the old
	// registration was removed, and which got the old one back. Each is in
	// Failed too.
	RolledBack []string
	// Untouched are registered servers Register cannot judge or does not own:
	// a `claude mcp get` answer that does not parse (treating it as drift
	// would re-register every server the day Claude Code changes its output),
	// or a drifted registration in a local or project scope, which shadows
	// the user scope dotf registers in.
	Untouched []Untouched
	Skipped   []Skip
	Restored  int
}

// registration is what `claude mcp get` says a server runs: its transport,
// the scope that answered, and argv, the command and its args for stdio or
// the URL alone otherwise, so it can be handed back to McpAdd.
type registration struct {
	transport, scope string
	argv             []string
}

func (r registration) String() string { return r.transport + " " + strings.Join(r.argv, " ") }

// userScope reports whether the answering registration is the user-scope one
// dotf adds and removes. `claude mcp get` answers with the winning scope
// (local, then project, then user), worded "User config (...)".
func (r registration) userScope() bool { return strings.HasPrefix(r.scope, "User") }

// registered reads `claude mcp get`'s output: `Scope:` and `Type:`, then
// `Command:` and `Args:` (joined on single spaces, the way registration split
// them) for stdio or `URL:` otherwise. false when any of them is missing.
func registered(out string) (registration, bool) {
	fields := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if ok {
			fields[key] = strings.TrimSpace(value)
		}
	}
	r := registration{transport: fields["Type"], scope: fields["Scope"], argv: strings.Fields(fields["URL"])}
	if r.transport == "stdio" {
		r.argv = strings.Fields(fields["Command"] + " " + fields["Args"])
	}
	return r, r.transport != "" && r.scope != "" && len(r.argv) > 0
}

// declared is a Server in the shape registered reads.
func declared(srv Server) string {
	return srv.Transport + " " + strings.Join(strings.Fields(srv.Args), " ")
}

// Register adds every server `claude mcp get` does not know, and replaces one
// registered with another command, arguments or URL (BUG-128, #2397; it was a
// hive-only regex for HIVE-118's `uvx hive-vault`), each claude call inside the
// snapshot guard. A get that exits non-zero means "not registered", as in both
// twins; a CLI that is broken outright then fails on the add, where it is
// reported. An add or remove that fails is named in Failed and never counted,
// so unlike Sync there is no error a caller could act on.
func (s Syncer) Register(servers []Server, dryRun bool) MCPReport {
	var rep MCPReport
	guard := func(fn func() error) error {
		restored, err := Guard(s.ClaudeJSON, s.Floor, fn)
		rep.Restored += b2i(restored)
		return err
	}
	for _, srv := range servers {
		if srv.PrerequisiteBinary != "" && !s.HasCommand(srv.PrerequisiteBinary) {
			rep.Skipped = append(rep.Skipped, Skip{srv.Name, srv.PrerequisiteBinary})
			continue
		}
		var out string
		getErr := guard(func() error { var err error; out, err = s.Run.McpGet(srv.Name); return err })
		var drift *Drift
		var was registration
		if getErr == nil {
			var ok bool
			was, ok = registered(out)
			switch {
			case !ok:
				rep.Untouched = append(rep.Untouched, Untouched{srv.Name, "`claude mcp get` output did not parse"})
				continue
			case was.String() == declared(srv):
				rep.Present = append(rep.Present, srv.Name)
				continue
			case !was.userScope():
				rep.Untouched = append(rep.Untouched, Untouched{srv.Name, fmt.Sprintf("registered as %s in %q, not the user scope dotf owns", was, was.scope)})
				continue
			}
			drift = &Drift{Name: srv.Name, Was: was.String(), Now: declared(srv)}
			if !dryRun {
				if err := guard(func() error { return s.Run.McpRemove(srv.Name) }); err != nil {
					rep.Failed = append(rep.Failed, srv.Name)
					continue
				}
			}
		}
		// A migration is reported only once the replacement is in. A re-add
		// that fails puts the old registration back rather than leave the
		// server unregistered, and is still Failed: the drift remains.
		if !dryRun {
			if err := guard(func() error { return s.Run.McpAdd(srv.Name, srv.Transport, strings.Fields(srv.Args)) }); err != nil {
				rep.Failed = append(rep.Failed, srv.Name)
				if drift != nil && guard(func() error { return s.Run.McpAdd(srv.Name, was.transport, was.argv) }) == nil {
					rep.RolledBack = append(rep.RolledBack, srv.Name)
				}
				continue
			}
		}
		rep.Added = append(rep.Added, srv.Name)
		if drift != nil {
			rep.Migrated = append(rep.Migrated, *drift)
		}
	}
	return rep
}
