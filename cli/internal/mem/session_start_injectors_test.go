package mem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEncodeProjectPath(t *testing.T) {
	if got := encodeProjectPath("/home/me/Projects/dotfiles"); got != "-home-me-Projects-dotfiles" {
		t.Errorf("got %q", got)
	}
}

func TestClaudeJSONSize(t *testing.T) {
	t.Run("absent file is silent", func(t *testing.T) {
		if got := claudeJSONSize(filepath.Join(t.TempDir(), "nope.json"), 10240); got != "" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("healthy size is silent", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), ".claude.json")
		mustWrite(t, p, strings.Repeat("x", 20000))
		if got := claudeJSONSize(p, 10240); got != "" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("truncated size is flagged byte-exact", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), ".claude.json")
		mustWrite(t, p, strings.Repeat("x", 100))
		want := "\n[claude.json] WARNING: ~/.claude/.claude.json is 100 bytes (threshold 10240). Healthy state is ~75 KB; truncation bug (anthropics/claude-code#59870) reduces it to ~1.5 KB and silently drops subscription state. Recovery: ls -t ~/.claude/backups/.claude.json.backup.* | head -1 && cp <newest-backup> ~/.claude/.claude.json"
		if got := claudeJSONSize(p, 10240); got != want {
			t.Errorf("got %q", got)
		}
	})
}

func TestKnowledgeHealth(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)

	t.Run("absent memory file is silent", func(t *testing.T) {
		if got := knowledgeHealth(filepath.Join(t.TempDir(), "MEMORY.md"), 150, 14, now); got != "" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("never-crystallized marker", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "MEMORY.md")
		mustWrite(t, p, "a\nb\n")
		want := "\nKnowledge crystallization never run — run: ./scripts/knowledge-crystallize.sh"
		if got := knowledgeHealth(p, 150, 14, now); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("over-limit line count is flagged", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "MEMORY.md")
		mustWrite(t, p, strings.Repeat("line\n", 10)+"## Last Crystallized: 2026-06-23\n")
		got := knowledgeHealth(p, 5, 14, now)
		if !strings.HasPrefix(got, "\nMEMORY.md has 11 lines (limit: 5) — run /crystallize to trim") {
			t.Errorf("got %q", got)
		}
	})

	t.Run("stale crystallize date", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "MEMORY.md")
		mustWrite(t, p, "## Last Crystallized: 2026-06-01\n")
		want := "\nCRYSTALLIZE NEEDED (22 days stale)"
		if got := knowledgeHealth(p, 150, 14, now); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("fresh crystallize date is silent", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "MEMORY.md")
		mustWrite(t, p, "## Last Crystallized: 2026-06-23\n")
		if got := knowledgeHealth(p, 150, 14, now); got != "" {
			t.Errorf("got %q", got)
		}
	})
}

func TestMemoryTemperature(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)

	t.Run("absent dir is silent", func(t *testing.T) {
		if got := memoryTemperature(filepath.Join(t.TempDir(), "nope"), 7, 30, 60, now); got != "" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("classifies by age and flags archive-cold", func(t *testing.T) {
		dir := t.TempDir()
		ages := map[string]int{"archive.md": 90, "cold.md": 45, "hot.md": 0, "warm.md": 15}
		for name, days := range ages {
			p := filepath.Join(dir, name)
			mustWrite(t, p, "x")
			mt := now.Add(-time.Duration(days) * 24 * time.Hour)
			if err := os.Chtimes(p, mt, mt); err != nil {
				t.Fatalf("chtimes: %v", err)
			}
		}
		mustWrite(t, filepath.Join(dir, "MEMORY.md"), "x") // must be excluded

		got := memoryTemperature(dir, 7, 30, 60, now)
		for _, want := range []string{
			"\nMemory temperature:",
			"\n  ARCHIVE: archive.md (90d ago)",
			"\n  COLD: cold.md (45d ago)",
			"\n  HOT: hot.md (0d ago)",
			"\n  WARM: warm.md (15d ago)",
			"\nARCHIVE NEEDED: Move memory files >60d old to memory/archive/ and update MEMORY.md index",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %q in %q", want, got)
			}
		}
		// MEMORY.md must not be classified as a temperature entry ("  LABEL: name (Nd ago)").
		// (It still appears in the ARCHIVE NEEDED guidance text, which is fine.)
		if strings.Contains(got, "MEMORY.md (") {
			t.Errorf("MEMORY.md should be excluded from the temperature list: %q", got)
		}
	})
}
