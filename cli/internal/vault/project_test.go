package vault

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteProjectEntryWritesFilesAndSymlink(t *testing.T) {
	vaultDir := t.TempDir()
	repoRoot := filepath.Join(t.TempDir(), "myproj")
	claudeDir := t.TempDir()

	res, err := WriteProjectEntry(ProjectEntryOptions{
		VaultPath:         vaultDir,
		RepoRoot:          repoRoot,
		Stack:             "go",
		Date:              "2026-06-14",
		ClaudeProjectsDir: claudeDir,
	})
	if err != nil {
		t.Fatalf("WriteProjectEntry: %v", err)
	}
	if res.Action != "written" {
		t.Fatalf("Action = %q, want written (reason: %s)", res.Action, res.Reason)
	}

	entry := filepath.Join(vaultDir, "10_projects", "myproj")
	for _, f := range []string{"00-context.md", "10-roadmap.md", filepath.Join("memory", "MEMORY.md")} {
		if _, err := os.Stat(filepath.Join(entry, f)); err != nil {
			t.Errorf("expected vault file %s: %v", f, err)
		}
	}

	// Decision: no 11-tasks.md in the vault entry (task state lives in the bitácora, ADR-018).
	if _, err := os.Stat(filepath.Join(entry, "11-tasks.md")); err == nil {
		t.Error("vault entry must NOT contain 11-tasks.md (ADR-018)")
	}

	ctx, err := os.ReadFile(filepath.Join(entry, "00-context.md"))
	if err != nil {
		t.Fatalf("read 00-context.md: %v", err)
	}
	ctxStr := string(ctx)
	if strings.Contains(ctxStr, "{{repo}}") || strings.Contains(ctxStr, "{{stack}}") || strings.Contains(ctxStr, "{{date}}") {
		t.Errorf("00-context.md still has unsubstituted placeholders:\n%s", ctxStr)
	}
	if !strings.Contains(ctxStr, "myproj") || !strings.Contains(ctxStr, "2026-06-14") {
		t.Errorf("00-context.md missing substituted repo/date:\n%s", ctxStr)
	}

	if runtime.GOOS != "windows" {
		encoded := strings.ReplaceAll(repoRoot, string(filepath.Separator), "-")
		link := filepath.Join(claudeDir, encoded, "memory")
		target, err := os.Readlink(link)
		if err != nil {
			t.Errorf("expected a memory symlink at %s: %v", link, err)
		} else if target != filepath.Join(entry, "memory") {
			t.Errorf("symlink target = %q, want %q", target, filepath.Join(entry, "memory"))
		}
	}
}

func TestWriteProjectEntrySkipsWithoutVault(t *testing.T) {
	res, err := WriteProjectEntry(ProjectEntryOptions{VaultPath: "", RepoRoot: "/x/myproj"})
	if err != nil {
		t.Fatalf("WriteProjectEntry: %v", err)
	}
	if res.Action != "skipped" {
		t.Errorf("Action = %q, want skipped", res.Action)
	}
}

// TestWriteProjectEntrySkipsPresentAndForceRegenerates locks the #395 acceptance:
// a re-run never clobbers an existing entry, and --force regenerates the files.
func TestWriteProjectEntrySkipsPresentAndForceRegenerates(t *testing.T) {
	vaultDir := t.TempDir()
	repoRoot := filepath.Join(t.TempDir(), "myproj")
	entry := filepath.Join(vaultDir, "10_projects", "myproj")
	opts := ProjectEntryOptions{VaultPath: vaultDir, RepoRoot: repoRoot, Stack: "go", Date: "2026-06-14"}

	first, err := WriteProjectEntry(opts)
	if err != nil {
		t.Fatalf("WriteProjectEntry (first): %v", err)
	}
	if len(first.Created) != 3 {
		t.Fatalf("first run Created = %v, want 3 files", first.Created)
	}

	// Hand-edit, then re-run without --force: the entry must be left untouched.
	ctxPath := filepath.Join(entry, "00-context.md")
	if err := os.WriteFile(ctxPath, []byte("hand-edited"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := WriteProjectEntry(opts)
	if err != nil {
		t.Fatalf("WriteProjectEntry (second): %v", err)
	}
	if len(second.Created) != 0 || len(second.Skipped) != 3 {
		t.Errorf("second run Created=%v Skipped=%v, want 0 created / 3 skipped", second.Created, second.Skipped)
	}
	if got, _ := os.ReadFile(ctxPath); string(got) != "hand-edited" {
		t.Errorf("skip-if-present clobbered a hand-edited file: %q", got)
	}

	// Re-run with --force: the entry files are regenerated.
	forced := opts
	forced.Force = true
	third, err := WriteProjectEntry(forced)
	if err != nil {
		t.Fatalf("WriteProjectEntry (force): %v", err)
	}
	if len(third.Created) != 3 {
		t.Errorf("force run Created = %v, want 3 regenerated", third.Created)
	}
	if got, _ := os.ReadFile(ctxPath); string(got) == "hand-edited" {
		t.Error("--force did not regenerate 00-context.md")
	}
}

func TestResolveVaultMissingIsEmpty(t *testing.T) {
	t.Setenv("VAULT_PATH", filepath.Join(t.TempDir(), "definitely-absent"))
	if got := ResolveVault(); got != "" {
		t.Errorf("ResolveVault() = %q, want empty for an absent vault", got)
	}
}

func TestResolveVaultFromEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VAULT_PATH", dir)
	if got := ResolveVault(); got != dir {
		t.Errorf("ResolveVault() = %q, want %q", got, dir)
	}
}
