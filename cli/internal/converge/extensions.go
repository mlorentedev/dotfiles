package converge

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/agy"
	"github.com/mlorentedev/dotfiles/cli/internal/claude"
)

// agentExtensions registers what each agent loads beyond its config files: for
// Claude Code, the MCP servers in mcp-servers.json and the plugins in
// ai/claude/plugins.json (#1843 B10); for agy, the MCP configs it reads from
// ~/.gemini (CLI-105, #2384). Until it existed only a bare `dotf deploy` did
// the Claude half and only the setup twins the agy half, and only
// setup-linux.sh and setup-windows.ps1 ran either, so a Mac brought up by
// install.sh had a Claude Code with no plugins and no hive (#2387) and an agy
// with no MCP config at all. It runs the same claude.ApplyMCP,
// claude.ApplyPlugins and agy.ApplyMCP as the bare deploy. Every OS.
//
// It runs after tools, whose PATH reaches the claude that mise installs, and
// after configs-deploy, which writes Claude Code's settings. It sits last among
// the native steps because the first failure stops a run: a plugin that cannot
// install for want of network must not keep git or the SSH target from
// converging.
type agentExtensions struct {
	// claude builds the guarded claude seam for home; false when claude is not
	// installed. Unset, the Claude half is skipped.
	claude func(home string) (claude.Syncer, bool)
	// agyPaths says where agy's configs go and which vault hive-vault reads.
	// Unset, the agy half is skipped. agy's files are written whether or not
	// agy is installed, as every ai/deploy.json agy entry is.
	agyPaths func(home string) agy.Paths
}

func (agentExtensions) Name() string        { return "agent-extensions" }
func (agentExtensions) Platforms() []string { return nil }

// extensionsPlan is one pass over every agent: what it changed, or under a
// plan would change, and its one-line account.
type extensionsPlan struct {
	changes int
	detail  string
	todo    []string // what is still to add, named
	skipped []string // why a half did not run
}

func (a agentExtensions) apply(e Env, dryRun bool) (extensionsPlan, error) {
	var p extensionsPlan
	var parts []string
	cErr := a.applyClaude(e, dryRun, &p, &parts)
	aErr := a.applyAgy(e, dryRun, &p, &parts)
	if err := errors.Join(cErr, aErr); err != nil {
		return extensionsPlan{}, err
	}
	p.detail = strings.Join(parts, "; ")
	return p, nil
}

func (a agentExtensions) applyClaude(e Env, dryRun bool, p *extensionsPlan, parts *[]string) error {
	if a.claude == nil {
		p.skipped = append(p.skipped, "no claude runner is wired into this registry")
		return nil
	}
	s, ok := a.claude(e.Home)
	if !ok {
		p.skipped = append(p.skipped, "claude is not installed")
		return nil
	}
	mcp, mErr := claude.ApplyMCP(s, e.RepoRoot, dryRun)
	plugins, pErr := claude.ApplyPlugins(s, e.RepoRoot, dryRun)
	if err := errors.Join(prefixed("claude mcp", mErr), prefixed("claude plugins", pErr)); err != nil {
		return err
	}
	added, removed, migrated := "added", "removed", "re-registered"
	if dryRun {
		added, removed, migrated = "to add", "to remove", "to re-register"
	}
	p.changes += len(mcp.Added) + len(plugins.Plugins.Added) + len(plugins.Retired.Removed)
	c := []string{fmt.Sprintf("claude: %d MCP server(s) %s, %d present", len(mcp.Added), added, len(mcp.Present))}
	for _, d := range mcp.Migrated {
		c = append(c, fmt.Sprintf("%s %s (%s -> %s)", d.Name, migrated, d.Was, d.Now))
	}
	if len(mcp.Unverified) > 0 {
		c = append(c, "unverified (mcp get did not parse): "+strings.Join(mcp.Unverified, ", "))
	}
	for _, sk := range mcp.Skipped {
		c = append(c, fmt.Sprintf("%s skipped (%s not on PATH)", sk.Name, sk.Missing))
	}
	c = append(c, fmt.Sprintf("%d plugin(s) %s, %d present", len(plugins.Plugins.Added), added, len(plugins.Plugins.Present)))
	if len(plugins.Retired.Removed) > 0 {
		c = append(c, fmt.Sprintf("retired marketplace(s) %s: %s", removed, strings.Join(plugins.Retired.Removed, ", ")))
	}
	if n := mcp.Restored + plugins.Plugins.Restored + plugins.Retired.Restored; n > 0 {
		c = append(c, fmt.Sprintf(".claude.json restored %d time(s) after a claude call shrank it", n))
	}
	*parts = append(*parts, c...)
	p.todo = append(p.todo, slices.Concat(mcp.Added, plugins.Plugins.Added, plugins.Retired.Removed)...)
	return nil
}

func (a agentExtensions) applyAgy(e Env, dryRun bool, p *extensionsPlan, parts *[]string) error {
	if a.agyPaths == nil {
		p.skipped = append(p.skipped, "no agy paths are wired into this registry")
		return nil
	}
	rep, err := agy.ApplyMCP(e.RepoRoot, a.agyPaths(e.Home), dryRun)
	if err != nil {
		return prefixed("agy mcp", err)
	}
	if rep.NoSource {
		p.skipped = append(p.skipped, "no "+agy.MCPRel+" in the checkout")
		return nil
	}
	written, removed := "written", "removed"
	if dryRun {
		written, removed = "to write", "to remove"
	}
	c := fmt.Sprintf("agy: %d MCP config(s) %s, %d in sync", len(rep.Written), written, len(rep.InSync))
	if len(rep.Removed) > 0 {
		c += fmt.Sprintf(", %d stale copy %s", len(rep.Removed), removed)
	}
	if rep.VaultMissing != "" {
		c += ", no vault at " + rep.VaultMissing
	}
	*parts = append(*parts, c)
	p.changes += rep.Changes()
	p.todo = append(p.todo, slices.Concat(rep.Written, rep.Removed)...)
	return nil
}

// Reconcile is skipped only when neither half ran; a half that did not run is
// named in the detail of one that did.
func (a agentExtensions) Reconcile(e Env, dryRun bool) (Result, error) {
	p, err := a.apply(e, dryRun)
	if err != nil {
		return Result{}, err
	}
	if p.detail == "" {
		return Result{Skip: strings.Join(p.skipped, "; ")}, nil
	}
	detail := p.detail
	if len(p.skipped) > 0 {
		detail += " (" + strings.Join(p.skipped, "; ") + ")"
	}
	return Result{Changes: p.changes, Detail: detail}, nil
}

// Probe re-plans: after an apply, nothing may still be to add or to remove. A
// plan only asks claude (`mcp get`, `plugin list`, `marketplace list`) and
// reads agy's files.
func (a agentExtensions) Probe(e Env) error {
	p, err := a.apply(e, true)
	if err != nil {
		return err
	}
	if p.changes > 0 {
		return fmt.Errorf("still to register after the apply: %s", strings.Join(p.todo, ", "))
	}
	return nil
}

func prefixed(what string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", what, err)
}
