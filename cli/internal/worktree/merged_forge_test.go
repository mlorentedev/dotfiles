package worktree

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// #2759: gh talks to GitHub only. On an origin hosted elsewhere it fails with a
// bare exit status, which says nothing about why the branch was kept.
func TestMergedPRHeadsNamesAnOriginGHCannotAsk(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the gh stub is a shell script")
	}
	repo := t.TempDir()
	gitOut(t, repo, "init", "-q")
	gitOut(t, repo, "remote", "add", "origin", "https://gitea.example.test/manu/kubelab.git")
	bin := t.TempDir()
	marker := filepath.Join(bin, "gh-ran")
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\ntouch "+marker+"\nexit 1\n"), 0o755); err != nil { //nolint:gosec // an executable test stub
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err := ghMergedPRHeads(repo, "fix/x")
	if err == nil || !strings.Contains(err.Error(), "gitea.example.test") || !strings.Contains(err.Error(), "GitHub only") {
		t.Fatalf("want an error naming the forge, got %v", err)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Error("gh ran against an origin it cannot ask")
	}
}
