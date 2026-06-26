package cmd

import (
	"fmt"
	"os"

	"github.com/mlorentedev/dotfiles/cli/internal/initrepo"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
)

// ghSecretSetter is the GitHub Actions write seam (GHSecretSet in production), overridable
// so command tests inject a fake with no gh, no network, no secrets. repoOriginResolver
// derives the current repo's origin slug for the --repo default; a var so tests bypass git.
var (
	ghSecretSetter     secrets.GitHubSecretSetter = secrets.GHSecretSet{}
	repoOriginResolver                            = defaultOriginRepo
)

// defaultOriginRepo resolves the current working directory's git origin to an owner/name
// slug (git -C cwd walks up to the repo root, so any subdir works).
func defaultOriginRepo() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return initrepo.OriginRepo(cwd)
}

// newSecretsSyncCmd is the `dotf secrets sync <target>` noun (ADR-029): backend-agnostic,
// deploy-time materialization of a scoped secret set for headless consumers. This slice
// ships the `ci` target only; container/agent are designed but deferred.
func newSecretsSyncCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "sync",
		Short: "Materialize a scoped secret set for a headless consumer (CI, …)",
		Long: "sync resolves a scoped secret set backend-agnostically (age|bw, via the same\n" +
			"Loader as `run`) and pushes it to a headless consumer's delivery surface, ahead\n" +
			"of time — the consumer never talks to Bitwarden at runtime (ADR-028/ADR-029).\n" +
			"This slice implements the `ci` target (GitHub Actions secrets).",
	}
	c.AddCommand(newSecretsSyncCiCmd())
	return c
}

// newSecretsSyncCiCmd implements `dotf secrets sync ci [--repo OWNER/REPO] [--dry-run]`:
// select the repo's ci:<repo> secrets, resolve each value via the Loader (age or bw,
// transparently), and upload via the gh seam. Idempotent (`gh secret set` overwrites);
// --dry-run reports VAR→repo with byte lengths and never a value, and never uploads.
func newSecretsSyncCiCmd() *cobra.Command {
	var repo string
	var dryRun bool
	c := &cobra.Command{
		Use:   "ci",
		Short: "Push a repo's ci:* secrets to its GitHub Actions secrets (age|bw agnostic)",
		Long: "ci selects every registry secret whose consumers contains ci:<repo>, resolves\n" +
			"each value backend-agnostically, and uploads it to the repo's GitHub Actions\n" +
			"secrets via `gh secret set`. File, floor/offline, and GITHUB_*-prefixed secrets\n" +
			"are excluded with a reason. --repo defaults to the current repo's origin.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			reg, err := loadRegistry()
			if err != nil {
				return err
			}
			if repo == "" {
				repo, err = repoOriginResolver()
				if err != nil {
					return fmt.Errorf("--repo not given and could not derive it: %w\n"+
						"pass --repo owner/name", err)
				}
			}
			if !initrepo.ValidRepoSlug(repo) {
				return fmt.Errorf("invalid --repo %q: want owner/name", repo)
			}

			sel := reg.SelectCI(repo)
			out := cmd.OutOrStdout()
			for _, sk := range sel.Skipped {
				_, _ = fmt.Fprintf(out, "skip %s: %s\n", sk.ID, sk.Reason)
			}
			if len(sel.Upload) == 0 {
				_, _ = fmt.Fprintf(out, "no ci secrets selected for %s\n", repo)
				return nil
			}

			// One Loader path resolves both age and bw entries — the backend-agnostic
			// boundary ADR-029 introduces. Fail-fast: a resolution error aborts before any
			// upload, so the repo is never left with a partial secret set.
			env, err := secretLoader().EnvFor(sel.Upload, nil)
			if err != nil {
				return err
			}
			for i, kv := range env {
				name := sel.Upload[i].Var
				value := kv[len(name)+1:] // EnvFor returns "name=value"
				if dryRun {
					_, _ = fmt.Fprintf(out, "would set %s → %s (%d bytes)\n", name, repo, len(value))
					continue
				}
				if err := ghSecretSetter.SetSecret(repo, name, value); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(out, "set %s → %s\n", name, repo)
			}
			return nil
		},
	}
	c.Flags().StringVar(&repo, "repo", "", "target repo owner/name (default: current repo's origin)")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "report VAR→repo without uploading (names + byte lengths, never values)")
	return c
}
