package mem

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorDrift(t *testing.T) {
	t.Run("no drift lines → empty", func(t *testing.T) {
		if got := doctorDrift("all good\n  [OK] env-contract\n"); got != "" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("warn/fail lines are surfaced double-newline framed", func(t *testing.T) {
		in := "scanning...\n  [WARN] X not in PATH\n  [FAIL] Y missing\ntrailing\n"
		want := "\n\n[doctor] env-contract drift detected (run 'dotf doctor --fix'):\n  [WARN] X not in PATH\n  [FAIL] Y missing"
		if got := doctorDrift(in); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestHiveProject(t *testing.T) {
	t.Run("personal 10_projects match", func(t *testing.T) {
		vault := t.TempDir()
		mustMkdirAll(t, filepath.Join(vault, "10_projects", "myrepo"))
		want := "\n[hive] Project 'myrepo' found in vault. Use hive-vault MCP tools for on-demand context:" +
			"\n  - vault_query(project=\"myrepo\", section=\"context\") — project overview" +
			"\n  - vault_query(project=\"myrepo\", section=\"tasks\") — active backlog" +
			"\n  - vault_search(query=\"...\") — search across vault" +
			"\n  - vault_query(project=\"_meta\", path=\"patterns/...\") — cross-project patterns"
		if got := hiveProject("/x/myrepo", vault); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("work-SDK component match", func(t *testing.T) {
		vault := t.TempDir()
		mustMkdirAll(t, filepath.Join(vault, "50_work", "45-development", "FamilyX", "CompY"))
		got := hiveProject("/dev/FamilyX/CompY-repo", vault)
		if !strings.HasPrefix(got, "\n[hive] Work SDK 'CompY' (family: FamilyX) found in vault. Use Hive MCP:") {
			t.Errorf("got %q", got)
		}
		if !strings.Contains(got, "path=\"50_work/45-development/FamilyX/CompY/memory/MEMORY.md\") — session memory") {
			t.Errorf("missing component memory line: %q", got)
		}
	})

	t.Run("work-SDK family-only match", func(t *testing.T) {
		vault := t.TempDir()
		mustMkdirAll(t, filepath.Join(vault, "50_work", "45-development", "FamilyX", "Unrelated"))
		want := "\n[hive] Work SDK family 'FamilyX' found in vault. Use Hive MCP:" +
			"\n  - vault_query(project=\"_meta\", path=\"50_work/45-development/FamilyX/context.md\") — all repos in family"
		if got := hiveProject("/dev/FamilyX-thing", vault); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("no match → empty", func(t *testing.T) {
		if got := hiveProject("/somewhere/else", t.TempDir()); got != "" {
			t.Errorf("got %q", got)
		}
	})
}

func TestMemorySymlink(t *testing.T) {
	t.Run("creates the link and surfaces the auto-memory line", func(t *testing.T) {
		vault := t.TempDir()
		mustWrite(t, filepath.Join(vault, "10_projects", "proj", "memory", "MEMORY.md"), "x")
		home := t.TempDir()

		got := memorySymlink("/w/proj", vault, home)
		if !strings.HasPrefix(got, "\n[auto-memory] Created ") || !strings.HasSuffix(got, " for proj") {
			t.Errorf("unexpected message: %q", got)
		}
	})

	t.Run("no vault source → empty", func(t *testing.T) {
		if got := memorySymlink("/w/proj", t.TempDir(), t.TempDir()); got != "" {
			t.Errorf("got %q", got)
		}
	})
}
