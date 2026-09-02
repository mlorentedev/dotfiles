package cmd

import (
	"io"
	"os"
	"path/filepath"
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
	go func() {
		defer func() { _ = outR.Close() }()
		b, _ := io.ReadAll(outR)
		outCh <- string(b)
	}()
	go func() {
		defer func() { _ = errR.Close() }()
		b, _ := io.ReadAll(errR)
		errCh <- string(b)
	}()

	func() {
		defer func() {
			os.Stdout, os.Stderr = origOut, origErr
			_ = outW.Close()
			_ = errW.Close()
		}()
		cmd := New("dev", "")
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
	// `agent run` refuses on a machine with no declared identity (ADR-032 §8).
	declareIdentity(t)

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
		{
			// The resolved id is substituted straight into rendered agent
			// frontmatter. On stderr it would capture as "", and the render
			// would emit a bare `model:` — a definition naming no model, which
			// is the exact degrade model-map.json was built to prevent.
			name:     "harness resolve-tier — compile-harness substitutes this into agent frontmatter",
			args:     []string{"harness", "resolve-tier", "top", "--harness", "claude"},
			wantSub:  "opus",
			consumer: `compile-harness.sh: model_id="$(dotf harness resolve-tier "$tier" --harness "$agent")"`,
		},
		{
			// The whole point of `agent run` is to be composed: its consumer is
			// a dispatcher piping stdout into a parser, never a person reading
			// a terminal. A record on stderr would parse as empty input, which
			// jq reports as a null rather than as an error — a dispatch that
			// silently reads as "no answer" instead of failing.
			name:     "agent run — the record is piped into a JSON parser by its caller",
			args:     []string{"agent", "run", "--role", "reviewer", "--task", "probe", "--tier", "mid", "--backend", "dry-run", "--timeout", "30s"},
			wantSub:  `"status":"dry_run"`,
			consumer: `a dispatcher: dotf agent run ... | jq -r .status`,
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

// The table above pins that `version` REACHES stdout. It does not pin the
// SHAPE, and the shape is load-bearing in a way that is easy to break by
// accident: both installers regex `(\d+\.\d+\.\d+|dev)` out of the merged
// streams to decide whether to skip the install. #1158 added a commit stamp to
// this command, and the tempting place to put it was a second line of the
// default output. These cases say why it is behind a flag instead — a second
// line risks the idempotence skip that decides whether every machine reinstalls
// dotf on every setup run, and that skip failing OPEN is silent.
func TestVersionDefaultOutputStaysSingleLineForTheInstallers(t *testing.T) {
	stdout, _, err := captureRealStreams(t, "version")
	if err != nil {
		t.Fatalf("version failed: %v", err)
	}
	// Asserted EXACTLY, trailing newline included. Trimming first and counting
	// lines would let "dotf version dev\n\n\n" pass as one line, which is the
	// blank-line drift this test claims to prevent — a test that permits the
	// thing it names is the same class of defect as the check this PR adds.
	const want = "dotf version dev\n"
	if stdout != want {
		t.Fatalf("default `dotf version` must be exactly %q (the installers regex it); got %q", want, stdout)
	}
}

// The stamp is machine-read by `dotf doctor`, which feeds it straight to
// `git merge-base`. It must be the bare value with nothing else on the line —
// no label, no "commit: " prefix — or the git call receives a non-SHA.
func TestVersionCommitFlagPrintsBareValue(t *testing.T) {
	stdout, _, err := captureRealStreams(t, "version", "--commit")
	if err != nil {
		t.Fatalf("version --commit failed: %v", err)
	}
	// New("dev", "") in the harness: a source build, so the stamp is empty and
	// the command prints an empty line rather than a placeholder word that
	// would be indistinguishable from a real hash. Exact, for the same reason
	// as above: doctor feeds this value straight to `git merge-base`, so any
	// extra byte is a non-SHA handed to git.
	const want = "\n"
	if stdout != want {
		t.Errorf("--commit must print the bare stamp and nothing else, got %q", stdout)
	}
}

// TestEnvGenerateStdoutFlagWritesToStdout is separated because --stdout is the
// starkest case: a flag named for the stream it was not using.
//
// The contract is written into a temp DOTFILES_REPO_DIR rather than inherited
// from the ambient checkout, so the case cannot degrade into a skip. A skip
// here would be indistinguishable from a pass while silently testing nothing —
// the failure mode this whole file exists to close.
func TestEnvGenerateStdoutFlagWritesToStdout(t *testing.T) {
	dir := t.TempDir()
	contract := `{"env_vars":[{"name":"PROBE_DIR","required":false,` +
		`"default":{"linux":"$HOME/probe","windows":"$env:USERPROFILE\\probe"}}]}`
	if err := os.WriteFile(filepath.Join(dir, "env-contract.json"), []byte(contract), 0o600); err != nil {
		t.Fatalf("seed contract: %v", err)
	}
	t.Setenv("DOTFILES_REPO_DIR", dir)
	t.Setenv("DOTFILES_DIR", dir)

	stdout, stderr, err := captureRealStreams(t, "env", "generate", "--stdout")
	if err != nil {
		t.Fatalf("env generate --stdout failed: %v (stderr=%q)", err, strings.TrimSpace(stderr))
	}
	if !strings.Contains(stdout, "PROBE_DIR") {
		t.Errorf("--stdout did not write the rendered contract to stdout (got %q); stderr was: %q",
			strings.TrimSpace(stdout), strings.TrimSpace(stderr))
	}
}
