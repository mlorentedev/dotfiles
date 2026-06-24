package mem

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClaudeEnvelope(t *testing.T) {
	t.Run("matches jq shape: 2-space indent + trailing newline", func(t *testing.T) {
		got, err := ClaudeEnvelope("hello")
		if err != nil {
			t.Fatal(err)
		}
		want := "{\n  \"hookSpecificOutput\": {\n    \"hookEventName\": \"SessionStart\",\n    \"additionalContext\": \"hello\"\n  }\n}\n"
		if got != want {
			t.Errorf("got %q\nwant %q", got, want)
		}
	})

	t.Run("does NOT HTML-escape < > & (jq parity)", func(t *testing.T) {
		got, err := ClaudeEnvelope("recovery: cp <newest-backup> & retry > log")
		if err != nil {
			t.Fatal(err)
		}
		// Go's json default escapes <>& to < etc.; jq does not. With
		// SetEscapeHTML(false) the literal phrase survives intact — if any char were
		// escaped, this exact substring would not appear.
		if !strings.Contains(got, "cp <newest-backup> & retry > log") {
			t.Errorf("literal <>& not preserved (HTML-escaped, diverges from jq): %q", got)
		}
	})
}

func TestClaudeContextAssembly(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	scriptsNoHealth := filepath.Join(t.TempDir(), "noscripts") // vault-health.sh absent

	t.Run("sdd reminder is always first; bare CWD appends only health", func(t *testing.T) {
		ctx := ClaudeContext(ClaudeContextInput{
			Cwd: t.TempDir(), Vault: t.TempDir(), ScriptsDir: scriptsNoHealth,
			Home: t.TempDir(), Now: now,
		})
		if !strings.HasPrefix(ctx, sddReminder) {
			t.Errorf("ctx must start with the [sdd] reminder, got: %q", ctx[:min(80, len(ctx))])
		}
		if !strings.Contains(ctx, "vault-health.sh not found") {
			t.Errorf("expected the health 'not found' line, got: %q", ctx)
		}
	})

	t.Run("vault CWD prepends the headline before the reminder", func(t *testing.T) {
		vaultCwd := t.TempDir()
		mustMkdirAll(t, filepath.Join(vaultCwd, ".obsidian"))
		ctx := ClaudeContext(ClaudeContextInput{
			Cwd: vaultCwd, Vault: t.TempDir(), ScriptsDir: scriptsNoHealth,
			Home: t.TempDir(), Now: now,
		})
		if !strings.HasPrefix(ctx, "Obsidian vault detected:") {
			t.Errorf("headline should be prepended, got: %q", ctx[:min(80, len(ctx))])
		}
		if !strings.Contains(ctx, "\n\n"+sddReminder) {
			t.Errorf("headline should be followed by a blank line then the reminder")
		}
	})

	t.Run("doctor-drift is gated off without a contract", func(t *testing.T) {
		ctx := ClaudeContext(ClaudeContextInput{
			Cwd: t.TempDir(), Vault: t.TempDir(), ScriptsDir: scriptsNoHealth, Home: t.TempDir(),
			ContractPath: filepath.Join(t.TempDir(), "absent.json"),
			DoctorQuick:  func() string { return "  [WARN] should not appear" },
			Now:          now,
		})
		if strings.Contains(ctx, "[doctor]") {
			t.Errorf("doctor-drift must be skipped when the contract is absent: %q", ctx)
		}
	})
}
