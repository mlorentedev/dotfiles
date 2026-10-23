package cmd

import (
	"bytes"
	"os"
	"os/exec"
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

// A tracked lesson moves with git mv, so the rename is staged as one; the
// command names the ambiguous link it left and a second run changes nothing.
func TestLessonsRenumber_MovesATrackedLessonWithGitMvAndIsIdempotent(t *testing.T) {
	repo := t.TempDir()
	dir := filepath.Join(repo, "docs", "lessons")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := func(title, extra string) string {
		return "---\ntitle: \"" + title + "\"\ndate: \"2026-10-05\"\n---\n\n# " + title + "\n\n" + extra
	}
	for name, b := range map[string]string{
		"lesson-497-mine.md":   body("Mine", ""),
		"lesson-497-theirs.md": body("Theirs", ""),
		"lesson-498-cites.md":  body("Cites", "[[lesson-497-mine]] and [[lesson-497]]\n"),
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(b), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	src := filepath.Join(dir, "lesson-497-mine.md")

	stdout, _, err := execute(t, "lessons", "renumber", src, "--next", "--dir", dir)
	if err != nil {
		t.Fatalf("renumber: %v\n%s", err, stdout)
	}
	for _, must := range []string{"lesson-499-mine.md", "lesson-498-cites.md links [[lesson-497]]"} {
		if !strings.Contains(stdout, must) {
			t.Errorf("output lacks %q:\n%s", must, stdout)
		}
	}
	staged, err := exec.Command("git", "-C", repo, "diff", "--cached", "--name-status").Output()
	if err != nil || !strings.Contains(string(staged), "lesson-499-mine.md") {
		t.Errorf("the rename is not staged (git mv): %v\n%s", err, staged)
	}

	stdout, _, err = execute(t, "lessons", "renumber", src, "--next", "--dir", dir)
	if err != nil || !strings.Contains(stdout, "already") {
		t.Errorf("second run: err %v\n%s", err, stdout)
	}
}

func TestLessonsRenumber_WantsExactlyOneOfNextAndTo(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{
		{"lessons", "renumber", filepath.Join(dir, "lesson-001-a.md"), "--dir", dir},
		{"lessons", "renumber", filepath.Join(dir, "lesson-001-a.md"), "--next", "--to", "5", "--dir", dir},
		{"lessons", "renumber", filepath.Join(dir, "lesson-001-a.md"), "--to", "0", "--dir", dir},
	} {
		if _, _, err := execute(t, args...); err == nil {
			t.Errorf("%v: accepted", args)
		}
	}
}

// A relative --dir names the same lessons as the absolute file path: both are
// made absolute before they are compared.
func TestLessonsRenumber_AcceptsARelativeDir(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "docs", "lessons")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lesson-001-a.md"), []byte("---\ntitle: \"A\"\ndate: \"2026-10-05\"\n---\n\n# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	stdout, _, err := execute(t, "lessons", "renumber", "docs/lessons/lesson-001-a.md", "--to", "7", "--dir", "docs/lessons")
	if err != nil || !strings.Contains(stdout, "lesson-007-a.md") {
		t.Fatalf("relative --dir: err %v\n%s", err, stdout)
	}
}
