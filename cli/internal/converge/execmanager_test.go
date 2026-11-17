package converge

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestExecManager_StopsAHungWinget: the system-upgrade step runs winget
// unattended; a winget that hangs after printing its error must end as an
// error, not hold the converge for hours (#3513).
func TestExecManager_StopsAHungWinget(t *testing.T) {
	if os.Getenv("DOTF_TEST_HANG") == "1" {
		time.Sleep(time.Minute)
		return
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	// The command must be named "winget" to be bounded: a copy of this binary.
	name := "winget"
	if runtime.GOOS == "windows" {
		name = "winget.exe"
	}
	copyTo := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(copyTo, data, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTF_TEST_HANG", "1")
	start := time.Now()
	_, err = execManager(300*time.Millisecond, copyTo, "-test.run=^TestExecManager_StopsAHungWinget$")
	if err == nil || !strings.Contains(err.Error(), "did not finish") {
		t.Fatalf("want a timeout error, got %v", err)
	}
	if took := time.Since(start); took > 20*time.Second {
		t.Errorf("a hung winget held the caller for %s", took)
	}
}

// TestExecManager_KeepsStdoutAndLastLine: bounding must not change what the
// step parses: stdout alone on success, the manager's last line on failure.
func TestExecManager_KeepsStdoutAndLastLine(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs sh")
	}
	out, err := ExecManager("sh", "-c", "echo out; echo warn >&2")
	if err != nil || string(out) != "out\n" {
		t.Fatalf("got %q, %v", out, err)
	}
	_, err = ExecManager("sh", "-c", "echo boom >&2; exit 3")
	if err == nil || !strings.HasSuffix(err.Error(), ": boom") {
		t.Fatalf("want an error ending in the last line, got %v", err)
	}
}
