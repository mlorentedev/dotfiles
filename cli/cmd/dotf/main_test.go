package main

import (
	"bytes"
	"fmt"
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

func TestRunNormalError(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "testcmd",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("regular error")
		},
	}
	rootCmd.SetArgs([]string{})

	var stderr bytes.Buffer
	code := run(rootCmd, &stderr)

	if code == 0 {
		t.Errorf("expected non-zero exit code, got %d", code)
	}

	out := stderr.String()
	if !strings.HasPrefix(out, "Error: regular error") {
		t.Errorf("expected output to start with 'Error: regular error', got %q", out)
	}
}

func TestRunSilentError(t *testing.T) {
	rootCmd := &cobra.Command{
		Use:           "testcmd",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("silent error")
		},
	}
	rootCmd.SetArgs([]string{})

	var stderr bytes.Buffer
	code := run(rootCmd, &stderr)

	if code == 0 {
		t.Errorf("expected non-zero exit code, got %d", code)
	}

	out := stderr.String()
	if out != "" {
		t.Errorf("expected silent error output to be empty, got %q", out)
	}
}
