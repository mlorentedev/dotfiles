package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/claude"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/mem"
)

// claudeCLI is the production claude.Runner: the real `claude` on PATH, run
// with CLAUDE_CONFIG_DIR pinned to the directory the snapshot guard watches.
// Claude Code keeps .claude.json in $CLAUDE_CONFIG_DIR when it is set and in
// the home root when it is not, so a guard that resolved the directory one way
// and a child that inherited a different environment would protect a file the
// CLI never writes. Passing the value down makes them agree by construction.
type claudeCLI struct{ configDir string }

func (c claudeCLI) command(args ...string) *exec.Cmd {
	cmd := exec.Command("claude", args...)
	cmd.Env = append(os.Environ(), "CLAUDE_CONFIG_DIR="+c.configDir)
	return cmd
}

func (c claudeCLI) List() (string, error) {
	out, err := c.command("plugin", "list").Output()
	return string(out), err
}

func (c claudeCLI) Install(id string) error {
	return c.combined("plugin", "install", id)
}

func (c claudeCLI) Marketplaces() (string, error) {
	out, err := c.command("plugin", "marketplace", "list", "--json").Output()
	return string(out), err
}

func (c claudeCLI) RemoveMarketplace(name string) error {
	return c.combined("plugin", "marketplace", "remove", name)
}

func (c claudeCLI) McpGet(name string) (string, error) {
	out, err := c.command("mcp", "get", name).Output()
	return string(out), err
}

func (c claudeCLI) McpAdd(name, transport string, args []string) error {
	return c.combined(mcpAddArgv(name, transport, args)...)
}

// mcpAddArgv is the twins' `claude mcp add` shape: options before `--`, the
// server's own argv after it, so a server flag is never read as claude's.
func mcpAddArgv(name, transport string, args []string) []string {
	return append([]string{"mcp", "add", "--transport", transport, name, "--scope", "user", "--"}, args...)
}

func (c claudeCLI) McpRemove(name string) error {
	return c.combined("mcp", "remove", name, "--scope", "user")
}

