package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// newHarnessDoctrineCmd deploys the compact doctrine payload: the Go form of
// compile-harness.sh's deploy_doctrine, which moves onto it once a release
// carries it (CLI-026).
func newHarnessDoctrineCmd() *cobra.Command {
	var (
		repoRoot, home string
		dryRun         bool
	)
	cmd := &cobra.Command{
		Use:   "doctrine",
		Short: "Write the compact doctrine payload into each capped harness's rules file",
		Long: "doctrine builds, for each harness/manifest.json doctrine.deploy[] target, the\n" +
			"compact payload: the enforced records doctrine.inject names, without their\n" +
			"provenance notice or full-only regions, then that harness's persona roster.\n" +
			"A target with a char_cap gets it folded to ASCII. The payload replaces the\n" +
			"GENERATED region of the rules file under $HOME, or is appended as one; a file\n" +
			"that does not exist is started. Everything else in the file is kept. A file\n" +
			"that shadows the target, non-ASCII the fold does not know, and a breached cap\n" +
			"are reported as warnings.",
		Example:      "  dotf harness doctrine --dry-run\n  dotf harness doctrine --repo-root /path/to/checkout",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if repoRoot == "" {
				repoRoot = env.RepoDir()
			}
			if repoRoot == "" {
				return fmt.Errorf("cannot locate the dotfiles checkout — pass --repo-root, set DOTFILES_REPO_DIR, or run from inside it")
			}
			if home == "" {
				home = env.Home()
			}
			outcomes, err := harness.DeployDoctrine(repoRoot, home, dryRun)
			reportDoctrineOutcomes(cmd.OutOrStdout(), cmd.ErrOrStderr(), outcomes, dryRun)
			return err
		},
	}
	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "checkout holding harness/manifest.json (default: DOTFILES_REPO_DIR or the cwd walk-up)")
	cmd.Flags().StringVar(&home, "home", "", "directory the manifest's files are relative to (default: the user's home)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report what would be written and write nothing")
	return cmd
}

// reportDoctrineOutcomes prints a line per rules file started or written, and
// each warning on stderr.
func reportDoctrineOutcomes(w, ew io.Writer, outcomes []harness.DoctrineOutcome, dryRun bool) {
	verb := map[bool]string{false: "", true: "would be "}[dryRun]
	for _, o := range outcomes {
		if o.Created {
			_, _ = fmt.Fprintf(w, "[deploy] doctrine: %screated %s\n", verb, o.Path)
		}
		if o.Status == harness.SkillDeployed {
			_, _ = fmt.Fprintf(w, "[deploy] doctrine %s-> %s\n", verb, o.Path)
		} else {
			_, _ = fmt.Fprintf(w, "[deploy] doctrine current: %s\n", o.Path)
		}
		for _, warn := range o.Warnings {
			_, _ = fmt.Fprintf(ew, "[deploy] WARN %s\n", warn)
		}
	}
}
