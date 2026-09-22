package cmd

import (
	"fmt"
	"os"

	"github.com/mlorentedev/dotfiles/cli/internal/initrepo"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
)

// ghSecretSetter is the GitHub Actions write seam (GHSecretSet in production), overridable
// so command tests inject a fake with no gh, no network, no secrets. ghTokenValidator is
// the liveness seam for entries marked `validate: github-token` (GHTokenValidate in
// production), so sync refuses to upload a dead PAT. repoOriginResolver derives the
// current repo's origin slug for the --repo default; a var so tests bypass git.
var (
	ghSecretSetter     secrets.GitHubSecretSetter   = secrets.GHSecretSet{}
	ghTokenValidator   secrets.GitHubTokenValidator = secrets.GHTokenValidate{}
	repoOriginResolver                              = defaultOriginRepo
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
	var skipVerify bool
	c := &cobra.Command{
		Use:   "ci [SECRET_NAME...]",
		Short: "Push a repo's ci:* secrets to its GitHub Actions secrets (age|bw agnostic)",
		Long: "ci selects every registry secret whose consumers contains ci:<repo>, resolves\n" +
			"each value backend-agnostically, and uploads it to the repo's GitHub Actions\n" +
			"secrets via `gh secret set`. File, floor/offline, and GITHUB_*-prefixed secrets\n" +
			"are excluded with a reason. --repo defaults to the current repo's origin.\n\n" +
			"Naming secrets scopes the push to exactly those GitHub secret names (the env\n" +
			"vars), and a name the repo's set does not contain fails before any upload.",
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, names []string) error {
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
			if sel.Upload, err = scopeUpload(sel.Upload, names, repo); err != nil {
				return err
			}
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

			// Pre-upload liveness gate: an entry marked `validate: github-token` must
			// authenticate before ANY upload, so a dead/expired PAT is never pushed to
			// Actions (the BITACORA_PAT incident — a redeploy refreshed updated_at on a
			// 401 token). Opt-in per entry; liveness can't be probed generically across
			// providers, so unmarked secrets are untouched. --skip-verify bypasses.
			if !skipVerify {
				for i, kv := range env {
					e := sel.Upload[i]
					if e.Validate != "github-token" {
						continue
					}
					name := e.Var
					value := kv[len(name)+1:]
					if err := ghTokenValidator.Validate(value); err != nil {
						return fmt.Errorf("%s failed its github-token liveness check — refusing to upload a dead "+
							"credential (rotate it, or re-run with --skip-verify): %w", name, err)
					}
					_, _ = fmt.Fprintf(out, "verified %s (live github token)\n", name)
				}
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
	c.Flags().BoolVar(&skipVerify, "skip-verify", false, "skip the github-token liveness check for entries marked validate: github-token")
	return c
}

// scopeUpload narrows a repo's CI selection to the named GitHub secrets, or keeps it
// whole when none are named.
//
// Scoped by the secret's NAME — the env var — never by registry id. One entry can
// expose several vars (NAN_API_KEY also exposes HIVE_WORKER_API_KEY), and a scope by
// id would push all of them to a repo whose workflows read one: a second copy of a
// credential where nothing uses it. Every name must be in the selection, checked
// before anything resolves or uploads, so a typo cannot read as "synced".
func scopeUpload(upload []secrets.Entry, names []string, repo string) ([]secrets.Entry, error) {
	if len(names) == 0 {
		return upload, nil
	}
	byVar := make(map[string]secrets.Entry, len(upload))
	for _, e := range upload {
		byVar[e.Var] = e
	}
	scoped := make([]secrets.Entry, 0, len(names))
	for _, n := range names {
		e, ok := byVar[n]
		if !ok {
			return nil, fmt.Errorf("%s is not among %s's ci secrets; nothing uploaded (drop it, or add ci:%s to its consumers)", n, repo, repo)
		}
		scoped = append(scoped, e)
	}
	return scoped, nil
}
