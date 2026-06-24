package mem

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestSessionStartByteEquivalence is PR2a's regression gate: the Go core
// (Brief + RenderBrief) must be byte-for-byte identical to the shell core
// scripts/session-brief.sh across representative CWDs and both formats. Passing this
// is what lets HARNESS-026's session-brief.sh be deleted at PR3.
//
// POSIX-only by design: session-brief.sh is a POSIX shell script, and its
// "vault-health.sh not found at <path>" line renders the path in bash's form
// (/c/Users/…) vs Go's native form (C:\Users\…) on Windows, so the two cannot be
// byte-equal there. The Windows session-start equivalence target is
// claude-session-start.ps1, covered in PR2b. The harness copies the core into a
// scriptsDir WITHOUT a sibling vault-health.sh, pinning the health emitter to its
// deterministic "not found" branch so the whole comparison stays hermetic (no
// Obsidian GUI, no git state, no flakiness).
func TestSessionStartByteEquivalence(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("session-brief.sh is POSIX; the Windows equivalence target is the .ps1 (PR2b)")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash required to run session-brief.sh")
	}

	realCore := filepath.Join(repoRoot(t), "scripts", "session-brief.sh")
	if _, err := os.Stat(realCore); err != nil {
		t.Skipf("session-brief.sh not found: %v", err)
	}

	// Copy the core into a scriptsDir with no vault-health.sh sibling: both runners
	// then emit the same "not found at <scriptsDir>/vault-health.sh" health line.
	scriptsDir := t.TempDir()
	core := filepath.Join(scriptsDir, "session-brief.sh")
	copyFile(t, realCore, core)
	if err := os.Chmod(core, 0o755); err != nil {
		t.Fatalf("chmod core: %v", err)
	}

	now := time.Now()
	cwds := map[string]string{
		"empty-outside-vault": newEmptyCWD(t),
		"repo-like-no-vault":  newRepoLikeCWD(t, now),                     // fresh lessons
		"inside-vault":        newVaultCWD(t, now.Add(-100*24*time.Hour)), // stale lessons
	}

	for name, cwd := range cwds {
		for _, format := range []string{"stdout", "markdown"} {
			t.Run(name+"/"+format, func(t *testing.T) {
				cmd := exec.Command("bash", core, "--format="+format)
				cmd.Env = append(os.Environ(), "SESSION_BRIEF_CWD="+cwd)
				shOut, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("shell core failed: %v\n%s", err, shOut)
				}

				brief := Brief(BriefOptions{Cwd: cwd, ScriptsDir: scriptsDir, StaleDays: 14, Now: now})
				goOut, err := RenderBrief(brief, format)
				if err != nil {
					t.Fatalf("RenderBrief: %v", err)
				}

				if string(shOut) != goOut {
					t.Errorf("byte mismatch:\n--- shell ---\n%q\n--- go ---\n%q", shOut, goOut)
				}
			})
		}
	}
}

// repoRoot walks up from the test's working dir (the package dir) to the dotfiles
// root that holds scripts/session-brief.sh.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "scripts", "session-brief.sh")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("repo root with scripts/session-brief.sh not found")
		}
		dir = parent
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

func newEmptyCWD(t *testing.T) string {
	return t.TempDir()
}

// newRepoLikeCWD builds a non-vault repo: 2 active specs (one carrying an
// AGENT-DRAFT tag), 1 archived, and a docs/lessons.md stamped at lessonsTime.
func newRepoLikeCWD(t *testing.T, lessonsTime time.Time) string {
	dir := t.TempDir()
	mustMkdirAll(t, filepath.Join(dir, "specs", "AAA-1"))
	mustMkdirAll(t, filepath.Join(dir, "specs", "archive", "OLD-1"))
	mustWrite(t, filepath.Join(dir, "specs", "BBB-2", "proposal.md"), "[AGENT-DRAFT] todo\n")
	lessons := filepath.Join(dir, "docs", "lessons.md")
	mustWrite(t, lessons, "lessons\n")
	if err := os.Chtimes(lessons, lessonsTime, lessonsTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	return dir
}

// newVaultCWD builds a healthy vault root (.obsidian + canonical 00_meta artifacts)
// with a docs/lessons.md stamped at lessonsTime — exercises headline + lessons + a
// silent (healthy) baseline.
func newVaultCWD(t *testing.T, lessonsTime time.Time) string {
	dir := t.TempDir()
	mustMkdirAll(t, filepath.Join(dir, ".obsidian"))
	mustWrite(t, filepath.Join(dir, "00_meta", "patterns", "_index.md"), "")
	mustWrite(t, filepath.Join(dir, "00_meta", "skills", "README.md"), "")
	mustWrite(t, filepath.Join(dir, "00_meta", "skills", "foo", "SKILL.md"), "content\n")
	mustWrite(t, filepath.Join(dir, "README.md"), "")
	lessons := filepath.Join(dir, "docs", "lessons.md")
	mustWrite(t, lessons, "lessons\n")
	if err := os.Chtimes(lessons, lessonsTime, lessonsTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	return dir
}
