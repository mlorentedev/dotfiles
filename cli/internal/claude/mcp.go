package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
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

// MCPReport is what one Register found and did.
type MCPReport struct {
	Present, Added, Failed []string
	Migrated               []string // stale entries removed so the loop re-adds them
	Skipped                []Skip
	Restored               int
}

// staleHive matches a `claude mcp get hive` that still runs the per-session
// `uvx hive-vault` server HIVE-118 replaced with the `hive client` proxy.
var staleHive = regexp.MustCompile(`uvx|hive-vault`)

// Register adds every server `claude mcp get` does not know, each claude call
// inside the snapshot guard. A get that exits non-zero means "not registered",
// as in both twins; a CLI that is broken outright then fails on the add, where
// it is reported. An add that fails is named in Failed and never counted.
func (s Syncer) Register(servers []Server, dryRun bool) (MCPReport, error) {
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
		registered := getErr == nil
		if registered && srv.Name == "hive" && staleHive.MatchString(out) {
			rep.Migrated = append(rep.Migrated, srv.Name)
			if !dryRun {
				if err := guard(func() error { return s.Run.McpRemove(srv.Name) }); err != nil {
					rep.Failed = append(rep.Failed, srv.Name)
					continue
				}
			}
			registered = false
		}
		switch {
		case registered:
			rep.Present = append(rep.Present, srv.Name)
		case dryRun:
			rep.Added = append(rep.Added, srv.Name)
		case guard(func() error { return s.Run.McpAdd(srv.Name, srv.Transport, strings.Fields(srv.Args)) }) != nil:
			rep.Failed = append(rep.Failed, srv.Name)
		default:
			rep.Added = append(rep.Added, srv.Name)
		}
	}
	return rep, nil
}
