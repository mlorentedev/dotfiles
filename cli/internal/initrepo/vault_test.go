package initrepo

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteVaultEntryWritesFilesAndSymlink(t *testing.T) {
	vault := t.TempDir()
	repoRoot := filepath.Join(t.TempDir(), "myproj")
	claudeDir := t.TempDir()

	res, err := WriteVaultEntry(VaultEntryOptions{
		VaultPath:         vault,
		RepoRoot:          repoRoot,
		Stack:             "go",
		Date:              "2026-06-14",
		ClaudeProjectsDir: claudeDir,
	})
	if err != nil {
		t.Fatalf("WriteVaultEntry: %v", err)
	}
	if res.Action != "written" {
		t.Fatalf("Action = %q, want written (reason: %s)", res.Action, res.Reason)
	}

	entry := filepath.Join(vault, "10_projects", "myproj")
	for _, f := range []string{"00-context.md", "10-roadmap.md", filepath.Join("memory", "MEMORY.md")} {
		if _, err := os.Stat(filepath.Join(entry, f)); err != nil {
			t.Errorf("expected vault file %s: %v", f, err)
		}
	}

	// Decision: no 11-tasks.md in the vault entry (task state lives in the bitácora, ADR-018).
	if _, err := os.Stat(filepath.Join(entry, "11-tasks.md")); err == nil {
		t.Error("vault entry must NOT contain 11-tasks.md (ADR-018)")
	}

	ctx := readFile(t, filepath.Join(entry, "00-context.md"))
	if strings.Contains(ctx, "{{repo}}") || strings.Contains(ctx, "{{stack}}") || strings.Contains(ctx, "{{date}}") {
		t.Errorf("00-context.md still has unsubstituted placeholders:\n%s", ctx)
	}
	if !strings.Contains(ctx, "myproj") || !strings.Contains(ctx, "2026-06-14") {
		t.Errorf("00-context.md missing substituted repo/date:\n%s", ctx)
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

func TestWriteVaultEntrySkipsWithoutVault(t *testing.T) {
	res, err := WriteVaultEntry(VaultEntryOptions{VaultPath: "", RepoRoot: "/x/myproj"})
	if err != nil {
		t.Fatalf("WriteVaultEntry: %v", err)
	}
	if res.Action != "skipped" {
		t.Errorf("Action = %q, want skipped", res.Action)
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
