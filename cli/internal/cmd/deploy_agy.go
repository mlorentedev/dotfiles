package cmd

import (
	"io"
	"os"

	"github.com/mlorentedev/dotfiles/cli/internal/agy"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
)

// agyPaths is where agy's MCP configs go on this machine and the vault
// hive-vault is pointed at. Shared by the bare deploy and converge's
// agent-extensions step, so both write the same files.
func agyPaths(home string) agy.Paths {
	return agy.ResolvePaths(home, os.Getenv, env.ResolvePath)
}

// deployAgyMCP writes agy's MCP configs (CLI-105, #2384), which the setup
// twins wrote until the bare deploy did. One row per file it writes or
// removes, one for the files already in sync.
func deployAgyMCP(w io.Writer, repoRoot, home string, dryRun bool) error {
	const step = "agy-mcp"
	rep, err := agy.ApplyMCP(repoRoot, agyPaths(home), dryRun)
	if err != nil {
		deployRow(w, "failed", step, "%v", err)
		return err
	}
	if rep.NoSource {
		deployRow(w, "skipped", step, "(no %s in this checkout)", agy.MCPRel)
		return nil
	}
	wrote, removed := "deployed", "removed"
	if dryRun {
		wrote, removed = "would deploy", "would remove"
	}
	for _, p := range rep.Written {
		deployRow(w, wrote, step, "%s", p)
	}
	for _, p := range rep.Removed {
		deployRow(w, removed, step, "%s (stale copy)", p)
	}
	if rep.Changes() == 0 {
		deployRow(w, "in sync", step, "%d file(s)", len(rep.InSync))
	}
	if rep.VaultMissing != "" {
		deployRow(w, "warning", step, "no vault at %s: hive-vault errors until it exists (set VAULT_PATH to move it)", rep.VaultMissing)
	}
	return nil
}
