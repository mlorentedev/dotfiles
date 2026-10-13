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

// MCPReport is what one Register found and did.
type MCPReport struct {
	Present, Added, Failed []string
	// Migrated are the drifted registrations replaced (or, in a dry run, that
	// would be). Each is in Added too, once its re-add landed.
	Migrated []Drift
	// Unverified are registered servers whose `claude mcp get` output did not
	// parse, so whether they drifted is unknown. They are left alone: treating
	// an unreadable answer as drift would re-register every server on every
	// run the day Claude Code changes its output.
	Unverified []string
	Skipped    []Skip
	Restored   int
}

// registered reads what `claude mcp get` says a server runs, as
// "<transport> <argv>" for stdio (Command and Args joined on single spaces,
// the way registration split them) and "<transport> <URL>" otherwise. false
// when the output names no transport, or no command or URL to go with it.
func registered(out string) (string, bool) {
	fields := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if ok {
			fields[key] = strings.TrimSpace(value)
		}
	}
	transport, target := fields["Type"], fields["URL"]
	if transport == "stdio" {
		target = strings.Join(strings.Fields(fields["Command"]+" "+fields["Args"]), " ")
	}
	if transport == "" || target == "" {
		return "", false
	}
	return transport + " " + target, true
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
		if getErr == nil {
			was, ok := registered(out)
			switch {
			case !ok:
				rep.Unverified = append(rep.Unverified, srv.Name)
				continue
			case was == declared(srv):
				rep.Present = append(rep.Present, srv.Name)
				continue
			}
			drift = &Drift{Name: srv.Name, Was: was, Now: declared(srv)}
			if !dryRun {
				if err := guard(func() error { return s.Run.McpRemove(srv.Name) }); err != nil {
					rep.Failed = append(rep.Failed, srv.Name)
					continue
				}
			}
		}
		// A migration is reported only once the replacement is in: a remove
		// whose re-add failed left the server unregistered, and says so in Failed.
		if !dryRun {
			if err := guard(func() error { return s.Run.McpAdd(srv.Name, srv.Transport, strings.Fields(srv.Args)) }); err != nil {
				rep.Failed = append(rep.Failed, srv.Name)
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
