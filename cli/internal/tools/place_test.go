package tools

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestHelperBlockUntilStdinCloses is not a test: placeBinary's tests run a copy
// of this test binary as a stand-in for a running dotf, and this is what the
// copy does. It holds its own executable open until stdin closes.
func TestHelperBlockUntilStdinCloses(_ *testing.T) {
	if os.Getenv("DOTF_TOOLS_TEST_BLOCK") != "1" {
		return
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
	os.Exit(0)
}

// runningCopy copies the test binary to dir/<name> and starts it, blocked on
// stdin. The returned func closes stdin and waits for it to exit.
func runningCopy(t *testing.T, dir, name string) (path string, stop func()) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(self) //nolint:gosec // the test binary itself
	if err != nil {
		t.Fatal(err)
	}
	path = filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o755); err != nil { //nolint:gosec // it must run
		t.Fatal(err)
	}
	cmd := exec.Command(path, "-test.run=^TestHelperBlockUntilStdinCloses$") //nolint:gosec // the copy written above
	cmd.Env = append(os.Environ(), "DOTF_TOOLS_TEST_BLOCK=1")
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	stopped := false
	stop = func() {
		if stopped {
			return
		}
		stopped = true
		_ = in.Close()
		_ = cmd.Wait()
	}
	t.Cleanup(stop)
	return path, stop
}

// CLI-133 AC4: `dotf update` replaces the binary it is running from. Windows
// refuses to rename over, or delete, a running executable but lets it be
// renamed aside, so placeBinary does that on every OS.
func TestPlaceBinaryReplacesARunningExecutable(t *testing.T) {
	dir := t.TempDir()
	dst, stop := runningCopy(t, dir, binFilename("dotf", runtime.GOOS))
	src := filepath.Join(t.TempDir(), "new")
	if err := os.WriteFile(src, []byte("new release"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := placeBinary(src, dst); err != nil {
		t.Fatalf("replacing a running executable: %v", err)
	}
	if got, _ := os.ReadFile(dst); !bytes.Equal(got, []byte("new release")) { //nolint:gosec // test path
		t.Fatalf("dst holds %q, want the new release", got)
	}

	// Once the old process is gone, the next placement leaves nothing aside.
	stop()
	if err := placeBinary(src, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dst + ".old"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s.old left behind after a placement with nothing running: %v", dst, err)
	}
}

// When the new file cannot be renamed in, the old one goes back: a failed
// update must never leave the machine without a dotf.
func TestPlaceBinaryRestoresTheOldFileWhenTheFinalRenameFails(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "dotf")
	if err := os.WriteFile(dst, []byte("old release"), 0o600); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(t.TempDir(), "new")
	if err := os.WriteFile(src, []byte("new release"), 0o600); err != nil {
		t.Fatal(err)
	}
	real := renameFile
	t.Cleanup(func() { renameFile = real })
	renameFile = func(from, to string) error {
		if to == dst && filepath.Dir(from) == dir && from != dst+".old" {
			return errors.New("injected: final rename refused")
		}
		return real(from, to)
	}

	if err := placeBinary(src, dst); err == nil {
		t.Fatal("placeBinary succeeded although the final rename failed")
	}
	if got, _ := os.ReadFile(dst); !bytes.Equal(got, []byte("old release")) { //nolint:gosec // test path
		t.Fatalf("dst holds %q, want the old release restored", got)
	}
	if _, err := os.Stat(dst + ".old"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s.old left behind after the restore: %v", dst, err)
	}
}
