package lane

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPromptDir_IsNotNamedLikeTheIdentityRecord(t *testing.T) {
	if PromptDirName == LaneIdentityFile {
		t.Fatalf("the prompt directory and the identity record share the name %q (#3320)", PromptDirName)
	}
	dir, err := PromptDir(t.TempDir())
	if err != nil || filepath.Base(dir) != PromptDirName {
		t.Fatalf("PromptDir = %q, %v", dir, err)
	}
}

func TestPromptDir_MovesTheOldDirectoryOnceAndKeepsItsPrompts(t *testing.T) {
	common := t.TempDir()
	legacy := filepath.Join(common, "dotf-lane")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "7.md"), []byte("prompt"), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		dir, err := PromptDir(common)
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if got, err := os.ReadFile(filepath.Join(dir, "7.md")); err != nil || string(got) != "prompt" {
			t.Fatalf("call %d: the prompt is not reachable under %s: %q, %v", i, dir, got, err)
		}
		if _, err := os.Stat(legacy); !os.IsNotExist(err) {
			t.Fatalf("call %d: the old directory is still there: %v", i, err)
		}
	}
}

func TestPromptDir_LeavesAFileUnderTheOldNameAlone(t *testing.T) {
	common := t.TempDir()
	record := filepath.Join(common, "dotf-lane")
	if err := os.WriteFile(record, []byte("feat-lane-primary\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := PromptDir(common); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(record); err != nil || string(got) != "feat-lane-primary\n" {
		t.Fatalf("the file under the old name was touched: %q, %v", got, err)
	}
}

func TestPromptDir_PrefersTheNewDirectoryWhenBothExist(t *testing.T) {
	common := t.TempDir()
	for _, name := range []string{"dotf-lane", PromptDirName} {
		if err := os.MkdirAll(filepath.Join(common, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	dir, err := PromptDir(common)
	if err != nil || filepath.Base(dir) != PromptDirName {
		t.Fatalf("PromptDir = %q, %v", dir, err)
	}
	if _, err := os.Stat(filepath.Join(common, "dotf-lane")); err != nil {
		t.Fatalf("the old directory was removed although the new one existed: %v", err)
	}
}
