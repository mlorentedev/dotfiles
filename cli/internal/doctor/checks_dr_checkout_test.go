package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The escrow lives in the checkout — that is where `dotf secrets backup` writes
// it — and the deploy mirror is copy-only. setup-windows.ps1 mirrored sensitive/
// without its dr/ subdirectory, and doctor reported total-loss risk for 28
// secrets over a 140 KB escrow sitting in the checkout (WIN-011/#1292). The
// checkout copy must count, whatever the mirror holds.
func TestDR_EscrowIsReadFromTheCheckoutTheWriterUses(t *testing.T) {
	now := time.Now()
	repo := t.TempDir()
	touchAt(t, filepath.Join(repo, "sensitive", "dr", "bitwarden-export.age"), now.Add(-24*time.Hour))
	sys := drSysExposed(t, now, 28, nil)
	sys.Getenv = func(k string) string {
		if k == "DOTFILES_REPO_DIR" {
			return repo
		}
		return ""
	}
	cfg := &Config{DotfilesDir: t.TempDir()} // the mirror: no dr/ at all

	var buf bytes.Buffer
	rep := capture(&buf)
	checkDisasterRecovery(sys, cfg, rep)

	if line := lineContaining(buf.String(), "DR escrow"); !strings.Contains(line, "[ OK ]") {
		t.Fatalf("the checkout escrow must count, got: %s\n%s", line, buf.String())
	}
	if rep.Failures() != 0 {
		t.Errorf("no failure expected\n%s", buf.String())
	}
}
