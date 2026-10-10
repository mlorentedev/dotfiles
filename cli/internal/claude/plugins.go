package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
)

// PluginsRel is the plugin list, relative to the repo root.
const PluginsRel = "ai/claude/plugins.json"

// PluginList is ai/claude/plugins.json: the plugins a box must carry, and the
// marketplaces it must not (retired ones whose registration outlives the
// plugin, so Claude Code keeps re-cloning them).
type PluginList struct {
	Plugins             []string
	RetiredMarketplaces []string
}

// LoadPlugins reads the plugin list. Strict for the same reason the deploy
// manifest is: a file this binary cannot fully read is one it must not act on.
func LoadPlugins(path string) (PluginList, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // repo-relative, fixed name
	if err != nil {
		return PluginList{}, err
	}
	var doc struct {
		Comment             []string `json:"$comment"`
		Plugins             []string `json:"plugins"`
		RetiredMarketplaces []string `json:"retired_marketplaces"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return PluginList{}, fmt.Errorf("parse %s: %w", PluginsRel, err)
	}
	seen := map[string]bool{}
	used := map[string]bool{}
	for _, id := range doc.Plugins {
		name, market, ok := strings.Cut(id, "@")
		if !ok || name == "" || market == "" || strings.Contains(market, "@") {
			return PluginList{}, fmt.Errorf("%s: %q is not <plugin>@<marketplace>", PluginsRel, id)
		}
		if seen[id] {
			return PluginList{}, fmt.Errorf("%s: duplicate plugin %q", PluginsRel, id)
		}
		seen[id] = true
		used[market] = true
	}
	for _, m := range doc.RetiredMarketplaces {
		if m == "" || strings.Contains(m, "@") {
			return PluginList{}, fmt.Errorf("%s: retired marketplace %q is not a marketplace name", PluginsRel, m)
		}
		if used[m] {
			// Removing it would take a declared plugin with it on every run.
			return PluginList{}, fmt.Errorf("%s: marketplace %q is both retired and the source of a declared plugin", PluginsRel, m)
		}
	}
	return PluginList{Plugins: doc.Plugins, RetiredMarketplaces: doc.RetiredMarketplaces}, nil
}

// Runner is the claude CLI as the deploy uses it: the seam a test fills with a
// fake instead of a real install.
type Runner interface {
	List() (string, error)                              // `claude plugin list` output
	Install(id string) error                            // `claude plugin install <id>`
	McpGet(name string) (string, error)                 // `claude mcp get <name>`
	McpAdd(name, transport string, args []string) error // `claude mcp add --transport <t> <name> --scope user -- <args>`
	McpRemove(name string) error                        // `claude mcp remove <name> --scope user`
	Marketplaces() (string, error)                      // `claude plugin marketplace list --json` output
	RemoveMarketplace(name string) error                // `claude plugin marketplace remove <name>`
}

// Syncer installs the declared plugins a box lacks, each claude call inside
// the snapshot guard on ClaudeJSON.
type Syncer struct {
	Run        Runner
	ClaudeJSON string
	Floor      int
	HasCommand func(string) bool // PATH lookup for an MCP server's prerequisite binary
}

// PluginReport is what one Sync found and did. Added holds only installs that
// succeeded, or would be attempted on a dry run; a failed install is in Failed
// and nowhere else (AC7; the PowerShell twin counts attempts, #1491).
type PluginReport struct {
	Present, Added, Failed []string
	Restored               int // .claude.json restores the guard made
}

// Sync installs every id in ids that `claude plugin list` does not name.
// A plugin counts as present when its id is a whole whitespace-separated token
// of the list output (the CLI prints one `<glyph> <id>` line per plugin). The
// twins grep for a substring, under which a declared `lsp@m` reads as present
// on a box that only has `gopls-lsp@m`. A list that fails is returned, not read
// as empty: treating a broken CLI as "nothing installed" reinstalls everything.
func (s Syncer) Sync(ids []string, dryRun bool) (PluginReport, error) {
	var rep PluginReport
	var listed string
	restored, err := Guard(s.ClaudeJSON, s.Floor, func() error {
		var err error
		listed, err = s.Run.List()
		return err
	})
	rep.Restored += b2i(restored)
	if err != nil {
		return rep, fmt.Errorf("claude plugin list: %w", err)
	}
	installed := map[string]bool{}
	for _, tok := range strings.Fields(listed) {
		installed[tok] = true
	}
	for _, id := range ids {
		switch {
		case installed[id]:
			rep.Present = append(rep.Present, id)
		case dryRun:
			rep.Added = append(rep.Added, id)
		default:
			restored, err := Guard(s.ClaudeJSON, s.Floor, func() error { return s.Run.Install(id) })
			rep.Restored += b2i(restored)
			if err != nil {
				rep.Failed = append(rep.Failed, id)
				continue
			}
			rep.Added = append(rep.Added, id)
		}
	}
	return rep, nil
}

// RetireReport is what one Retire found and did. Removed holds marketplaces
// whose registration is gone afterwards (or would be removed, on a dry run);
// Failed holds those still registered when Retire finished.
type RetireReport struct {
	Removed, Failed []string
	Restored        int
}

// Retire removes every marketplace in names that Claude Code still has
// registered, then lists again and reports any that survived. The second list
// is the point: MEM-002 deleted directories and a settings.json key, Claude Code
// kept the registration in a file the cleanup never read, and the marketplace
// re-cloned on every session for months (#1431). Asking the CLI that owns the
// registry, and checking its answer after the removal, cannot drift that way.
func (s Syncer) Retire(names []string, dryRun bool) (RetireReport, error) {
	var rep RetireReport
	if len(names) == 0 {
		return rep, nil
	}
	registered, err := s.marketplaces(&rep)
	if err != nil {
		return rep, err
	}
	for _, name := range names {
		switch {
		case !registered[name]:
		case dryRun:
			rep.Removed = append(rep.Removed, name)
		default:
			restored, err := Guard(s.ClaudeJSON, s.Floor, func() error { return s.Run.RemoveMarketplace(name) })
			rep.Restored += b2i(restored)
			if err != nil {
				rep.Failed = append(rep.Failed, name)
			}
		}
	}
	if dryRun {
		return rep, nil
	}
	after, err := s.marketplaces(&rep)
	if err != nil {
		return rep, err
	}
	for _, name := range names {
		if !registered[name] || slices.Contains(rep.Failed, name) {
			continue
		}
		if after[name] {
			rep.Failed = append(rep.Failed, name)
			continue
		}
		rep.Removed = append(rep.Removed, name)
	}
	return rep, nil
}

// marketplaces is the set of marketplace names Claude Code has registered. A
// list that fails or does not parse is an error, never an empty set: reading
// a broken CLI as "nothing registered" would report a retired one as gone.
func (s Syncer) marketplaces(rep *RetireReport) (map[string]bool, error) {
	var out string
	restored, err := Guard(s.ClaudeJSON, s.Floor, func() error {
		var err error
		out, err = s.Run.Marketplaces()
		return err
	})
	rep.Restored += b2i(restored)
	if err != nil {
		return nil, fmt.Errorf("claude plugin marketplace list: %w", err)
	}
	var list []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return nil, fmt.Errorf("claude plugin marketplace list --json: %w", err)
	}
	set := map[string]bool{}
	for _, m := range list {
		set[m.Name] = true
	}
	return set, nil
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
