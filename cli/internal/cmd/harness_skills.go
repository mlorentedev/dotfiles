package cmd

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// newHarnessSkillsCmd deploys the committed skill records: the Go form of
// compile-harness.sh's deploy_skills and of Deploy-SkillRecord in
// setup-windows.ps1, which both move onto it once a release carries it (CLI-026).
func newHarnessSkillsCmd() *cobra.Command {
	var (
		repoRoot, home string
		dryRun         bool
	)
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Render each committed skill record to every skills.deploy target",
		Long: "skills renders every harness/skills/<name>/SKILL.md record to each\n" +
			"harness/manifest.json skills.deploy[] target under $HOME, as a skill directory\n" +
			"(with the record's auxiliary files), an opencode command or an agy prompt. A\n" +
			"record's targets: list is matched by exact harness name; an absent one means\n" +
			"every harness. An output that already holds its render is not rewritten. A\n" +
			"target whose requires_command is not on PATH is skipped. Generated outputs whose\n" +
			"record is gone, or no longer targets that harness, are pruned; an unmarked copy\n" +
			"of a managed skill is reported and kept. Last, the skills.catalog file (copilot's\n" +
			"instructions) gets one bullet per record that targets its harness, written into\n" +
			"its one GENERATED region; a file that is absent or has no such region is skipped.",
		Example:      "  dotf harness skills --dry-run\n  dotf harness skills --repo-root /path/to/checkout",
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
			has := func(name string) bool { _, err := exec.LookPath(name); return err == nil }
			outcomes, err := harness.DeploySkills(repoRoot, home, has, dryRun)
			reportSkillOutcomes(cmd.OutOrStdout(), cmd.ErrOrStderr(), outcomes, dryRun)
			return err
		},
	}
	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "checkout holding harness/manifest.json (default: DOTFILES_REPO_DIR or the cwd walk-up)")
	cmd.Flags().StringVar(&home, "home", "", "directory the manifest's dirs are relative to (default: the user's home)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report what would be deployed or pruned and write nothing")
	return cmd
}

// reportSkillOutcomes prints one line per output that changed or needs a
// look, then a count, so a converged run reads as "0 deployed".
func reportSkillOutcomes(w, ew io.Writer, outcomes []harness.SkillOutcome, dryRun bool) {
	verb := map[bool]string{false: "", true: "would be "}[dryRun]
	counts := map[string]int{}
	for _, o := range outcomes {
		if o.Catalog {
			reportCatalogOutcome(w, ew, o, verb)
			continue
		}
		counts[o.Status]++
		switch o.Status {
		case harness.SkillDeployed:
			_, _ = fmt.Fprintf(w, "[deploy] skill %s-> %s\n", verb, o.Path)
		case harness.SkillPruned:
			_, _ = fmt.Fprintf(w, "[deploy] %spruned stale -> %s\n", verb, o.Path)
		case harness.SkillSkipped:
			_, _ = fmt.Fprintf(w, "[deploy] skill target %s skipped: %s\n", o.Agent, o.Reason)
		case harness.SkillUnmanaged:
			_, _ = fmt.Fprintf(ew, "[deploy] WARN unmanaged copy of a managed skill (no provenance marker, not prunable) -> %s\n", o.Path)
		}
	}
	_, _ = fmt.Fprintf(w, "[deploy] skills: %d %sdeployed, %d current, %d %spruned\n",
		counts[harness.SkillDeployed], verb, counts[harness.SkillCurrent], counts[harness.SkillPruned], verb)
}

// reportCatalogOutcome prints the catalog's line in the shell's words; it is
// not counted with the skill outputs.
func reportCatalogOutcome(w, ew io.Writer, o harness.SkillOutcome, verb string) {
	switch o.Status {
	case harness.SkillDeployed:
		_, _ = fmt.Fprintf(w, "[deploy] catalog %s-> %s\n", verb, o.Path)
	case harness.SkillSkipped:
		_, _ = fmt.Fprintf(ew, "[deploy] %s, skipping: %s\n", o.Reason, o.Path)
	}
}
