package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/vm"
)

const syntheticSecret = "s3cr3t-synthetic-value"

// echoSecretGuest points the verbs at one held session, x, whose guest prints
// the first line of its stdin, and at a resolver of a synthetic value.
func echoSecretGuest(t *testing.T) {
	t.Helper()
	noGuest(t)
	dir := filepath.Join(t.TempDir(), "sessions")
	if err := os.MkdirAll(filepath.Join(dir, "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	oldS, oldR := linuxSessions, execSecrets
	t.Cleanup(func() { linuxSessions, execSecrets = oldS, oldR })
	linuxSessions = func() (vm.LinuxSessions, error) {
		return vm.LinuxSessions{Dir: dir, Now: time.Now, L: func(_ context.Context, in io.Reader, out, _ io.Writer, args ...string) error {
			if args[0] == "list" {
				_, err := io.WriteString(out, `{"name":"dotf-x","status":"Running"}`+"\n")
				return err
			}
			_, err := io.Copy(out, in)
			return err
		}}, nil
	}
	execSecrets = func(ids []string) (func() ([]string, error), error) {
		return func() ([]string, error) { return []string{"API_TOKEN=" + syntheticSecret}, nil }, nil
	}
}

// AC7: masked in the record and on plain stdout; AC5: with --json stdout is one
// record, without it stdout is exactly the guest's output.
func TestVMExecMasksAForwardedSecretOnBothOutputs(t *testing.T) {
	echoSecretGuest(t)
	stdout, _, err := runVerb(t, "exec", "--session", "x", "--secret", "api-token", "--json", "--", "env")
	var rec vm.Record
	if jerr := json.Unmarshal([]byte(stdout), &rec); err != nil || jerr != nil {
		t.Fatalf("exec --json: stdout=%q err=%v jerr=%v", stdout, err, jerr)
	}
	if strings.Contains(stdout, syntheticSecret) || rec.Stdout != "[REDACTED:API_TOKEN]\n" {
		t.Errorf("record leaks or lost the output: %q", stdout)
	}

	stdout, _, err = runVerb(t, "exec", "--session", "x", "--secret", "api-token", "--", "env")
	if err != nil || stdout != "[REDACTED:API_TOKEN]\n" {
		t.Errorf("plain exec: stdout=%q err=%v, want only the guest's masked output", stdout, err)
	}
}

// ADR-055: a guest command that exits non-zero exits 1; its code is in the record.
func TestVMExecGuestFailureExitsOne(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs sh")
	}
	noGuest(t)
	dir := filepath.Join(t.TempDir(), "sessions")
	if err := os.MkdirAll(filepath.Join(dir, "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	old := linuxSessions
	t.Cleanup(func() { linuxSessions = old })
	failing := exec.Command("sh", "-c", "exit 7").Run()
	linuxSessions = func() (vm.LinuxSessions, error) {
		return vm.LinuxSessions{Dir: dir, Now: time.Now, L: func(_ context.Context, _ io.Reader, out, _ io.Writer, args ...string) error {
			if args[0] == "list" {
				_, err := io.WriteString(out, `{"name":"dotf-x","status":"Running"}`+"\n")
				return err
			}
			return failing
		}}, nil
	}
	stdout, _, err := runVerb(t, "exec", "--session", "x", "--json", "--", "false")
	var rec vm.Record
	if jerr := json.Unmarshal([]byte(stdout), &rec); jerr != nil {
		t.Fatal(jerr)
	}
	if ExitCode(err) != 1 || rec.Status != vm.StatusOK || rec.Exit == nil || *rec.Exit != 7 {
		t.Errorf("exit %d, record %+v; want 1, ok, exit 7", ExitCode(err), rec)
	}
}

// Killing limactl leaves its ssh child holding the pipes (measured on Lima
// 2.2.1): a timeout must come back anyway, not wait for that child.
func TestExecLimactlReturnsWhenAChildHoldsThePipes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs sh")
	}
	dir := t.TempDir()
	fake := "#!/bin/sh\nsleep 30 &\nwait\n"
	if err := os.WriteFile(filepath.Join(dir, "limactl"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	old := limactlGrace
	t.Cleanup(func() { limactlGrace = old })
	limactlGrace = 50 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		var out strings.Builder
		done <- execLimactl(ctx, strings.NewReader(""), &out, &out, "shell")
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Error("a killed limactl reported success")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("execLimactl waited on the orphan's pipes")
	}
}

// A secret that cannot travel on one line is refused before anything resolves.
func TestExecSecretsRefusesWhatIsNotOneValue(t *testing.T) {
	_, err := resolveExecSecrets([]string{"no-such-secret"})
	if err == nil || !strings.Contains(err.Error(), "no-such-secret") {
		t.Errorf("err = %v, want the unknown id named", err)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Errorf("err = %v, want the catalog's refusal, not a missing registry", err)
	}
}
