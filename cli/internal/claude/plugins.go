package claude

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// PluginsRel is the plugin list, relative to the repo root.
const PluginsRel = "ai/claude/plugins.json"

// MarketplacesRel declares where each marketplace of a plugin comes from,
// relative to the repo root. It is not a field of plugins.json because every
// released dotf reads that file strictly: a new field there would fail the
// plugin step on any box whose binary predates it.
const MarketplacesRel = "ai/claude/marketplaces.json"

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

// LoadMarketplaceSources reads the marketplace sources as a name-to-source
// map, each source in the form `claude plugin marketplace add` takes. A
// checkout that predates the file has no sources, which is not an error: Sync
// then only refreshes marketplaces the box already registers.
func LoadMarketplaceSources(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // repo-relative, fixed name
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var doc struct {
		Comment      []string          `json:"$comment"`
		Marketplaces map[string]string `json:"marketplaces"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", MarketplacesRel, err)
	}
	for name, src := range doc.Marketplaces {
		if name == "" || strings.Contains(name, "@") {
			return nil, fmt.Errorf("%s: %q is not a marketplace name", MarketplacesRel, name)
		}
		if strings.TrimSpace(src) == "" {
			return nil, fmt.Errorf("%s: marketplace %q has an empty source", MarketplacesRel, name)
		}
	}
	if doc.Marketplaces == nil {
		doc.Marketplaces = map[string]string{}
	}
	return doc.Marketplaces, nil
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
	AddMarketplace(source string) error                 // `claude plugin marketplace add <source>`
	UpdateMarketplace(name string) error                // `claude plugin marketplace update <name>`
}

// Syncer installs the declared plugins a box lacks, each claude call inside
// the snapshot guard on ClaudeJSON.
type Syncer struct {
	Run        Runner
	ClaudeJSON string
	Floor      int
	HasCommand func(string) bool // PATH lookup for an MCP server's prerequisite binary
	Sources    map[string]string // marketplace name to its `marketplace add` source
}

// PluginReport is what one Sync found and did. Added holds only installs that
// succeeded, or would be attempted on a dry run; a failed install is in Failed
// and nowhere else (AC7; the PowerShell twin counted attempts, #1491), and
// Causes holds what the CLI said for it. Every id Sync is given lands in
// exactly one of Present, Added and Failed.
type PluginReport struct {
	Present, Added, Failed []string
	Causes                 map[string]error // the install's error, for each id in Failed
	Restored               int              // .claude.json restores the guard made
}

// Sync installs every id in ids that `claude plugin list` does not name.
// A plugin counts as present when its id is a whole whitespace-separated token
// of the list output (the CLI prints one `<glyph> <id>` line per plugin). The
// twins grep for a substring, under which a declared `lsp@m` reads as present
// on a box that only has `gopls-lsp@m`. A list that fails is returned, not read
// as empty: treating a broken CLI as "nothing installed" reinstalls everything.
//
// Before the first install from a marketplace, Sync prepares it once: a
// marketplace the box does not register is added from its declared source,
// and one it registers is refreshed, since its copy may predate a plugin
// declared since. A fresh Claude Code no longer registers
// claude-plugins-official, so without this every declared plugin failed on a
// new machine (#2378). A failed preparation is not fatal, since the install
// may still succeed; it is named in the cause of any install from that
// marketplace that then fails.
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
	prep := &marketPrep{done: map[string]error{}}
	for _, id := range ids {
		switch {
		case installed[id]:
			rep.Present = append(rep.Present, id)
		case dryRun:
			rep.Added = append(rep.Added, id)
		default:
			prepErr := s.prepare(marketplaceOf(id), prep, &rep.Restored)
			restored, err := Guard(s.ClaudeJSON, s.Floor, func() error { return s.Run.Install(id) })
			rep.Restored += b2i(restored)
			if err != nil {
				if prepErr != nil {
					err = fmt.Errorf("%w (preparing its marketplace failed first: %v)", err, prepErr)
				}
				rep.Failed = append(rep.Failed, id)
				if rep.Causes == nil {
					rep.Causes = map[string]error{}
				}
				rep.Causes[id] = err
				continue
			}
			rep.Added = append(rep.Added, id)
		}
	}
	return rep, nil
}

// marketPrep is one Sync's preparation state: the registry, listed at most
// once and only when there is something to install, and each marketplace's
// outcome, so a failure is reported per install and never retried.
type marketPrep struct {
	listed     bool
	registered map[string]bool
	listErr    error
	done       map[string]error
}

// prepare readies the marketplace name for an install the first time it is
// asked, and returns that outcome on every later ask without calling again.
// An id with no marketplace part has nothing to prepare.
func (s Syncer) prepare(name string, p *marketPrep, restored *int) error {
	if name == "" {
		return nil
	}
	if err, ok := p.done[name]; ok {
		return err
	}
	if !p.listed {
		p.listed = true
		p.registered, p.listErr = s.marketplaces(restored)
	}
	var err error
	switch src := s.Sources[name]; {
	case p.listErr != nil:
		err = fmt.Errorf("cannot tell whether marketplace %q is registered: %w", name, p.listErr)
	case p.registered[name]:
		err = s.guarded(restored, func() error { return s.Run.UpdateMarketplace(name) })
	case src != "":
		if err = s.guarded(restored, func() error { return s.Run.AddMarketplace(src) }); err != nil {
			err = fmt.Errorf("registering marketplace %q from %s: %w", name, src, err)
		}
	default:
		err = fmt.Errorf("marketplace %q is not registered and %s declares no source", name, MarketplacesRel)
	}
	p.done[name] = err
	return err
}

// guarded runs fn inside the .claude.json snapshot guard and counts a restore.
func (s Syncer) guarded(restored *int, fn func() error) error {
	r, err := Guard(s.ClaudeJSON, s.Floor, fn)
	*restored += b2i(r)
	return err
}

// marketplaceOf returns the marketplace of a `<plugin>@<marketplace>` id, or ""
// when the id names none.
func marketplaceOf(id string) string {
	if i := strings.LastIndex(id, "@"); i >= 0 {
		return id[i+1:]
	}
	return ""
}

// RetireReport is what one Retire found and did. Removed holds marketplaces
// whose registration is gone afterwards (or would be removed, on a dry run);
// Failed holds those still registered when Retire finished, and Causes the
// CLI's error for each of them whose removal exited non-zero.
type RetireReport struct {
	Removed, Failed []string
	Causes          map[string]error
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
	registered, err := s.marketplaces(&rep.Restored)
	if err != nil {
		return rep, err
	}
	causes := map[string]error{}
	for _, name := range names {
		switch {
		case !registered[name]:
		case dryRun:
			rep.Removed = append(rep.Removed, name)
		default:
			restored, err := Guard(s.ClaudeJSON, s.Floor, func() error { return s.Run.RemoveMarketplace(name) })
			rep.Restored += b2i(restored)
			if err != nil {
				causes[name] = err
			}
		}
	}
	if dryRun {
		return rep, nil
	}
	// The second list judges every name, a failed removal included: the end
	// state decides, and the CLI's error only explains a survivor.
	after, err := s.marketplaces(&rep.Restored)
	if err != nil {
		return rep, err
	}
	for _, name := range names {
		if !registered[name] {
			continue
		}
		if !after[name] {
			rep.Removed = append(rep.Removed, name)
			continue
		}
		rep.Failed = append(rep.Failed, name)
		if causes[name] != nil {
			if rep.Causes == nil {
				rep.Causes = map[string]error{}
			}
			rep.Causes[name] = causes[name]
		}
	}
	return rep, nil
}

// marketplaces is the set of marketplace names Claude Code has registered. A
// list that fails or does not parse is an error, never an empty set: reading
// a broken CLI as "nothing registered" would report a retired one as gone.
func (s Syncer) marketplaces(restored *int) (map[string]bool, error) {
	var out string
	err := s.guarded(restored, func() error {
		var err error
		out, err = s.Run.Marketplaces()
		return err
	})
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
	for i, m := range list {
		if m.Name == "" {
			// The list's shape changed. Read leniently it would be an empty
			// registry, and every retired marketplace would look gone.
			return nil, fmt.Errorf("claude plugin marketplace list --json: entry %d has no name", i)
		}
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
