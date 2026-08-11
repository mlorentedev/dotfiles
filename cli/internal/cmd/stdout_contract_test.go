package cmd

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureRealStreams executes the root command with the process-level os.Stdout
// and os.Stderr swapped for pipes, and deliberately does NOT call cmd.SetOut or
// cmd.SetErr.
//
// That omission is the whole point. Cobra's Command.Print/Printf/Println write
// to OutOrStderr(), and OutOrStderr() returns the writer installed by SetOut
// whenever one exists. So the execute() helper in root_test.go reports Print*
// output as "stdout" regardless of the stream the real binary writes to — it
// passes identically against a command that honours its stdout contract and one
// that does not (BUG-070 #915). Swapping at the os level is the only way to
// observe what a shell's $(...) would actually capture.
func captureRealStreams(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	outR, outW, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("os.Pipe: %v", pipeErr)
	}
	errR, errW, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("os.Pipe: %v", pipeErr)
	}

	origOut, origErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW

	// Drain concurrently: a command writing more than the pipe buffer would
	// otherwise block forever on the write.
	outCh, errCh := make(chan string, 1), make(chan string, 1)
	go func() { b, _ := io.ReadAll(outR); outCh <- string(b) }()
	go func() { b, _ := io.ReadAll(errR); errCh <- string(b) }()

	func() {
		defer func() {
			os.Stdout, os.Stderr = origOut, origErr
			_ = outW.Close()
			_ = errW.Close()
		}()
		cmd := New("dev")
		cmd.SetArgs(args)
		err = cmd.Execute()
	}()

	return <-outCh, <-errCh, err
}

// TestStdoutContracts pins the subcommands whose output is machine-read. Each
// one is consumed through a capture that only sees stdout — a shell $(...), a
// PowerShell pipeline, or a redirect — so output landing on stderr is silently
// an empty string at the call site rather than a visible failure.
func TestStdoutContracts(t *testing.T) {
	t.Setenv("VAULT_PATH", "/tmp/stdout-contract-probe")

	tests := []struct {
		name     string
		args     []string
		wantSub  string
		consumer string
	}{
		{
			name:     "env path — its own help documents ${X:-default} fallback",
			args:     []string{"env", "path", "VAULT_PATH"},
			wantSub:  "/tmp/stdout-contract-probe",
			consumer: `setup-linux.sh: HIVE_VAULT_RESOLVED="$(dotf env path HIVE_VAULT_PATH)"`,
		},
		{
			name:     "version — the installer greps this to decide idempotence",
			args:     []string{"version"},
			wantSub:  "dotf version",
			consumer: `install-dotf.sh: dotf version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := captureRealStreams(t, tt.args...)
			if err != nil {
				t.Fatalf("execute: %v (stderr=%q)", err, stderr)
			}
			if !strings.Contains(stdout, tt.wantSub) {
				t.Errorf("stdout missing %q — a caller capturing stdout gets %q.\n"+
					"consumer: %s\nstderr was: %q",
					tt.wantSub, strings.TrimSpace(stdout), tt.consumer, strings.TrimSpace(stderr))
			}
		})
	}
}

// TestEnvGenerateStdoutFlagWritesToStdout is separated because --stdout is the
// starkest case: a flag named for the stream it was not using.
func TestEnvGenerateStdoutFlagWritesToStdout(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOTFILES_DIR", dir)

	stdout, stderr, err := captureRealStreams(t, "env", "generate", "--stdout")
	if err != nil {
		t.Skipf("env generate unavailable in this environment: %v (stderr=%q)", err, stderr)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Errorf("--stdout wrote nothing to stdout; stderr was: %q", strings.TrimSpace(stderr))
	}
}
