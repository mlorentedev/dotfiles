package memlink

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	mkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// --- resolveVaultMemory: the three conventions -------------------------------

func TestResolveVaultMemory(t *testing.T) {
	t.Run("convention 1: 10_projects/<project>/memory", func(t *testing.T) {
		vault := t.TempDir()
		want := filepath.Join(vault, "10_projects", "myproj", "memory")
		mkdirAll(t, want)
		if got := resolveVaultMemory("/anywhere/myproj", "myproj", vault); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("convention 2: CWD inside the vault uses CWD/memory", func(t *testing.T) {
		vault := t.TempDir()
		cwd := filepath.Join(vault, "somedir") // under vault, no 10_projects entry
		want := filepath.Join(cwd, "memory")
		mkdirAll(t, want)
		if got := resolveVaultMemory(cwd, "somedir", vault); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("convention 3: work-SDK family/component slug match", func(t *testing.T) {
		vault := t.TempDir()
		want := filepath.Join(vault, "50_work", "45-development", "FamilyX", "CompY", "memory")
		mkdirAll(t, want)
		// CWD slug must contain both 'familyx' and 'compy'.
		if got := resolveVaultMemory("/dev/FamilyX/CompY-repo", "irrelevant", vault); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("convention 1 wins over a present convention-2 dir", func(t *testing.T) {
		vault := t.TempDir()
		conv1 := filepath.Join(vault, "10_projects", "p", "memory")
		mkdirAll(t, conv1)
		cwd := filepath.Join(vault, "p")
		mkdirAll(t, filepath.Join(cwd, "memory"))
		if got := resolveVaultMemory(cwd, "p", vault); got != conv1 {
			t.Errorf("got %q, want conv1 %q", got, conv1)
		}
	})

	t.Run("no source anywhere → empty", func(t *testing.T) {
		if got := resolveVaultMemory("/nowhere/x", "x", t.TempDir()); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

// --- ClaudeProjectKey / ClaudeMemoryTarget: cross-OS encoding ----------------

func TestClaudeProjectKey(t *testing.T) {
	for in, want := range map[string]string{
		"/home/me/Projects/dotfiles":                     "-home-me-Projects-dotfiles",
		`C:\Users\mlorente\Projects\Workspace\dotfiles`:  "C--Users-mlorente-Projects-Workspace-dotfiles",
	} {
		if got := ClaudeProjectKey(in); got != want {
			t.Errorf("ClaudeProjectKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClaudeMemoryTarget(t *testing.T) {
	got := ClaudeMemoryTarget("/h", "/home/me/proj")
	want := filepath.Join("/h", ".claude", "projects", "-home-me-proj", "memory")
	if got != want {
		t.Errorf("ClaudeMemoryTarget = %q, want %q", got, want)
	}
}

// --- Status: the read-only classifier doctor reports on ----------------------

func TestStatus(t *testing.T) {
	t.Run("no source → StateNoSource", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "memory")
		if got := Status("/nowhere/x", target, "x", t.TempDir()); got != StateNoSource {
			t.Errorf("got %v, want StateNoSource", got)
		}
	})

	t.Run("source exists, target missing → StateRepairable", func(t *testing.T) {
		vault := t.TempDir()
		mkdirAll(t, filepath.Join(vault, "10_projects", "p", "memory"))
		target := filepath.Join(t.TempDir(), "memory")
		if got := Status("/x/p", target, "p", vault); got != StateRepairable {
			t.Errorf("got %v, want StateRepairable", got)
		}
	})

	t.Run("real non-empty dir → StateRealDir (Ensure would leave it; doctor must not destroy)", func(t *testing.T) {
		vault := t.TempDir()
		mkdirAll(t, filepath.Join(vault, "10_projects", "p", "memory"))
		target := filepath.Join(t.TempDir(), "memory")
		writeFile(t, filepath.Join(target, "own.md"), "agent data")
		if got := Status("/x/p", target, "p", vault); got != StateRealDir {
			t.Errorf("got %v, want StateRealDir", got)
		}
	})

	t.Run("already linked → StateLinked", func(t *testing.T) {
		vault := t.TempDir()
		writeFile(t, filepath.Join(vault, "10_projects", "p", "memory", "MEMORY.md"), "x")
		target := filepath.Join(t.TempDir(), "agent", "p", "memory")
		if _, err := Ensure("/w/p", target, "p", vault); err != nil {
			t.Fatalf("Ensure setup: %v", err)
		}
		if got := Status("/w/p", target, "p", vault); got != StateLinked {
			t.Errorf("got %v, want StateLinked", got)
		}
	})
}

// --- Ensure: idempotency + happy path ----------------------------------------

func TestEnsure(t *testing.T) {
	t.Run("no source → no-op, empty message", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "agent", "x", "memory")
		msg, err := Ensure("/nowhere/x", target, "x", t.TempDir())
		if err != nil || msg != "" {
			t.Errorf("msg=%q err=%v", msg, err)
		}
		if _, statErr := os.Lstat(target); statErr == nil {
			t.Errorf("target should not have been created")
		}
	})

	t.Run("target is a non-empty dir → no-op", func(t *testing.T) {
		vault := t.TempDir()
		mkdirAll(t, filepath.Join(vault, "10_projects", "p", "memory"))
		target := filepath.Join(t.TempDir(), "memory")
		writeFile(t, filepath.Join(target, "agent-own.md"), "data")
		msg, err := Ensure("/x/p", target, "p", vault)
		if err != nil || msg != "" {
			t.Errorf("expected no-op, got msg=%q err=%v", msg, err)
		}
	})

	t.Run("happy path creates a link reading through to the vault source", func(t *testing.T) {
		vault := t.TempDir()
		src := filepath.Join(vault, "10_projects", "myproj", "memory")
		writeFile(t, filepath.Join(src, "MEMORY.md"), "vault-content")
		target := filepath.Join(t.TempDir(), "agent", "myproj", "memory")

		msg, err := Ensure("/work/myproj", target, "myproj", vault)
		if err != nil {
			t.Fatalf("Ensure: %v", err)
		}
		if !strings.Contains(msg, "Created") || !strings.Contains(msg, "myproj") {
			t.Errorf("unexpected message: %q", msg)
		}
		got, err := os.ReadFile(filepath.Join(target, "MEMORY.md"))
		if err != nil {
			t.Fatalf("link did not read through: %v", err)
		}
		if string(got) != "vault-content" {
			t.Errorf("read %q through link, want vault-content", got)
		}
	})

	t.Run("already linked → no-op on a second call", func(t *testing.T) {
		vault := t.TempDir()
		writeFile(t, filepath.Join(vault, "10_projects", "p", "memory", "MEMORY.md"), "x")
		target := filepath.Join(t.TempDir(), "agent", "p", "memory")

		if _, err := Ensure("/w/p", target, "p", vault); err != nil {
			t.Fatalf("first Ensure: %v", err)
		}
		msg, err := Ensure("/w/p", target, "p", vault)
		if err != nil || msg != "" {
			t.Errorf("second call should be a no-op, got msg=%q err=%v", msg, err)
		}
	})
}
