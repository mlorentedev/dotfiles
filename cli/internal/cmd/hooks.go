package cmd

import (
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/hooks"
	"github.com/spf13/cobra"
)

func newHooksCmd() *cobra.Command {
	h := &cobra.Command{
		Use:          "hooks",
		Short:        "Global git hook dispatcher (GUARD-001)",
		Long:         "Manage the machine-wide memory-sink dispatcher wired through git's core.hooksPath.",
		SilenceUsage: true,
		RunE:         func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	h.AddCommand(newHooksInstallCmd())
	return h
}

func newHooksInstallCmd() *cobra.Command {
	var source, dotfilesDir string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Deploy the GUARD dispatcher and wire core.hooksPath at it",
		Long: `install mirrors the dispatcher tree into $DOTFILES_DIR/git-hooks and points
git's global core.hooksPath at it.

It replaces scripts/install-git-hooks.{sh,ps1} (CLI-072). Those twins carried the
same guards but not the same tests — 13 bats cases against 9 Pester ones — so the
self-mirror guard, the dispatcher-equivalence probe and the trailing-slash path
comparison were verified on Linux only.

Safety, unchanged from the twins: the mirror is a clean one, because a hook
removed upstream must stop firing and a stale security hook is worse than none;
an unrelated pre-existing core.hooksPath is preserved and reported, never
clobbered, since a global hooksPath has machine-wide blast radius; and a
dispatcher reached by another path is reported ACTIVE rather than inactive,
because what matters is that the guard runs.`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(c *cobra.Command, _ []string) error {
			if dotfilesDir == "" {
				dotfilesDir = env.DotfilesDir(env.Home())
			}
			if source == "" {
				source = defaultHookSource(dotfilesDir)
			}
			return hooks.Install(c.Context(), hooks.Options{
				Source:      source,
				DotfilesDir: dotfilesDir,
				Out:         c.OutOrStdout(),
			})
		},
	}
	cmd.Flags().StringVar(&source, "source", "",
		"dispatcher tree to deploy (default: the checkout's git-hooks/, else the deploy mirror's)")
	cmd.Flags().StringVar(&dotfilesDir, "dotfiles-dir", "",
		"deploy root; the mirror lands at <dir>/git-hooks (default: $DOTFILES_DIR)")
	return cmd
}

// defaultHookSource prefers the repository checkout and falls back to the deploy
// mirror, which is what setup already does: it runs from a checkout, and the
// mirror is that checkout's copy. Resolving in this order keeps a developer
// deploying edited hooks from their working tree rather than from the last
// deployed snapshot of them.
func defaultHookSource(dotfilesDir string) string {
	if repo := env.RepoDir(); repo != "" {
		if candidate := filepath.Join(repo, "git-hooks"); dirExists(candidate) {
			return candidate
		}
	}
	return filepath.Join(dotfilesDir, "git-hooks")
}
