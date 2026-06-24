package mem

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// These tests mirror tests/session-brief.bats: they pin each sb_* emitter and the
// --format runner to the shell core's byte-exact output, since PR2a's contract is
// byte-equivalence with session-brief.sh, not loose "contains".

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// --- vaultDetect -------------------------------------------------------------

func TestVaultDetect(t *testing.T) {
	tests := []struct {
		name, root, want string
	}{
		{"headline for a vault root", "/some/myvault", "Obsidian vault detected: myvault (/some/myvault)"},
		{"empty without a root", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vaultDetect(tt.root); got != tt.want {
				t.Errorf("vaultDetect(%q) = %q, want %q", tt.root, got, tt.want)
			}
		})
	}
}

// --- specs -------------------------------------------------------------------

func TestSpecs(t *testing.T) {
	t.Run("silent when there is no specs dir", func(t *testing.T) {
		if got := specs(t.TempDir()); got != "" {
			t.Errorf("want empty, got %q", got)
		}
	})

	t.Run("counts active and archived (leading newline)", func(t *testing.T) {
		dir := t.TempDir()
		mustMkdirAll(t, filepath.Join(dir, "specs", "FOO-1"))
		mustMkdirAll(t, filepath.Join(dir, "specs", "BAR-2"))
		mustMkdirAll(t, filepath.Join(dir, "specs", "archive", "OLD-1"))
		want := "\n[specs] 2 active, 1 archived"
		if got := specs(dir); got != want {
			t.Errorf("specs() = %q, want %q", got, want)
		}
	})

	t.Run("flags specs carrying AGENT-DRAFT tags", func(t *testing.T) {
		dir := t.TempDir()
		mustMkdirAll(t, filepath.Join(dir, "specs", "FOO-1"))
		mustWrite(t, filepath.Join(dir, "specs", "FOO-1", "proposal.md"), "[AGENT-DRAFT] todo\n")
		want := "\n[specs] 1 active, 0 archived — 1 with unresolved [AGENT-DRAFT]/[AGENT-SUGGESTION] tags:\n  - FOO-1"
		if got := specs(dir); got != want {
			t.Errorf("specs() = %q, want %q", got, want)
		}
	})
}

// --- vaultBaseline -----------------------------------------------------------

func healthyVault(t *testing.T) string {
	t.Helper()
	v := t.TempDir()
	mustWrite(t, filepath.Join(v, "00_meta", "patterns", "_index.md"), "")
	mustWrite(t, filepath.Join(v, "00_meta", "skills", "README.md"), "")
	mustWrite(t, filepath.Join(v, "README.md"), "")
	mustWrite(t, filepath.Join(v, "00_meta", "skills", "foo", "SKILL.md"), "content\n")
	return v
}

func TestVaultBaseline(t *testing.T) {
	t.Run("silent on a healthy vault", func(t *testing.T) {
		if got := vaultBaseline(healthyVault(t)); got != "" {
			t.Errorf("want empty, got %q", got)
		}
	})

	t.Run("flags a missing critical file", func(t *testing.T) {
		v := t.TempDir()
		mustWrite(t, filepath.Join(v, "00_meta", "patterns", "_index.md"), "")
		got := vaultBaseline(v)
		if !strings.Contains(got, "Vault baseline FAIL") || !strings.Contains(got, "MISSING: README.md") {
			t.Errorf("got %q", got)
		}
	})

	t.Run("flags an empty SKILL.md", func(t *testing.T) {
		v := t.TempDir()
		mustWrite(t, filepath.Join(v, "00_meta", "patterns", "_index.md"), "")
		mustWrite(t, filepath.Join(v, "00_meta", "skills", "README.md"), "")
		mustWrite(t, filepath.Join(v, "README.md"), "")
		mustWrite(t, filepath.Join(v, "00_meta", "skills", "foo", "SKILL.md"), "")
		got := vaultBaseline(v)
		if !strings.Contains(got, "EMPTY: 00_meta/skills/foo/SKILL.md") {
			t.Errorf("got %q", got)
		}
	})

	t.Run("silent when vault root is empty", func(t *testing.T) {
		if got := vaultBaseline(""); got != "" {
			t.Errorf("want empty, got %q", got)
		}
	})
}

