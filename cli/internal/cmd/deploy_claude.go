package cmd

import (
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
	ids, err := claude.LoadPlugins(filepath.Join(repoRoot, filepath.FromSlash(claude.PluginsRel)))
	if err != nil {
		return fmt.Errorf("%s: %w", step, err)
	}
	rep, err := s.Sync(ids, dryRun)
	reportRestored(w, step, s.ClaudeJSON, rep.Restored)
	if err != nil {
		return fmt.Errorf("%s: %w", step, err)
	}
	reportCounts(w, step, dryRun, len(rep.Added), len(rep.Failed), len(rep.Present))
	for _, id := range rep.Failed {
		_, _ = fmt.Fprintf(w, "failed    %-10s %s did not install\n", step, id)
	}
	if len(rep.Failed) > 0 {
		return fmt.Errorf("%s: %d plugin(s) failed to install: %s", step, len(rep.Failed), strings.Join(rep.Failed, ", "))
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
	migrated := "migrated  %-10s %s (stale uvx hive-vault entry replaced by the SSOT one)\n"
	if dryRun {
		migrated = "would migrate %-6s %s (stale uvx hive-vault entry)\n"
	}
	for _, name := range rep.Migrated {
		_, _ = fmt.Fprintf(w, migrated, step, name)
	}
	for _, sk := range rep.Skipped {
		_, _ = fmt.Fprintf(w, "skipped   %-10s %s (%s not on PATH)\n", step, sk.Name, sk.Missing)
	}
	reportCounts(w, step, dryRun, len(rep.Added), len(rep.Failed), len(rep.Present))
	for _, name := range rep.Failed {
		_, _ = fmt.Fprintf(w, "failed    %-10s %s did not register\n", step, name)
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
		_, _ = fmt.Fprintf(w, "skipped   %-10s (claude not installed)\n", step)
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
		_, _ = fmt.Fprintf(w, "restored  %-10s %s shrank during a claude call (a known Claude Code truncation bug); restored it from the snapshot\n", step, path)
	}
}

func reportCounts(w io.Writer, step string, dryRun bool, added, failed, present int) {
	state, verb := "installed", "added"
	switch {
	case added == 0 && failed == 0:
		state = "in sync"
	case dryRun:
		state, verb = "would add", "would add"
	}
	_, _ = fmt.Fprintf(w, "%-9s %-10s %d %s, %d already present\n", state, step, added, verb, present)
}
