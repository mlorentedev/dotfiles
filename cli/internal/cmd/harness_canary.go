package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
	"github.com/mlorentedev/dotfiles/cli/internal/nanprobe"
)

// Exit statuses of `dotf harness canary`. The workflow branches on them: 1
// edits the canary's issue, 2 fails the run without touching it, because a
// canary that could not look has nothing to say about the models.
const (
	canaryExitModels = 1
	canaryExitSetup  = 2
)

var errCanaryModels = errors.New("at least one bound NaN model needs attention (see the report)")

// newHarnessCanaryCmd probes every NaN model this repository binds, with no PR
// involved (AI-045 AC8, #1860). The scheduled workflow model-canary.yml runs
// it daily; anyone can run it locally, where it also reads the deployed pin
// sites under $HOME.
func newHarnessCanaryCmd() *cobra.Command {
	var (
		repoRoot string
		report   string
		timeout  time.Duration
	)
	cmd := &cobra.Command{
		Use:   "canary",
		Short: "Send one minimal call to every bound NaN model and report the ones that do not answer",
		Long: `canary collects every NaN model bound by harness/model-map.json and by the pin
sites harness/model-pins.json declares, then sends each one a minimal chat
completion, one at a time. A model that does not answer is probed a second
time before it counts.

Each result is classed by what it asks of the reader: refused (401/403/404,
replace the model in the files listed), quota (402/429, nothing to fix) or
unavailable (no completed reply on two passes). Only a completed reply counts
as an answer; a status line whose body never arrives does not.

The key is read from NAN_API_KEY and never printed. NAN_API_BASE overrides
the endpoint. Locally: dotf secrets run --only NAN_API_KEY -- dotf harness canary

Exit status: 0 every model answered or is only over quota; 1 a model was
refused or stayed unavailable; 2 the canary could not run.`,
		Example: `  dotf secrets run --only NAN_API_KEY -- dotf harness canary
  dotf harness canary --report "$RUNNER_TEMP/canary.md"   # from the workflow`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if repoRoot == "" {
				repoRoot = env.RepoDir()
			}
			if repoRoot == "" {
				return withExitCode(canaryExitSetup, fmt.Errorf("cannot locate the dotfiles checkout: pass --repo-root, set DOTFILES_REPO_DIR, or run from inside it"))
			}
			key := os.Getenv("NAN_API_KEY")
			if key == "" {
				return withExitCode(canaryExitSetup, fmt.Errorf("NAN_API_KEY is not set: run it under `dotf secrets run --only NAN_API_KEY --`"))
			}
			base := os.Getenv("NAN_API_BASE")
			if base == "" {
				base = nanprobe.DefaultBaseURL
			}

			m, err := harness.LoadModelMap(repoRoot)
			if err != nil {
				return withExitCode(canaryExitSetup, err)
			}
			pins, err := harness.LoadModelPins(repoRoot)
			if err != nil {
				return withExitCode(canaryExitSetup, err)
			}
			bindings, notes, err := nanprobe.Collect(m, pins, repoRoot, env.Home(), "nan")
			if err != nil {
				return withExitCode(canaryExitSetup, err)
			}
			if len(bindings.Files) == 0 {
				// A canary that probes nothing must not report "all answered".
				return withExitCode(canaryExitSetup, fmt.Errorf("no NaN model is bound anywhere: the map or the registry changed shape"))
			}

			prober := nanprobe.Prober{Client: &http.Client{}, BaseURL: base, Key: key, Timeout: timeout}
			results := prober.ProbeAll(cmd.Context(), bindings.Targets())

			if err := nanprobe.WriteMarkdown(cmd.OutOrStdout(), results, bindings, notes); err != nil {
				return withExitCode(canaryExitSetup, err)
			}
			if report != "" {
				f, err := os.Create(report) // #nosec G304 -- the caller names its own report file
				if err != nil {
					return withExitCode(canaryExitSetup, err)
				}
				werr := nanprobe.WriteMarkdown(f, results, bindings, notes)
				if cerr := f.Close(); werr == nil {
					werr = cerr
				}
				if werr != nil {
					return withExitCode(canaryExitSetup, werr)
				}
			}
			if nanprobe.Failing(results) {
				return withExitCode(canaryExitModels, errCanaryModels)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "checkout holding harness/model-map.json and harness/model-pins.json (default: DOTFILES_REPO_DIR or the cwd walk-up)")
	cmd.Flags().StringVar(&report, "report", "", "also write the Markdown report to this file")
	cmd.Flags().DurationVar(&timeout, "timeout", 90*time.Second, "how long one model may take to complete its reply")
	return cmd
}