// --- vaultHealth -------------------------------------------------------------

func TestVaultHealth(t *testing.T) {
	t.Run("reports not-installed when vault-health.sh is absent", func(t *testing.T) {
		got := vaultHealth("", "", filepath.Join(t.TempDir(), "nodir"))
		if !strings.Contains(got, "vault-health.sh not found") {
			t.Errorf("got %q", got)
		}
	})

	t.Run("reports ALL CHECKS PASSED on a clean stub", func(t *testing.T) {
		if _, err := exec.LookPath("bash"); err != nil {
			t.Skip("bash required to run vault-health.sh stub")
		}
		sd := t.TempDir()
		stub := filepath.Join(sd, "vault-health.sh")
		mustWrite(t, stub, "#!/bin/sh\nexit 0\n")
		if err := os.Chmod(stub, 0o755); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		want := "\nVault health: ALL CHECKS PASSED"
		if got := vaultHealth("/v", "v", sd); got != want {
			t.Errorf("vaultHealth() = %q, want %q", got, want)
		}
	})
}

// --- lessonsStaleness --------------------------------------------------------

func TestLessonsStaleness(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)

	t.Run("absent lessons file is silent", func(t *testing.T) {
		if got := lessonsStaleness(t.TempDir(), 14, now); got != "" {
			t.Errorf("want empty, got %q", got)
		}
	})

	t.Run("fresh lessons file is silent", func(t *testing.T) {
		dir := t.TempDir()
		lessons := filepath.Join(dir, "docs", "lessons.md")
		mustWrite(t, lessons, "fresh\n")
		if err := os.Chtimes(lessons, now, now); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
		if got := lessonsStaleness(dir, 14, now); got != "" {
			t.Errorf("want empty, got %q", got)
		}
	})

	t.Run("stale lessons file is flagged", func(t *testing.T) {
		dir := t.TempDir()
		lessons := filepath.Join(dir, "docs", "lessons.md")
		mustWrite(t, lessons, "stale\n")
		old := now.Add(-100 * 24 * time.Hour)
		if err := os.Chtimes(lessons, old, old); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
		got := lessonsStaleness(dir, 14, now)
		if !strings.HasPrefix(got, "\n[lessons] docs/lessons.md not updated in >14 days") {
			t.Errorf("got %q", got)
		}
	})
}

// --- Brief assembly ----------------------------------------------------------

func TestBriefDropsLeadingBlankWhenNoHeadline(t *testing.T) {
	dir := t.TempDir() // no .obsidian, no specs, no lessons
	b := Brief(BriefOptions{
		Cwd:        dir,
		ScriptsDir: filepath.Join(dir, "noscripts"), // health → "not found" (leading \n)
		StaleDays:  14,
		Now:        time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC),
	})
	if strings.HasPrefix(b, "\n") {
		t.Errorf("leading blank line not dropped: %q", b)
	}
	if !strings.HasPrefix(b, "vault-health.sh not found") {
		t.Errorf("expected brief to start with the health line, got %q", b)
	}
}

// --- RenderBrief -------------------------------------------------------------

func TestRenderBrief(t *testing.T) {
	t.Run("stdout is not fenced", func(t *testing.T) {
		got, err := RenderBrief("hello", "stdout")
		if err != nil || got != "hello\n" {
			t.Errorf("got %q, err %v", got, err)
		}
	})

	t.Run("markdown is fenced", func(t *testing.T) {
		got, err := RenderBrief("hello", "markdown")
		want := "```text\nhello\n```\n"
		if err != nil || got != want {
			t.Errorf("got %q, err %v", got, err)
		}
	})

	t.Run("unknown format errors", func(t *testing.T) {
		if _, err := RenderBrief("hello", "bogus"); err == nil {
			t.Errorf("want error for bogus format")
		}
	})
}
