package cmd

import (
	"archive/zip"
	"debug/pe"
	"encoding/binary"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The guest gets committed content only: an edit in the working tree, staged
// or not, must not reach it.
func TestArchiveRef_TakesTheCommittedTreeOnly(t *testing.T) {
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		c.Dir = repo
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q")
	write("install.ps1", "committed")
	git("add", "install.ps1")
	git("commit", "-q", "-m", "c")
	write("install.ps1", "edited")
	write("secret.txt", "untracked")
	// dotf runs from cli/, a subdirectory, where a bare git archive takes
	// only that subdirectory.
	if err := os.Mkdir(filepath.Join(repo, "cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Join(repo, "cli"))

	zp, err := archiveRef("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(zp) })
	z, err := zip.OpenReader(zp)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = z.Close() }()
	var names []string
	for _, f := range z.File {
		names = append(names, f.Name)
		if f.Name == "dotfiles/install.ps1" {
			r, _ := f.Open()
			b, _ := io.ReadAll(r)
			_ = r.Close()
			if string(b) != "committed" {
				t.Errorf("the archive holds the working tree's install.ps1: %q", b)
			}
		}
	}
	if got := strings.Join(names, " "); got != "dotfiles/ dotfiles/install.ps1" {
		t.Errorf("archive entries: %s", got)
	}

	for _, bad := range []string{"--output=/tmp/x", "no-such-ref"} {
		if _, err := archiveRef(bad); err == nil {
			t.Errorf("archiveRef(%q) succeeded", bad)
		}
	}
}

// A --dotf that names no file is a usage error, refused before any guest
// boots or any host check runs.
func TestVMWindows_RefusesADotfThatIsNotAFile(t *testing.T) {
	for _, path := range []string{filepath.Join(t.TempDir(), "absent.exe"), t.TempDir()} {
		c := newVMWindowsCmd()
		c.SetArgs([]string{"--dotf", path})
		c.SetOut(io.Discard)
		c.SetErr(io.Discard)
		err := c.Execute()
		if err == nil || ExitCode(err) != 2 || !strings.Contains(err.Error(), "not a file") {
			t.Errorf("--dotf %s: got %v (exit %d), want a usage error", path, err, ExitCode(err))
		}
	}
}

// writePE writes the smallest file debug/pe accepts: a DOS stub pointing at a
// PE signature and a file header for machine, with no sections. debug/pe
// reads a 96-byte DOS header, so the signature sits past it.
func writePE(t *testing.T, machine uint16) string {
	t.Helper()
	b := make([]byte, 0x80+4+20)
	copy(b, "MZ")
	binary.LittleEndian.PutUint32(b[0x3c:], 0x80)
	copy(b[0x80:], "PE\x00\x00")
	binary.LittleEndian.PutUint16(b[0x84:], machine)
	path := filepath.Join(t.TempDir(), "dotf.exe")
	if err := os.WriteFile(path, b, 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// The guest is Windows on Arm64: a --dotf built for anything else, or no
// Windows executable at all, is refused before a guest boots, not found out
// after the install has run.
func TestVMWindows_RefusesADotfTheGuestCannotRun(t *testing.T) {
	text := filepath.Join(t.TempDir(), "dotf.exe")
	if err := os.WriteFile(text, []byte("MZ dev build"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{text, writePE(t, pe.IMAGE_FILE_MACHINE_AMD64)} {
		c := newVMWindowsCmd()
		c.SetArgs([]string{"--dotf", path})
		c.SetOut(io.Discard)
		c.SetErr(io.Discard)
		err := c.Execute()
		if err == nil || ExitCode(err) != 2 || !strings.Contains(err.Error(), "Windows Arm64") {
			t.Errorf("--dotf %s: got %v (exit %d), want a usage error naming Windows Arm64", path, err, ExitCode(err))
		}
	}
	if err := windowsArm64(writePE(t, pe.IMAGE_FILE_MACHINE_ARM64)); err != nil {
		t.Errorf("an Arm64 PE was refused: %v", err)
	}
}
