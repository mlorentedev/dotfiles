package mem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
			// archive.md carries an archivable type so this case still exercises the
			// age buckets; the type rule that gates ARCHIVE is covered separately below.
			mustWrite(t, p, "---\nmetadata:\n  type: project\n---\n\nx\n")
			mt := now.Add(-time.Duration(days) * 24 * time.Hour)
			if err := os.Chtimes(p, mt, mt); err != nil {
				t.Fatalf("chtimes: %v", err)
			}
		}
		mustWrite(t, filepath.Join(dir, "MEMORY.md"), "x") // must be excluded

		got := memoryTemperature(dir, 7, 30, 60, now)
		for _, want := range []string{
			"\nMemory temperature:",
			"\n  ARCHIVE: archive.md (90d ago, type=project)",
			"\n  COLD: cold.md (45d ago)",
			"\n  HOT: hot.md (0d ago)",
			"\n  WARM: warm.md (15d ago)",
			"\nARCHIVE NEEDED (1): move to memory/archive/ and drop each file's MEMORY.md pointer in the same edit",
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

	// HARNESS-073 (#967): mtime measures when a rule was last EDITED, never when it
	// was last relied on, so a standing guardrail untouched for 90 days is the most
	// settled entry in the directory rather than the most disposable one. Following
	// the age sweep on 2026-08-14 archived four active guardrails at once.
	t.Run("only project and reference archive on age", func(t *testing.T) {
		dir := t.TempDir()
		files := map[string]string{
			"guardrail.md": "feedback",
			"finished.md":  "project",
			"links.md":     "reference",
			"untyped.md":   "", // no frontmatter at all
		}
		for name, typ := range files {
			body := "no frontmatter here\n"
			if typ != "" {
				body = "---\nname: x\nmetadata:\n  node_type: memory\n  type: " + typ + "\n---\n\nbody\n"
			}
			p := filepath.Join(dir, name)
			mustWrite(t, p, body)
			mt := now.Add(-90 * 24 * time.Hour)
			if err := os.Chtimes(p, mt, mt); err != nil {
				t.Fatalf("chtimes: %v", err)
			}
		}

		got := memoryTemperature(dir, 7, 30, 60, now)
		for _, want := range []string{
			"\n  ARCHIVE: finished.md (90d ago, type=project)",
			"\n  ARCHIVE: links.md (90d ago, type=reference)",
			// A standing instruction never ages out, and an entry whose kind cannot be
			// established is exempt too: archiving drops the MEMORY.md pointer, which is
			// the only surface loaded at session start, so the unknown case must degrade
			// to "kept" rather than to "moved".
			"\n  STANDING: guardrail.md (90d ago, type=feedback — not archivable on age)",
			"\n  STANDING: untyped.md (90d ago, type=unknown — not archivable on age)",
			// The banner names WHICH files qualify and why, so the call is reviewable
			// rather than a bare count.
			"\nARCHIVE NEEDED (2):",
			"\n  - finished.md (90d, type=project)",
			"\n  - links.md (90d, type=reference)",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %q in %q", want, got)
			}
		}
		if strings.Contains(got, "ARCHIVE: guardrail.md") || strings.Contains(got, "- guardrail.md") {
			t.Errorf("a feedback memory must never be proposed for archive: %q", got)
		}
	})

	t.Run("no archivable file means no banner", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "guardrail.md")
		mustWrite(t, p, "---\nmetadata:\n  type: feedback\n---\n\nbody\n")
		mt := now.Add(-400 * 24 * time.Hour)
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
		got := memoryTemperature(dir, 7, 30, 60, now)
		if strings.Contains(got, "ARCHIVE NEEDED") {
			t.Errorf("a directory of standing rules must not raise the banner: %q", got)
		}
	})
}
