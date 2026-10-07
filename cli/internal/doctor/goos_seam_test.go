package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These checks decide by the target OS, so they read the System.GOOS seam and
// every branch runs on any host (#2061). They used to read runtime.GOOS, which
// tied the answer to the machine running the test.

func TestCheckShadowedCatalogTools_FindsWindowsShapesThroughTheSeam(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "packages.json"), catalogWithOpencode)
	a, b := filepath.Join(t.TempDir(), "npm"), filepath.Join(t.TempDir(), "winget")
	writeExec(t, filepath.Join(a, "opencode.cmd"))
	writeExec(t, filepath.Join(b, "opencode.exe"))
	sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo, "PATH": a + string(os.PathListSeparator) + b}, nil, nil)
	sys.GOOS = "windows"
	var buf bytes.Buffer
	checkShadowedCatalogTools(sys, &Config{DotfilesDir: t.TempDir()}, capture(&buf))
	if !strings.Contains(buf.String(), "opencode resolves from 2 PATH directories") {
		t.Errorf("GOOS=windows must look for the .cmd and .exe shapes\n%s", buf.String())
	}
}

// The other direction: a non-Windows target looks for the bare name only, even
// on a Windows host. This is the half that runtime.GOOS used to override, so it
// fails on the windows-latest leg if the host check returns.
func TestCheckShadowedCatalogTools_IgnoresWindowsShapesForAPosixTarget(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "packages.json"), catalogWithOpencode)
	a, b := filepath.Join(t.TempDir(), "npm"), filepath.Join(t.TempDir(), "winget")
	writeExec(t, filepath.Join(a, "opencode.cmd"))
	writeExec(t, filepath.Join(b, "opencode.exe"))
	sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo, "PATH": a + string(os.PathListSeparator) + b}, nil, nil)
	sys.GOOS = "linux"
	var buf bytes.Buffer
	checkShadowedCatalogTools(sys, &Config{DotfilesDir: t.TempDir()}, capture(&buf))
	if strings.Contains(buf.String(), "PATH directories") {
		t.Errorf("GOOS=linux must not count .cmd/.exe files as the tool\n%s", buf.String())
	}
}

func TestCheckPathFiles_ChecksTheTargetOSsPathFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "paths.sh"), "# posix\n")
	sys := newSys(map[string]string{"DOTFILES_DIR": dir, "HOME": t.TempDir()}, nil, nil)
	sys.GOOS = "windows"
	var buf bytes.Buffer
	checkPathFiles(sys, &Config{ContractPath: filepath.Join(dir, "env-contract.json")}, capture(&buf))
	if !strings.Contains(buf.String(), "paths.ps1 not generated") {
		t.Errorf("GOOS=windows must check paths.ps1, not this host's paths.sh\n%s", buf.String())
	}
}

// And the POSIX target checks paths.sh, whichever host runs it.
func TestCheckPathFiles_ChecksPathsShForAPosixTarget(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "paths.ps1"), "# windows\n")
	sys := newSys(map[string]string{"DOTFILES_DIR": dir, "HOME": t.TempDir()}, nil, nil)
	sys.GOOS = "linux"
	var buf bytes.Buffer
	checkPathFiles(sys, &Config{ContractPath: filepath.Join(dir, "env-contract.json")}, capture(&buf))
	if !strings.Contains(buf.String(), "paths.sh not generated") {
		t.Errorf("GOOS=linux must check paths.sh, not paths.ps1\n%s", buf.String())
	}
}
