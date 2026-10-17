package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLessonsFmt_CheckFailsThenFmtConvergesThenCheckPasses(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
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

func TestLessonsFmt_CheckNamesACommittedIndexAndFmtLeavesItAlone(t *testing.T) {
	dir := t.TempDir()
	lesson := filepath.Join(dir, "lesson-001-a.md")
	index := filepath.Join(dir, "_index.md")
	if err := os.WriteFile(lesson, []byte("---\nid: \"lesson-001-a\"\ntype: lesson\nstatus: active\ntitle: \"A\"\ncreated: \"2026-01-01\"\n---\n\n# A\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(index, []byte("# Index\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, err := execute(t, "lessons", "fmt", "--check", "--dir", dir)
	if err == nil || !strings.Contains(stdout, "committed index: "+index) || !strings.Contains(stdout, "dotf lessons index") {
		t.Fatalf("check must name the committed index and the fix: err %v\n%s", err, stdout)
	}
	stdout, _, err = execute(t, "lessons", "fmt", "--dir", dir)
	if err != nil || !strings.Contains(stdout, "left alone: "+index) {
		t.Fatalf("fmt must say it left the index alone: err %v\n%s", err, stdout)
	}
	if got, rerr := os.ReadFile(index); rerr != nil || string(got) != "# Index\n" {
		t.Errorf("fmt touched the committed index: %q (%v)", got, rerr)
	}
}

func TestLessonsIndex_PrintsTheTableAndWritesNothing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "lesson-007-a.md"), []byte("# Lesson 007: Seven\n\n**Date:** 2026-01-07\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := execute(t, "lessons", "index", "--dir", dir)
	if err != nil || !strings.Contains(stdout, "| [007 - Seven](lesson-007-a.md) | 2026-01-07 |") {
		t.Fatalf("index output: err %v\n%s", err, stdout)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("index wrote into the directory: %v", entries)
	}
}

// `dotf lessons --help` is what an agent reads first. It once promised a generated
// index, which is the committed file CLI-131 removed. The assertion reads the
// rendered help, so a Long added later is covered too.
func TestLessonsHelp_DoesNotPromiseACommittedIndex(t *testing.T) {
	var out bytes.Buffer
	c := newLessonsCmd()
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs([]string{"--help"})
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, phrase := range []string{"generated index", "committed index"} {
		if strings.Contains(out.String(), phrase) {
			t.Errorf("the lessons help still promises %q:\n%s", phrase, out.String())
		}
	}
}
