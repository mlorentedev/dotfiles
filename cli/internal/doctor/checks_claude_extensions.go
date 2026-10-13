package doctor

import (
	"errors"
	"fmt"

	"github.com/mlorentedev/dotfiles/cli/internal/claude"
)

// claudeRemedy is what fixes every finding but an untouched registration.
const claudeRemedy = "run `dotf converge --only agent-extensions`"

// checkClaudeExtensions reports the Claude Code MCP servers and plugins the
// checkout declares that this machine lacks or registers differently (CLI-106,
// #2393). It asks through the claude.ApplyMCP and claude.ApplyPlugins converge
// runs, in a dry run, so it never makes a mutating claude call.
//
// It costs about a second per server, because `claude mcp get` starts the
// server it describes, and every claude call runs inside the .claude.json
// snapshot guard: if one shrinks that file (upstream #59870), the guard writes
// the snapshot back. That is the only write this check can cause.
func checkClaudeExtensions(sys *System, cfg *Config, rep *Report, syncer func(home string) (claude.Syncer, bool)) {
	rep.Section("Claude extensions")
	if syncer == nil {
		rep.Skip("no claude runner is wired into this doctor")
		return
	}
	if cfg.RepoDir == "" {
		rep.Skip("not run from a dotfiles checkout, which declares the servers and plugins")
		return
	}
	s, ok := syncer(sys.home())
	if !ok {
		rep.Skip("claude is not installed")
		return
	}
	mcp, mErr := claude.ApplyMCP(s, cfg.RepoDir, true)
	plugins, pErr := claude.ApplyPlugins(s, cfg.RepoDir, true)
	if mErr != nil || pErr != nil {
		rep.Warn(fmt.Sprintf("could not read Claude's registrations: %v", errors.Join(mErr, pErr)))
		return
	}
	findings := claudeFindings(mcp, plugins)
	for _, f := range findings {
		rep.Warn(f)
	}
	for _, sk := range mcp.Skipped {
		rep.Info(fmt.Sprintf("MCP server %s not checked: %s is not on PATH", sk.Name, sk.Missing))
	}
	if n := mcp.Restored + plugins.Plugins.Restored + plugins.Retired.Restored; n > 0 {
		rep.Warn(fmt.Sprintf(".claude.json restored %d time(s) after a claude call shrank it", n))
	}
	if len(findings) == 0 {
		rep.Pass(fmt.Sprintf("%d MCP server(s) and %d plugin(s) as declared", len(mcp.Present), len(plugins.Plugins.Present)))
	}
}

// claudeFindings is one line per server or plugin that differs from the
// declaration, each naming what fixes it.
func claudeFindings(mcp claude.MCPReport, plugins claude.PluginsResult) []string {
	migrated := map[string]claude.Drift{}
	for _, d := range mcp.Migrated {
		migrated[d.Name] = d
	}
	var out []string
	for _, name := range mcp.Added {
		if d, ok := migrated[name]; ok {
			out = append(out, fmt.Sprintf("MCP server %s is registered as %s, declared as %s (%s)", name, d.Was, d.Now, claudeRemedy))
			continue
		}
		out = append(out, fmt.Sprintf("MCP server %s is not registered (%s)", name, claudeRemedy))
	}
	for _, u := range mcp.Untouched {
		out = append(out, fmt.Sprintf("MCP server %s: %s; converge leaves it as it is", u.Name, u.Why))
	}
	for _, id := range plugins.Plugins.Added {
		out = append(out, fmt.Sprintf("plugin %s is not installed (%s)", id, claudeRemedy))
	}
	for _, name := range plugins.Retired.Removed {
		out = append(out, fmt.Sprintf("retired marketplace %s is still registered (%s)", name, claudeRemedy))
	}
	return out
}
