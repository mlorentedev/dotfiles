package worktree

import (
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
