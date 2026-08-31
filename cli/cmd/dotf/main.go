package main

import (
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
	rootCmd.SilenceErrors = true // We handle printing the error

	if err := rootCmd.Execute(); err != nil {
		if errors.IsTerminalFailure(err) {
			// Print exactly the JSON latch, without the Cobra "Error: " prefix
			fmt.Fprintln(stderr, err.Error())
		} else {
			// Standard fallback printing for other errors
			fmt.Fprintf(stderr, "Error: %v\n", err)
		}
		return cmd.ExitCode(err)
	}
	return 0
}