// combined runs a mutating claude call and folds its output into the error, so
// a failure names what the CLI said rather than only its exit status.
func (c claudeCLI) combined(args ...string) error {
	out, err := c.command(args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// claudeConfigDir is CLAUDE_CONFIG_DIR through the env contract: the process
// value, else the machine override, else the contract default (~/.claude on
// every OS, the value the rc files export).
func claudeConfigDir(home string) string {
	if dir := env.ResolvePath("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(home, ".claude")
}

// deployClaudeRunner is the seam behind the plugin step: nil when claude is
// not installed, so the step is skipped and says so. A test answers for it, and
// runDeploy defaults it to nil, so no test ever drives the box's real claude.
var deployClaudeRunner = func(configDir string) claude.Runner {
	if !deployCommandAvailable("claude") {
		return nil
	}
	return claudeCLI{configDir: configDir}
}

// deployClaudePlugins is the converge step a bare `dotf deploy` runs for the
// Claude Code plugins in ai/claude/plugins.json (CLI-063, #1339), each claude
// call inside the .claude.json snapshot guard. A failed install is reported by
// name and fails the run: the setups print a warning on a non-zero deploy, and
// a plugin that silently did not install is the #1491 miscount in another form.
func deployClaudePlugins(w io.Writer, repoRoot, home string, dryRun bool) error {
	const step = "claude-plugins"
	s, ok := claudeSyncer(w, step, home)
	if !ok {
		return nil
	}
	list, err := claude.LoadPlugins(filepath.Join(repoRoot, filepath.FromSlash(claude.PluginsRel)))
	if err != nil {
		return fmt.Errorf("%s: %w", step, err)
	}
	retireErr := retireMarketplaces(w, step, s, list.RetiredMarketplaces, dryRun)
	rep, err := s.Sync(list.Plugins, dryRun)
	reportRestored(w, step, s.ClaudeJSON, rep.Restored)
	if err != nil {
		return fmt.Errorf("%s: %w", step, err)
	}
	reportCounts(w, step, dryRun, len(rep.Added), len(rep.Failed), len(rep.Present))
	for _, id := range rep.Failed {
		deployRow(w, "failed", step, "%s did not install", id)
	}
	if len(rep.Failed) > 0 {
		return errors.Join(retireErr, fmt.Errorf("%s: %d plugin(s) failed to install: %s", step, len(rep.Failed), strings.Join(rep.Failed, ", ")))
	}
	return retireErr
}

// retireMarketplaces removes the retired marketplaces a box still registers,
// before the installs: a failure is reported by name and returned, and the
// installs still run. It replaces the MEM-002 blocks the setups carried, which
// never removed the registration (#1431).
func retireMarketplaces(w io.Writer, step string, s claude.Syncer, names []string, dryRun bool) error {
	rep, err := s.Retire(names, dryRun)
	reportRestored(w, step, s.ClaudeJSON, rep.Restored)
	if err != nil {
		return fmt.Errorf("%s: %w", step, err)
	}
	verb := "removed"
	if dryRun {
		verb = "would remove"
	}
	for _, name := range rep.Removed {
		deployRow(w, verb, step, "retired marketplace %s", name)
	}
	for _, name := range rep.Failed {
		if cause := rep.Causes[name]; cause != nil {
			deployRow(w, "failed", step, "retired marketplace %s is still registered: %v", name, cause)
			continue
		}
		deployRow(w, "failed", step, "retired marketplace %s is still registered", name)
	}
	if len(rep.Failed) > 0 {
		return fmt.Errorf("%s: retired marketplace(s) still registered: %s", step, strings.Join(rep.Failed, ", "))
	}
	return nil
}

// deployClaudeMCP is the converge step that registers the servers in
// mcp-servers.json with Claude Code at user scope (CLI-063, #1339). It runs
// before the plugins, as both twins do. A server whose prerequisite binary is
// absent is skipped by name; its prerequisite command is not run (#1993).
func deployClaudeMCP(w io.Writer, repoRoot, home string, dryRun bool) error {
	const step = "claude-mcp"
	s, ok := claudeSyncer(w, step, home)
	if !ok {
		return nil
	}
	servers, err := claude.LoadServers(filepath.Join(repoRoot, claude.ServersRel))
	if err != nil {
		return fmt.Errorf("%s: %w", step, err)
	}
	s.HasCommand = deployCommandAvailable
	rep := s.Register(servers, dryRun)
	reportRestored(w, step, s.ClaudeJSON, rep.Restored)
	verb, why := "migrated", "(stale uvx hive-vault entry replaced by the SSOT one)"
	if dryRun {
		verb, why = "would migrate", "(stale uvx hive-vault entry)"
	}
	for _, name := range rep.Migrated {
		deployRow(w, verb, step, "%s %s", name, why)
	}
	for _, sk := range rep.Skipped {
		deployRow(w, "skipped", step, "%s (%s not on PATH)", sk.Name, sk.Missing)
	}
	reportCounts(w, step, dryRun, len(rep.Added), len(rep.Failed), len(rep.Present))
	for _, name := range rep.Failed {
		deployRow(w, "failed", step, "%s did not register", name)
	}
	if len(rep.Failed) > 0 {
		return fmt.Errorf("%s: %d server(s) failed to register: %s", step, len(rep.Failed), strings.Join(rep.Failed, ", "))
	}
	return nil
}

// claudeSyncer builds the guarded claude seam both steps share, or reports the
// step skipped when claude is not installed.
func claudeSyncer(w io.Writer, step, home string) (claude.Syncer, bool) {
	configDir := claudeConfigDir(home)
	run := deployClaudeRunner(configDir)
	if run == nil {
		deployRow(w, "skipped", step, "(claude not installed)")
		return claude.Syncer{}, false
	}
	return claude.Syncer{
		Run:        run,
		ClaudeJSON: filepath.Join(configDir, ".claude.json"),
		Floor:      mem.ClaudeJSONFloor(memConfigPath()),
	}, true
}

func reportRestored(w io.Writer, step, path string, n int) {
	if n > 0 {
		deployRow(w, "restored", step, "%s shrank during a claude call (a known Claude Code truncation bug); restored it from the snapshot", path)
	}
}

func reportCounts(w io.Writer, step string, dryRun bool, added, failed, present int) {
	state, verb := "installed", "added"
	switch {
	case added == 0 && failed == 0:
		state = "in sync"
	case dryRun:
		state, verb = "would add", "missing"
	}
	deployRow(w, state, step, "%d %s, %d already present", added, verb, present)
}
