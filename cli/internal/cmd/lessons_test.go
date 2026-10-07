package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLessonsFmt_CheckFailsThenFmtConvergesThenCheckPasses(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"_index.md":       "# Index\n\n| Lesson | Date | Scope |\n|---|---|---|\n",
		"lesson-001-a.md": "# Lesson 001: A\n\n**Date:** 2026-01-01\n\nbody\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	stdout, _, err := execute(t, "lessons", "fmt", "--check", "--dir", dir)
	if err == nil || !strings.Contains(stdout, "not formatted:") {
		t.Fatalf("check on an unformatted tree: err %v\n%s", err, stdout)
	}
	if _, _, err := execute(t, "lessons", "fmt", "--dir", dir); err != nil {
		t.Fatal(err)
	}
	stdout, _, err = execute(t, "lessons", "fmt", "--check", "--dir", dir)
	if err != nil || !strings.Contains(stdout, "[OK]") {
		t.Fatalf("check after fmt: err %v\n%s", err, stdout)
	}
}
