package main

import (
	goerrors "errors"
	"fmt"
	"io"
	"os"

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
	// Suppress Cobra's AUTOMATIC error printing — its "Error: context: ..."
	// wrapper around a TerminalFailure — while leaving the command's own stderr
	// pointed at the real one.
	//
	// This was `rootCmd.SetErr(io.Discard)`, which achieved the first half by
	// discarding EVERYTHING: `cmd.ErrOrStderr()` resolves through the root, so
	// every deliberate diagnostic any subcommand writes went to the void.
	// Measured on d4ea0f5: 16 call sites across 9 command files, including all
	// four `dotf secrets` subcommands, `harness mirror`, `harness presence` — and
	// `harness gate`, which then blocked a tool call with exit 2 and NO reason,
	// leaving the operator no way to know which skill to invoke.
	//
	// SilenceErrors on the root is the mechanism Cobra provides for exactly this:
	// ExecuteC checks the executed command's flag OR the root's before printing.
	// The per-command flag stays readable below, so the "did this command ask for
	// silence" branch is untouched.
	// Forcing the flag on the root pollutes the "did this command ask for
	// silence" read below whenever the root IS the executed command, so its
	// original value is snapshotted and restored for that one case.
	rootSilencedByAuthor := rootCmd.SilenceErrors
	rootCmd.SilenceErrors = true
	rootCmd.SetErr(stderr)

	executedCmd, err := rootCmd.ExecuteC()
	if err != nil {
		var tfe *errors.TerminalFailureError
		if goerrors.As(err, &tfe) {
			// Print exactly the JSON latch, without any Cobra "Error: " or wrapper prefixes.
			_, _ = fmt.Fprintln(stderr, tfe.Error())
		} else {
			// If the specific command didn't request silence, print the error.
			silenced := executedCmd.SilenceErrors
			if executedCmd == rootCmd {
				silenced = rootSilencedByAuthor
			}
			if !silenced {
				_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
			}
		}
		return cmd.ExitCode(err)
	}
	return 0
}
