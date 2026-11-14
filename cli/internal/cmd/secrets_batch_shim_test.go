package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// shimOnPath puts an executable file named name first on PATH.
func shimOnPath(t *testing.T, name string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("@echo off\r\n"), 0o755); err != nil { //nolint:gosec // a test fixture that must be executable for LookPath
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// `dotf secrets run -- pi` where pi is an npm .cmd shim hands the child's
// arguments to cmd.exe, which re-parses % ^ & | < > " (BatBadBut, #3440). The
// refusal is the one `lane exec` makes, and it comes before the store is read.
func TestSecretsRun_RefusesABatchShimChildWhereWindowsRunsItThroughCmdExe(t *testing.T) {
	prev := batchShimsRunViaCmdExe
	t.Cleanup(func() { batchShimsRunViaCmdExe = prev })
	batchShimsRunViaCmdExe = true

	for _, name := range []string{"fakeagent.cmd", "fakeagent.bat", "fakeagent.CMD"} {
		t.Run(name, func(t *testing.T) {
			shimOnPath(t, name)
			err := assertSafeChildCommand([]string{name, "fix it & echo x"})
			if err == nil {
				t.Fatal("want a refusal")
			}
			for _, want := range []string{"cmd.exe", name, "native"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not name %q", err, want)
				}
			}
			if strings.HasPrefix(err.Error(), "lane:") {
				t.Errorf("a refusal from secrets run must not speak as lane: %q", err)
			}
		})
	}

	t.Run("through the command, before anything is decrypted", func(t *testing.T) {
		shimOnPath(t, "fakeagent.cmd")
		_, _, err := execute(t, "secrets", "run", "--", "fakeagent.cmd", "x")
		if err == nil || !strings.Contains(err.Error(), "cmd.exe") {
			t.Fatalf("want the batch-shim refusal, got %v", err)
		}
	})
}

// Only where Windows runs a .cmd through cmd.exe: a file with that name is an
// ordinary executable on Linux and macOS, and a command that cannot be found is
// the launch's error to report, not this check's.
func TestSecretsRun_AllowsWhatCmdExeNeverSees(t *testing.T) {
	prev := batchShimsRunViaCmdExe
	t.Cleanup(func() { batchShimsRunViaCmdExe = prev })

	shimOnPath(t, "fakeagent.cmd")
	batchShimsRunViaCmdExe = false
	if err := assertSafeChildCommand([]string{"fakeagent.cmd", "x"}); err != nil {
		t.Errorf("off Windows a .cmd file is no shim: %v", err)
	}
	batchShimsRunViaCmdExe = true
	for _, argv := range [][]string{{"no-such-agent-3440", "x"}, {"sh", "-c", "true"}} {
		if err := assertSafeChildCommand(argv); err != nil {
			t.Errorf("%v: want it left to the launch, got %v", argv, err)
		}
	}
}
