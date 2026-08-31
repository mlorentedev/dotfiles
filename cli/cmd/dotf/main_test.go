package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/errors"
	"github.com/spf13/cobra"
)

func TestRunTerminalFailure(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "testcmd",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.NewTerminalFailure("test terminal failure")
		},
	}
	// cobra requires arguments to execute or it defaults to os.Args
	rootCmd.SetArgs([]string{})

	var stderr bytes.Buffer
	code := run(rootCmd, &stderr)

	if code == 0 {
		t.Errorf("expected non-zero exit code, got %d", code)
	}

	out := stderr.String()
	if !strings.HasPrefix(out, errors.HandoffPrefix) {
		t.Errorf("expected output to start with %q, got %q", errors.HandoffPrefix, out)
	}
	if strings.Contains(out, "Error: ") {
		t.Errorf("expected output to not contain cobra 'Error: ' prefix, got %q", out)
	}
}
