package worktree

import (
	"reflect"
	"testing"
)

func TestIsDisposableIgnoredPath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{".dotf-worktree.json", true},
		{".DS_Store", true},
		{"Thumbs.db", true},
		{"coverage.out", true},
		{"coverage.html", true},
		{".coverage", true},
		{"cli/dotf", true},
		{"dotf", true},
		{"dotf.exe", true},
		{"cli/dotf.exe", true},
		{"specs/CLI-075-dotf-worktree-lifecycle/review-transcript.jsonl", true},
		{"specs/CLI-075-dotf-worktree-lifecycle/review-transcript.jsonl.stderr", true},
		{"node_modules/", true},
		{"node_modules/foo/index.js", true},
		{"web/node_modules/bar.ts", true},
		{"target/", true},
		{"target/debug/app", true},
		{".venv/", true},
		{".venv/bin/python", true},
		{"venv/lib/site.py", true},
		{"build/bundle.js", true},
		{"dist/index.html", true},
		{"out/main", true},
		{".cache/item", true},
		{".terraform/providers", true},
		{"site/.astro/", true},
		{"site/.astro/types.d.ts", true},
		{"foo.pyc", true},
		{"lib.so", true},
		// Non-disposable paths (scratchpad files, secrets, local configs)
		{".env", false},
		{".env.local", false},
		{".env.production", false},
		{"notes.txt", false},
		{"scratch.md", false},
		{"docs/notes.tmp", false},
		{"backup.tar.gz", false},
		{"my_script.sh", false},
		{"data.json", false},
	}

	for _, tt := range tests {
		got := IsDisposableIgnoredPath(tt.path)
		if got != tt.expected {
			t.Errorf("IsDisposableIgnoredPath(%q) = %v, want %v", tt.path, got, tt.expected)
		}
	}
}

func TestHasNonDisposableIgnored(t *testing.T) {
	disposableOutput := `!! .dotf-worktree.json
!! node_modules/
!! .venv/
!! cli/dotf
`
	if HasNonDisposableIgnored(disposableOutput) {
		t.Errorf("expected disposable output to return false, got true")
	}

	nonDisposableOutput := `!! .dotf-worktree.json
!! node_modules/
!! .env
`
	if !HasNonDisposableIgnored(nonDisposableOutput) {
		t.Errorf("expected output with .env to return true, got false")
	}

	scratchOutput := `!! notes.txt
`
	if !HasNonDisposableIgnored(scratchOutput) {
		t.Errorf("expected output with notes.txt to return true, got false")
	}
}

// #1607: the refusal has to say WHICH ignored paths it found, or the operator
// cannot tell a build cache from a .env and reaches for --force.
func TestNonDisposableIgnoredNamesThePaths(t *testing.T) {
	out := "!! .dotf-worktree.json\n!! site/dist/\n!! site/.astro/\n!! .env\n!! site/public/beoe/\n"
	got := NonDisposableIgnored(out)
	want := []string{".env", "site/public/beoe/"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NonDisposableIgnored = %q, want %q", got, want)
	}
	if NonDisposableIgnored("!! node_modules/\n") != nil {
		t.Errorf("only disposable content must yield no paths")
	}
}

func TestSummarizePathsCountsWhatItDoesNotName(t *testing.T) {
	if got := summarizePaths([]string{"a", "b"}); got != "a, b" {
		t.Errorf("summarizePaths(2) = %q", got)
	}
	if got := summarizePaths([]string{"a", "b", "c", "d", "e"}); got != "a, b, c (+2 more)" {
		t.Errorf("summarizePaths(5) = %q", got)
	}
}
