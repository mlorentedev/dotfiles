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
	for _, tc := range []struct {
		origin string
		ghRuns bool // false: the error names the forge and gh is never asked
	}{
		{"https://gitea.example.test/manu/kubelab.git", false},
		{"git@gitea.example.test:manu/kubelab.git", false},            // scp form, the usual one for a private Gitea
		{"ssh://git@gitea.example.test:2222/manu/kubelab.git", false}, // a port is not part of the host
		{"file:///srv/git/kubelab.git", true},                         // no host: gh keeps its own answer
	} {
		t.Run(tc.origin, func(t *testing.T) {
			repo := t.TempDir()
			gitOut(t, repo, "init", "-q")
			gitOut(t, repo, "remote", "add", "origin", tc.origin)
			bin := t.TempDir()
			marker := filepath.Join(bin, "gh-ran")
			if err := os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\ntouch "+marker+"\nexit 1\n"), 0o755); err != nil { //nolint:gosec // an executable test stub
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

			_, err := ghMergedPRHeads(repo, "fix/x")
			_, statErr := os.Stat(marker)
			if ran := statErr == nil; ran != tc.ghRuns {
				t.Fatalf("gh ran = %v, want %v (err %v)", ran, tc.ghRuns, err)
			}
			if tc.ghRuns {
				return
			}
			if err == nil || !strings.Contains(err.Error(), "origin is on gitea.example.test,") || !strings.Contains(err.Error(), "GitHub only") {
				t.Fatalf("want an error naming the forge, got %v", err)
			}
		})
	}
}
