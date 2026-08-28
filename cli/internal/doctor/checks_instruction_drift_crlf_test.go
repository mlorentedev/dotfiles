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

// A source that ends with a blank line and a deployed copy that ends with one
// newline are the same document. Windows' LF writer normalises the tail to a
// single "\n"; the repo source of copilot-instructions.md ends "-->\n\n", and
// the CI gate reported drift on nothing but that (#1308).
func TestCheckInstructionDrift_TrailingNewlinesAreNotDrift(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	for _, tgt := range deployedInstructionTargets {
		body := "# " + tgt.repoRel + "\n\nshared content\n<!-- BEGIN HARNESS GENERATED -->\n<!-- END HARNESS GENERATED -->\n"
		writeFile(t, filepath.Join(repo, tgt.repoRel), body+"\n")
		writeFile(t, filepath.Join(home, tgt.homeRel), body)
	}
	sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, []string{"copilot"}, nil)

	var buf bytes.Buffer
	rep := capture(&buf)
	checkInstructionDrift(sys, rep)

	if rep.Failures() != 0 {
		t.Fatalf("a trailing blank line must not be drift\n%s", buf.String())
	}
}
