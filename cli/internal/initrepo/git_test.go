package initrepo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitInitIsIdempotent(t *testing.T) {
	root := t.TempDir()

	action, err := GitInit(root)
	if err != nil {
		t.Fatalf("GitInit: %v", err)
	}
	if action != "initialized" {
		t.Errorf("first GitInit action = %q, want initialized", action)
	}
	if info, err := os.Stat(filepath.Join(root, ".git")); err != nil || !info.IsDir() {
		t.Errorf("expected a .git directory: %v", err)
	}

	action, err = GitInit(root)
	if err != nil {
		t.Fatalf("second GitInit: %v", err)
	}
	if action != "exists" {
		t.Errorf("re-run GitInit action = %q, want exists", action)
	}
}

func TestPreCommitInstallSkipsWithoutTool(t *testing.T) {
	root := t.TempDir()
	// Empty PATH guarantees pre-commit is not found -> graceful skip.
	t.Setenv("PATH", t.TempDir())
	action, err := PreCommitInstall(root)
	if err != nil {
		t.Fatalf("PreCommitInstall should not error when the tool is absent: %v", err)
	}
	if action != "skipped" {
		t.Errorf("action = %q, want skipped", action)
	}
}
