package tools

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
	// The old process still runs from <dst>.old, which Windows will not
	// delete: a long-lived `dotf secrets run` wrapper holds it for days. The
	// next update must still succeed.
	if err := placeBinary(src, dst); err != nil {
		t.Fatalf("replacing again while the set-aside binary still runs: %v", err)
	}

	// Once the old process is gone, the next placement sweeps every copy aside.
	stop()
	if err := placeBinary(src, dst); err != nil {
		t.Fatal(err)
	}
	if left := asideCopies(t, dst); len(left) != 0 {
		t.Errorf("copies set aside survived a placement with nothing running: %v", left)
	}
}

// asideCopies lists the files placeBinary set aside for dst.
func asideCopies(t *testing.T, dst string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(dst))
	if err != nil {
		t.Fatal(err)
	}
	var left []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), filepath.Base(dst)+".old") {
			left = append(left, e.Name())
		}
	}
	return left
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
		if to == dst && !strings.HasPrefix(filepath.Base(from), "dotf.old") {
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
	if left := asideCopies(t, dst); len(left) != 0 {
		t.Errorf("copies set aside survived the restore: %v", left)
	}
}

// The Windows case on every OS: <dst>.old cannot be removed because a process
// still runs from it. The update sets the current binary aside under another
// name instead of failing.
func TestPlaceBinarySetsAsideAgainWhileAnOldCopyIsHeld(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "dotf")
	for name, body := range map[string]string{"dotf": "current", "dotf.old": "still running"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	src := filepath.Join(t.TempDir(), "new")
	if err := os.WriteFile(src, []byte("new release"), 0o600); err != nil {
		t.Fatal(err)
	}
	real := removeFile
	t.Cleanup(func() { removeFile = real })
	removeFile = func(name string) error {
		if name == dst+".old" {
			return errors.New("injected: Access is denied")
		}
		return real(name)
	}

	if err := placeBinary(src, dst); err != nil {
		t.Fatalf("a held %s.old failed the update: %v", dst, err)
	}
	if got, _ := os.ReadFile(dst); !bytes.Equal(got, []byte("new release")) { //nolint:gosec // test path
		t.Errorf("dst holds %q, want the new release", got)
	}
}

// The sweep removes only what replaceFile set aside: a sibling whose name merely
// starts with <base>.old belongs to someone else. No fixture ends in a dot:
// Windows strips a trailing dot, so "foo.old." would be "foo.old" there.
func TestRemoveAsideLeavesUnrelatedSiblings(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "foo")
	for _, name := range []string{"foo.old", "foo.old.1731400000123456789", "foo.older", "foo.old-backup", "foo.old.x1"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	removeAside(dst)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var left []string
	for _, e := range entries {
		left = append(left, e.Name())
	}
	want := []string{"foo.old-backup", "foo.old.x1", "foo.older"}
	if strings.Join(left, ",") != strings.Join(want, ",") {
		t.Errorf("after the sweep: %v, want %v", left, want)
	}
}
