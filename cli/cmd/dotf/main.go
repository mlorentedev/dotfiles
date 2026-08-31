package main

import (
	"fmt"
	"os"

	"github.com/mlorentedev/dotfiles/cli/internal/cmd"
	"github.com/mlorentedev/dotfiles/cli/internal/errors"
)

var version = "dev"

func main() {
	rootCmd := cmd.New(version)
	rootCmd.SilenceErrors = true // We handle printing the error

	if err := rootCmd.Execute(); err != nil {
		if errors.IsTerminalFailure(err) {
			// Print exactly the JSON latch, without the Cobra "Error: " prefix
			fmt.Fprintln(os.Stderr, err.Error())
		} else {
			// Standard fallback printing for other errors
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(cmd.ExitCode(err))
	}
}
