package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// A deployed instruction file that differs from its repo source only by line
// endings is not drift (WIN-008/#1289). setup-windows.ps1's catalog injection
// rewrote copilot-instructions.md CRLF, and the check then failed on every run
// with a remedy (`compile-harness.sh --deploy`) that does not exist on Windows.
// The copilot target also carries the injected catalog region, as on a real
// box, so the strip and the normalisation are exercised together.
func TestCheckInstructionDrift_CRLFDeployedCopyIsNotDrift(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	for _, tgt := range deployedInstructionTargets {
		src := "# " + tgt.repoRel + "\n\nshared content\n"
		writeFile(t, filepath.Join(repo, tgt.repoRel), src)
		deployed := strings.ReplaceAll(src, "\n", "\r\n")
		if tgt.requiresCommand == "copilot" {
			deployed += "\r\n<!-- BEGIN HARNESS GENERATED (sha256:abc) -- skill catalog -->\r\n- **x** -- y\r\n<!-- END HARNESS GENERATED -->\r\n"
		}
		writeFile(t, filepath.Join(home, tgt.homeRel), deployed)
	}
	sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, []string{"copilot"}, nil)

	var buf bytes.Buffer
	rep := capture(&buf)
	checkInstructionDrift(sys, rep)

	if rep.Failures() != 0 {
		t.Fatalf("a CRLF-only difference must not be drift\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "(4 checked)") {
		t.Errorf("all four targets should have been compared\n%s", buf.String())
	}
}
