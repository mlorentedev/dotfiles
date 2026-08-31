package main

import (
	"io"
	"os"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/cmd"
	"github.com/mlorentedev/dotfiles/cli/internal/errors"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := cmd.New(version)
	os.Exit(run(rootCmd, os.Stderr))
}

type errInterceptor struct {
	target io.Writer
}

func (w *errInterceptor) Write(p []byte) (n int, err error) {
	s := string(p)
	// If Cobra is printing our TerminalFailure, it prefixes it with "Error: ".
	// We strip that prefix so the JSON latch is the raw output.
	if strings.HasPrefix(s, "Error: "+errors.HandoffPrefix) {
		s = strings.TrimPrefix(s, "Error: ")
		return w.target.Write([]byte(s))
	}
	return w.target.Write(p)
}

func run(rootCmd *cobra.Command, stderr io.Writer) int {
	// Intercept Cobra's error output to strip the "Error: " prefix for TerminalFailures,
	// while naturally respecting every subcommand's SilenceErrors configuration.
	rootCmd.SetErr(&errInterceptor{target: stderr})

	if err := rootCmd.Execute(); err != nil {
		return cmd.ExitCode(err)
	}
	return 0
}
