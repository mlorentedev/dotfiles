package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/pi"
)

func TestPiPackagesApplyFindsWindowsCmdShimWithoutFlag(t *testing.T) {
	agentDir, _ := piRepo(t, `{"version":1,"packages":[{"source":"npm:test@1","why":"test"}]}`, `{"packages":[]}`)
	t.Setenv("USERPROFILE", t.TempDir())
	binDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "called")
	writeFileAt(t, filepath.Join(binDir, "pi.cmd"), "@echo off\r\necho %* > \""+marker+"\"\r\n")
	writeFileAt(t, filepath.Join(binDir, "npm.cmd"), "@echo off\r\nexit /b 0\r\n")
	t.Setenv("PATH", binDir+";"+os.Getenv("PATH"))
	piRun, piLookPath = pi.ExecRunner, exec.LookPath

	out, _, err := execute(t, "pi", "packages", "apply", "--agent-dir", agentDir)
	if err != nil || !strings.Contains(out, "changed=1") {
		t.Fatalf("Windows pi shim must reconcile without --pi: err=%v\n%s", err, out)
	}
	called, err := os.ReadFile(marker)
	if err != nil || !strings.Contains(string(called), "install npm:test@1") {
		t.Fatalf("pi.cmd must receive the install: err=%v, called=%q", err, called)
	}
}
