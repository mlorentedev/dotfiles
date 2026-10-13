package converge

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/claude"
)

// agentExtensions registers what each agent loads beyond its config files: for
// Claude Code, the MCP servers in mcp-servers.json and the plugins in
// ai/claude/plugins.json (#1843 B10). Until it existed only a bare `dotf deploy`
// did this, and only setup-linux.sh and setup-windows.ps1 called that, so a Mac
// brought up by install.sh had a Claude Code with no plugins and no hive
// (#2387). It runs the same claude.ApplyMCP and claude.ApplyPlugins as the bare
// deploy. Every OS.
//
// It runs after tools, whose PATH reaches the claude that mise installs, and
// after configs-deploy, which writes Claude Code's settings. It sits last among
// the native steps because the first failure stops a run: a plugin that cannot
// install for want of network must not keep git or the SSH target from
// converging.
type agentExtensions struct {
	// claude builds the guarded claude seam for home; false when claude is not
	// installed. Unset, the step is skipped.
	claude func(home string) (claude.Syncer, bool)
}

func (agentExtensions) Name() string        { return "agent-extensions" }
func (agentExtensions) Platforms() []string { return nil }

// extensionsPlan is one pass over Claude Code: what it changed, or under a
// plan would change, and its one-line account.
type extensionsPlan struct {
	changes int
	detail  string
	todo    []string // what is still to add, named
}

func (a agentExtensions) apply(env Env, dryRun bool) (extensionsPlan, string, error) {
	if a.claude == nil {
		return extensionsPlan{}, "no claude runner is wired into this registry", nil
	}
	s, ok := a.claude(env.Home)
	if !ok {
		return extensionsPlan{}, "claude is not installed", nil
	}
	mcp, mErr := claude.ApplyMCP(s, env.RepoRoot, dryRun)
	plugins, pErr := claude.ApplyPlugins(s, env.RepoRoot, dryRun)
	if err := errors.Join(prefixed("claude mcp", mErr), prefixed("claude plugins", pErr)); err != nil {
		return extensionsPlan{}, "", err
	}
	added, removed := "added", "removed"
	if dryRun {
		added, removed = "to add", "to remove"
	}
	p := extensionsPlan{changes: len(mcp.Added) + len(plugins.Plugins.Added) + len(plugins.Retired.Removed)}
	parts := []string{fmt.Sprintf("claude: %d MCP server(s) %s, %d present", len(mcp.Added), added, len(mcp.Present))}
	for _, sk := range mcp.Skipped {
		parts = append(parts, fmt.Sprintf("%s skipped (%s not on PATH)", sk.Name, sk.Missing))
	}
	parts = append(parts, fmt.Sprintf("%d plugin(s) %s, %d present", len(plugins.Plugins.Added), added, len(plugins.Plugins.Present)))
	if len(plugins.Retired.Removed) > 0 {
		parts = append(parts, fmt.Sprintf("retired marketplace(s) %s: %s", removed, strings.Join(plugins.Retired.Removed, ", ")))
	}
	if n := mcp.Restored + plugins.Plugins.Restored + plugins.Retired.Restored; n > 0 {
		parts = append(parts, fmt.Sprintf(".claude.json restored %d time(s) after a claude call shrank it", n))
	}
	p.detail = strings.Join(parts, "; ")
	p.todo = slices.Concat(mcp.Added, plugins.Plugins.Added, plugins.Retired.Removed)
	return p, "", nil
}

func (a agentExtensions) Reconcile(env Env, dryRun bool) (Result, error) {
	p, skip, err := a.apply(env, dryRun)
	if err != nil {
		return Result{}, err
	}
	if skip != "" {
		return Result{Skip: skip}, nil
	}
	return Result{Changes: p.changes, Detail: p.detail}, nil
}

// Probe re-plans: after an apply, nothing may still be to add or to remove. A
// plan only asks claude (`mcp get`, `plugin list`, `marketplace list`).
func (a agentExtensions) Probe(env Env) error {
	p, _, err := a.apply(env, true)
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
