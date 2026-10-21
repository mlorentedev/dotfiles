package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"debug/pe"
	"encoding/binary"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// archiveRepo makes a repository whose committed install.ps1 and install.sh
// differ from the working tree's, with an untracked file beside them, and
// runs the test from its cli/ subdirectory, where dotf runs and where a bare
// git archive would take only that subdirectory.
func archiveRepo(t *testing.T) {
	t.Helper()
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
	write("install.sh", "committed")
	git("add", "install.ps1", "install.sh")
	git("commit", "-q", "-m", "c")
	write("install.ps1", "edited")
	write("install.sh", "edited")
	write("secret.txt", "untracked")
	if err := os.Mkdir(filepath.Join(repo, "cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Join(repo, "cli"))
}

// The guest gets committed content only: an edit in the working tree, staged
// or not, must not reach it.
func TestArchiveRef_TakesTheCommittedTreeOnly(t *testing.T) {
	archiveRepo(t)
	zp, err := archiveRef("HEAD", "zip", "install.ps1")
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
	if got := strings.Join(names, " "); got != "dotfiles/ dotfiles/install.ps1 dotfiles/install.sh" {
		t.Errorf("archive entries: %s", got)
	}

	for _, bad := range []string{"--output=/tmp/x", "no-such-ref"} {
		if _, err := archiveRef(bad, "zip", "install.ps1"); err == nil {
			t.Errorf("archiveRef(%q) succeeded", bad)
		}
	}
}

// The Linux guest gets the same committed tree as a tar.gz, which its tar
// extracts with no extra package, and any ref the host resolves works: an
// abbreviated or unpushed commit included (#2788).
func TestArchiveRef_TarGzForTheLinuxGuest(t *testing.T) {
	archiveRepo(t)
	short, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	tp, err := archiveRef(strings.TrimSpace(string(short)), "tar.gz", "install.sh")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(tp) })
	f, err := os.Open(tp)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("not a gzip stream: %v", err)
	}
	tr := tar.NewReader(gz)
	var names []string
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeXGlobalHeader {
			continue // git archive's commit id
		}
		names = append(names, h.Name)
		if h.Name == "dotfiles/install.sh" {
			if b, _ := io.ReadAll(tr); string(b) != "committed" {
				t.Errorf("the archive holds the working tree's install.sh: %q", b)
			}
		}
	}
	if got := strings.Join(names, " "); got != "dotfiles/ dotfiles/install.ps1 dotfiles/install.sh" {
		t.Errorf("archive entries: %s", got)
	}
	if _, err := archiveRef("HEAD", "tar.gz", "no-such-entry.sh"); err == nil {
		t.Error("a ref without the entry script was archived")
	}
}

// noGuest points dotf's state at an empty directory, so a refusal that
// regresses fails on the missing clean-base at once instead of booting the
// real guest a Mac that has run `dotf vm windows build` holds.
func noGuest(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
}

// A --dotf that names no file is a usage error, refused before any guest
// boots or any host check runs.
func TestVMWindows_RefusesADotfThatIsNotAFile(t *testing.T) {
	noGuest(t)
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
// PE signature, a file header for machine and, when exe, the PE32+ optional
// header and executable-image flag an .exe carries; no sections. debug/pe
// reads a 96-byte DOS header, so the signature sits past it.
func writePE(t *testing.T, machine uint16, exe bool) string {
	t.Helper()
	const sig, opt = 0x80, 112 + 16*8
	b := make([]byte, sig+4+20+opt)
	copy(b, "MZ")
	binary.LittleEndian.PutUint32(b[0x3c:], sig)
	copy(b[sig:], "PE\x00\x00")
	binary.LittleEndian.PutUint16(b[sig+4:], machine)
	if exe {
		binary.LittleEndian.PutUint16(b[sig+4+16:], opt)
		binary.LittleEndian.PutUint16(b[sig+4+18:], pe.IMAGE_FILE_EXECUTABLE_IMAGE|pe.IMAGE_FILE_LARGE_ADDRESS_AWARE)
		binary.LittleEndian.PutUint16(b[sig+24:], 0x20b)
		binary.LittleEndian.PutUint32(b[sig+24+108:], 16)
	} else {
		b = b[:sig+4+20]
	}
	path := filepath.Join(t.TempDir(), "dotf.exe")
	if err := os.WriteFile(path, b, 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// The guest is Windows on Arm64: a --dotf built for anything else, an Arm64
// object that is no executable image, or no PE at all, is refused before a guest boots, not found out
// after the install has run.
func TestVMWindows_RefusesADotfTheGuestCannotRun(t *testing.T) {
	noGuest(t)
	text := filepath.Join(t.TempDir(), "dotf.exe")
	if err := os.WriteFile(text, []byte("MZ dev build"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{text, writePE(t, pe.IMAGE_FILE_MACHINE_AMD64, true), writePE(t, pe.IMAGE_FILE_MACHINE_ARM64, false)} {
		c := newVMWindowsCmd()
		c.SetArgs([]string{"--dotf", path})
		c.SetOut(io.Discard)
		c.SetErr(io.Discard)
		err := c.Execute()
		if err == nil || ExitCode(err) != 2 || !strings.Contains(err.Error(), "Windows Arm64") {
			t.Errorf("--dotf %s: got %v (exit %d), want a usage error naming Windows Arm64", path, err, ExitCode(err))
		}
	}
	if err := windowsArm64(writePE(t, pe.IMAGE_FILE_MACHINE_ARM64, true)); err != nil {
		t.Errorf("an Arm64 PE was refused: %v", err)
	}
}
