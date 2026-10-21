package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// newSpecDigestCmd prints the contract digests review-request.json records.
// They are taken over a normalised form, so a raw sha256sum of the same files
// differs whenever a box was ticked or the status moved, and reads as drift
// that is not there (#2684).
func newSpecDigestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "digest <spec-id>",
		Short: "Print the contract digests a review request records, for comparing against one",
		Long: "digest prints the normalised digests of the spec's contract files in the\n" +
			"shape review-request.json records them. Compare a recorded request with this\n" +
			"output, not with sha256sum: the digest ignores ticked boxes, the status line and\n" +
			"line endings, so a raw hash differs from it without the contract changing.",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, specDir, err := existingSpecDir(args[0])
			if err != nil {
				return err
			}
			out, err := json.MarshalIndent(struct {
				ContractDigests map[string]string `json:"contract_digests"`
			}{spec.ContractDigests(specDir)}, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return err
		},
	}
}
