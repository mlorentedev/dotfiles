package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/deploy"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
)

// deployRenderer is the render seam. Production calls the SAME secrets.Render
// the setup scripts called; a test injects a no-op. Deliberately a call and not
// a reimplementation — a second substitution implementation is the defect this
// command removes.
var deployRenderer = func(path string) error {
	reg, err := loadRegistry()
	if err != nil {
		return err
	}
	_, err = secrets.Render(path, reg, secretLoader(), env.Home())
	return err
}

// newDeployCmd installs agent configs from the checkout to their deployed
// locations (CLI-039, #1023).
//
// The behaviour is one implementation for every OS. It used to be two — one in
// setup-linux.sh and one in setup-windows.ps1, per config — which is the twin
// shape ADR-020's strangler-fig rule says to collapse the next time it is
// touched. The setups now call this and get shorter.
func newDeployCmd() *cobra.Command {
	var dryRun bool
	c := &cobra.Command{
		Use:   "deploy [name]",
		Short: "Install agent configs from the checkout to their deployed locations",
		Long: "deploy installs the agent configs declared in ai/deploy.json: it stages the\n" +
			"source, substitutes {env:VAR} placeholders through `dotf secrets render` when\n" +
			"the config asks for it, compares against the installed copy, and installs\n" +
			"atomically with the declared permissions.\n\n" +
			"A config already in sync is reported and NOT rewritten, so a setup run does\n" +
			"not churn mtimes and \"did this change?\" stays answerable.\n\n" +
			"  dotf deploy              # every declared config\n" +
			"  dotf deploy pi           # one\n" +
			"  dotf deploy --dry-run    # report what would change, touch nothing",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot := env.RepoDir()
			if repoRoot == "" {
				return fmt.Errorf("cannot locate the dotfiles checkout — set DOTFILES_REPO_DIR or run from inside it")
			}
			manifestPath := filepath.Join(repoRoot, filepath.FromSlash(deploy.ManifestRel))
			raw, err := os.ReadFile(manifestPath) //nolint:gosec // repo-relative, fixed name
			if err != nil {
				return fmt.Errorf("reading %s: %w", deploy.ManifestRel, err)
			}
			man, err := deploy.ParseManifest(raw)
			if err != nil {
				return err
			}

			targets := man.Configs
			if len(args) == 1 {
				c := man.Lookup(args[0])
				if c == nil {
					return fmt.Errorf("%w: %q (declared: %s)", deploy.ErrNoSuchConfig, args[0], names(man))
				}
				targets = []deploy.Config{*c}
			}

			w := cmd.OutOrStdout()
			for _, target := range targets {
				res, err := deploy.Deploy(target, repoRoot, env.Home(), env.ResolvePath, deployRenderer, dryRun)
				if err != nil {
					return err
				}
				switch {
				case !res.Changed:
					_, _ = fmt.Fprintf(w, "in sync   %-10s %s\n", res.Name, res.Dst)
				case dryRun:
					_, _ = fmt.Fprintf(w, "would deploy %-7s %s\n", res.Name, res.Dst)
				default:
					_, _ = fmt.Fprintf(w, "deployed  %-10s %s\n", res.Name, res.Dst)
				}
			}
			return nil
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "report what would change without writing anything")
	return c
}

func names(m *deploy.Manifest) string {
	out := make([]string, 0, len(m.Configs))
	for _, c := range m.Configs {
		out = append(out, c.Name)
	}
	if len(out) == 0 {
		return "none"
	}
	return fmt.Sprint(out)
}
