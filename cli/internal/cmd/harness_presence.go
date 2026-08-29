package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// newHarnessPresenceCmd injects the agent-presence roster into every harness
// instructions file the manifest names (HARNESS-092, #1326).
//
// It is the fourth member of the family the shell renderer delegates to, and
// the first that writes: compile-harness.sh --deploy calls it on Linux, and
// setup-windows.ps1 calls it on Windows, where no port of the shell's
// deploy_agent_presence ever existed — so no harness on that OS had ever been
// told which skills a persona forces. One implementation, one sha, both OSes.
func newHarnessPresenceCmd() *cobra.Command {
	var repoRoot, home, render string
	cmd := &cobra.Command{
		Use:   "presence",
		Short: "Inject each persona's forced-skills roster into the harness instructions files",
		Long: `presence renders, for every harness the manifest's agents.presence[] names, the
roster of invocable personas that target it and the skills each MUST consume,
and writes it between AGENT-PRESENCE markers in that harness's instructions
file under $HOME — replacing an existing region in place, appending a fresh one
otherwise. Everything outside the markers is left byte-identical; a file whose
region already holds this roster is not rewritten.

The begin marker carries the roster's sha, so dotf doctor can tell current
from stale without re-rendering. A harness no persona targets gets no region.
An instructions file that does not exist yet is skipped and said so: the
harness's base file is deployed elsewhere, and presence has nothing to join.`,
		Example: `  dotf harness presence
  dotf harness presence --repo-root /path/to/checkout   # from a setup script`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoRoot == "" {
				repoRoot = env.RepoDir()
			}
			if repoRoot == "" {
				return fmt.Errorf("cannot locate the dotfiles checkout — pass --repo-root, set DOTFILES_REPO_DIR, or run from inside it")
			}
			if home == "" {
				home = env.Home()
			}
			if render != "" {
				block, err := harness.RenderPresence(repoRoot, render)
				if err != nil {
					return err
				}
				_, _ = fmt.Fprint(cmd.OutOrStdout(), block)
				return nil
			}
			outcomes, err := harness.DeployPresence(repoRoot, home)
			if err != nil {
				return err
			}
			w, e := cmd.OutOrStdout(), cmd.ErrOrStderr()
			for _, o := range outcomes {
				switch o.Status {
				case "injected":
					_, _ = fmt.Fprintf(w, "[deploy] presence -> %s (%s)\n", o.File, o.Agent)
				case "unchanged":
					_, _ = fmt.Fprintf(w, "[deploy] presence current: %s (%s)\n", o.File, o.Agent)
				case "absent":
					_, _ = fmt.Fprintf(e, "[deploy] presence target absent, skipping: %s\n", o.File)
				case "empty":
					_, _ = fmt.Fprintf(e, "[deploy] no persona targets %s; no presence region for %s\n", o.Agent, o.File)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "checkout holding harness/manifest.json and the agent records (default: DOTFILES_REPO_DIR or the cwd walk-up)")
	cmd.Flags().StringVar(&home, "home", "", "directory the manifest's presence files are relative to (default: the user's home)")
	cmd.Flags().StringVar(&render, "render", "", "print the roster block for this harness to stdout and write nothing (what --check compares against)")
	return cmd
}
