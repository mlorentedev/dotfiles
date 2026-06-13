package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/review"
)

func newReviewCmd() *cobra.Command {
	var (
		providerName string
		model        string
		maxBytes     int64
		timeout      time.Duration
	)

	cmd := &cobra.Command{
		Use:   "review",
		Short: "Cross-model code review of a diff read from stdin",
		Long: `Reads a unified diff from stdin and asks a non-Claude model for a
decorrelated second-opinion review, printed to stdout as markdown.

Providers and required environment:
  nan         NAN_BASE_URL + NAN_API_KEY   (default model deepseek-v4-flash)
  openrouter  OPENROUTER_API_KEY           (default model deepseek/deepseek-chat)

Exit codes: 0 review produced; 1 on empty stdin, missing env, oversized diff,
HTTP error or timeout.

Privacy: the diff is sent to the selected third-party API. Think before piping
diffs from repositories you do not own.`,
		Example:      "  git diff main...HEAD | dot review --provider nan",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			diff, err := review.ReadDiff(cmd.InOrStdin(), maxBytes)
			if err != nil {
				return err
			}
			prov, err := review.Resolve(providerName)
			if err != nil {
				return err
			}
			if model == "" {
				model = prov.DefaultModel
			}
			out, err := review.Request(prov, model, diff, timeout)
			if err != nil {
				return err
			}
			cmd.Println(out)
			return nil
		},
	}

	cmd.Flags().StringVar(&providerName, "provider", "nan", "review provider: nan or openrouter")
	cmd.Flags().StringVar(&model, "model", "", "model override (default: provider-specific)")
	cmd.Flags().Int64Var(&maxBytes, "max-bytes", 200_000, "fail if the diff exceeds this size instead of reviewing it truncated")
	cmd.Flags().DurationVar(&timeout, "timeout", 120*time.Second, "HTTP timeout for the provider request")
	return cmd
}
