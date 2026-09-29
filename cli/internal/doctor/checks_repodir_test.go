package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckRepoDirResolves drives the #696 guard: the DOTFILES_REPO_DIR cascade
// must resolve to a real git checkout. DOTFILES_REPO_DIR is set via the env
// (tier-1 of the cascade), so envpkg.ResolvePath returns it deterministically
// without a contract on disk.
func TestCheckRepoDirResolves(t *testing.T) {
	realCheckout := t.TempDir()
	if err := os.Mkdir(filepath.Join(realCheckout, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	worktree := t.TempDir()
	if err := os.WriteFile(filepath.Join(worktree, ".git"), []byte("gitdir: ../main/.git/worktrees/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	notGit := t.TempDir() // exists but carries no .git

	cases := []struct {
		name         string
		repoDir      string
		gitCheckout  bool
		wantFailures int
		wantSubstr   string
	}{
		{"real checkout -> pass", realCheckout, true, 0, "resolves to a checkout"},
		{"worktree checkout -> pass", worktree, true, 0, "resolves to a checkout"},
		{"missing path -> fail", filepath.Join(realCheckout, "nope"), false, 1, "missing path"},
		{"exists but not a git checkout -> fail", notGit, false, 1, "not a git checkout"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DOTFILES_REPO_DIR", tc.repoDir)
			cmdOut := map[string]string{}
			if tc.gitCheckout {
				cmdOut["git -C "+tc.repoDir+" rev-parse --is-inside-work-tree"] = "true\n"
			}
			var buf bytes.Buffer
			rep := capture(&buf)
			checkRepoDirResolves(newSys(nil, nil, cmdOut), rep)
			if rep.Failures() != tc.wantFailures {
				t.Fatalf("failures = %d, want %d\n%s", rep.Failures(), tc.wantFailures, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantSubstr) {
				t.Fatalf("output missing %q\n%s", tc.wantSubstr, buf.String())
			}
		})
	}
}
