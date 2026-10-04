package cmd

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/claude"
	"github.com/mlorentedev/dotfiles/cli/internal/mem"
)

// claudeCLI is the production claude.Runner: the real `claude` on PATH.
type claudeCLI struct{}

func (claudeCLI) List() (string, error) {
	out, err := exec.Command("claude", "plugin", "list").Output()
	return string(out), err
}

func (claudeCLI) Install(id string) error {
	out, err := exec.Command("claude", "plugin", "install", id).CombinedOutput() //nolint:gosec // id comes from the validated plugins.json
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// deployClaudeRunner is the seam behind the plugin step: nil when claude is
// not installed, so the step is skipped and says so. A test answers for it, and
// runDeploy defaults it to nil, so no test ever drives the box's real claude.
var deployClaudeRunner = func() claude.Runner {
	if !deployCommandAvailable("claude") {
		return nil
	}
	return claudeCLI{}
}

// deployClaudePlugins is the converge step a bare `dotf deploy` runs for the
// Claude Code plugins in ai/claude/plugins.json (CLI-063, #1339), each claude
// call inside the .claude.json snapshot guard. A failed install is reported by
// name and fails the run: the setups print a warning on a non-zero deploy, and
// a plugin that silently did not install is the #1491 miscount in another form.
func deployClaudePlugins(w io.Writer, repoRoot, home string, dryRun bool) error {
	const step = "claude-plugins"
	run := deployClaudeRunner()
	if run == nil {
		_, _ = fmt.Fprintf(w, "skipped   %-10s (claude not installed)\n", step)
		return nil
	}
	ids, err := claude.LoadPlugins(filepath.Join(repoRoot, filepath.FromSlash(claude.PluginsRel)))
	if err != nil {
		return fmt.Errorf("%s: %w", step, err)
	}
	s := claude.Syncer{
		Run:        run,
		ClaudeJSON: filepath.Join(home, ".claude", ".claude.json"),
		Floor:      mem.ClaudeJSONFloor(memConfigPath()),
	}
	rep, err := s.Sync(ids, dryRun)
	if rep.Restored > 0 {
		_, _ = fmt.Fprintf(w, "restored  %-10s %s shrank during a claude call (a known Claude Code truncation bug); restored it from the snapshot\n", step, s.ClaudeJSON)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", step, err)
	}
	state, verb := "installed", "added"
	switch {
	case len(rep.Added) == 0 && len(rep.Failed) == 0:
		state = "in sync"
	case dryRun:
		state, verb = "would add", "would add"
	}
	_, _ = fmt.Fprintf(w, "%-9s %-10s %d %s, %d already present\n", state, step, len(rep.Added), verb, len(rep.Present))
	for _, id := range rep.Failed {
		_, _ = fmt.Fprintf(w, "failed    %-10s %s did not install\n", step, id)
	}
	if len(rep.Failed) > 0 {
		return fmt.Errorf("%s: %d plugin(s) failed to install: %s", step, len(rep.Failed), strings.Join(rep.Failed, ", "))
	}
	return nil
}
