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

func TestRunWrappedTerminalFailure(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "testcmd",
		RunE: func(cmd *cobra.Command, args []string) error {
			tfe := errors.NewTerminalFailure("test terminal failure")
			return fmt.Errorf("some wrapper context: %w", tfe)
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
	if strings.Contains(out, "some wrapper context") {
		t.Errorf("expected output to not contain wrapper text, got %q", out)
	}
}

// TestRunDoesNotDiscardDeliberateStderr is the guard for a regression that
// silenced nine command files at once.
//
// `run` used to do `rootCmd.SetErr(io.Discard)` to suppress Cobra's automatic
// "Error: ..." wrapper. But `cmd.ErrOrStderr()` resolves through the root, so it
// discarded every diagnostic any subcommand deliberately wrote — measured on
// d4ea0f5: 17 call sites across 9 files, all four `dotf secrets` subcommands
// among them. The sharpest case was `dotf harness gate`, which then blocked a
// tool call with exit 2 and NOTHING on stderr, leaving the operator no way to
// learn which skill to invoke: an enforcement mechanism whose refusal is
// indistinguishable from a crash.
//
// The two concerns share a sink, so they cannot be separated by writer. Cobra's
// SilenceErrors is the mechanism for the first, and this pins that using it did
// not cost the second.
func TestRunDoesNotDiscardDeliberateStderr(t *testing.T) {
	const diagnostic = "[cmd] a deliberate diagnostic"

	rootCmd := &cobra.Command{
		Use: "testcmd",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), diagnostic)
			return nil
		},
	}
	rootCmd.SetArgs([]string{})

	var stderr bytes.Buffer
	if code := run(rootCmd, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stderr.String(), diagnostic) {
		t.Errorf("a command's deliberate stderr write was discarded; got %q.\n"+
			"Every `dotf` subcommand that explains itself on stderr depends on this.", stderr.String())
	}
}

// A command that writes a diagnostic AND fails must still get both out, with no
// duplicated Cobra wrapper.
func TestRunKeepsDiagnosticsAlongsideASilencedError(t *testing.T) {
	rootCmd := &cobra.Command{
		Use:           "testcmd",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "[cmd] why it failed")
			return fmt.Errorf("boom")
		},
	}
	rootCmd.SetArgs([]string{})

	var stderr bytes.Buffer
	if code := run(rootCmd, &stderr); code == 0 {
		t.Fatal("expected a non-zero exit")
	}
	out := stderr.String()
	if !strings.Contains(out, "[cmd] why it failed") {
		t.Errorf("the diagnostic was discarded: %q", out)
	}
	if strings.Contains(out, "Error: boom") {
		t.Errorf("a SilenceErrors command must not get Cobra's wrapper: %q", out)
	}
}
