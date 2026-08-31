package main

import (
	"fmt"
	"io"
	"os"
	goerrors "errors"

	"github.com/mlorentedev/dotfiles/cli/internal/cmd"
	"github.com/mlorentedev/dotfiles/cli/internal/errors"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := cmd.New(version)
	os.Exit(run(rootCmd, os.Stderr))
}

func run(rootCmd *cobra.Command, stderr io.Writer) int {
	// We divert Cobra's automatic error printing to io.Discard.
	// This prevents Cobra from printing wrapped TerminalFailure errors with "Error: context: ...",
	// while keeping the subcommand's SilenceErrors property perfectly intact so we can read it.
	rootCmd.SetErr(io.Discard)

	executedCmd, err := rootCmd.ExecuteC()
	if err != nil {
		var tfe *errors.TerminalFailureError
		if goerrors.As(err, &tfe) {
			// Print exactly the JSON latch, without any Cobra "Error: " or wrapper prefixes.
			_, _ = fmt.Fprintln(stderr, tfe.Error())
		} else {
			// If the specific command didn't request silence, print the error.
			if !executedCmd.SilenceErrors {
				_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
			}
		}
		return cmd.ExitCode(err)
	}
	return 0
}
