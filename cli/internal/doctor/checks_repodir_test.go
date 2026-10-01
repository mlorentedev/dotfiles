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
	subdir := filepath.Join(worktree, "cli")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	notGit := t.TempDir() // exists but carries no .git

	cases := []struct {
		name         string
		repoDir      string
		gitRoot      string
		wantFailures int
		wantSubstr   string
	}{
		{"real checkout -> pass", realCheckout, realCheckout, 0, "resolves to a checkout"},
		{"worktree checkout -> pass", worktree, worktree, 0, "resolves to a checkout"},
		{"checkout subdirectory -> fail", subdir, worktree, 1, "not the checkout root"},
		{"missing path -> fail", filepath.Join(realCheckout, "nope"), "", 1, "missing path"},
		{"exists but not a git checkout -> fail", notGit, "", 1, "not a git checkout"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DOTFILES_REPO_DIR", tc.repoDir)
			cmdOut := map[string]string{}
			if tc.gitRoot != "" {
				cmdOut["git -C "+tc.repoDir+" rev-parse --show-toplevel"] = tc.gitRoot + "\n"
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

	t.Run("symlinked checkout path equals physical git root", func(t *testing.T) {
		link := filepath.Join(t.TempDir(), "checkout-link")
		if err := os.Symlink(realCheckout, link); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if !sameCheckoutRoot(link, realCheckout) {
			t.Fatalf("symlink %q and physical root %q must compare equal", link, realCheckout)
		}
	})

	t.Run("git probe clears inherited repository overrides", func(t *testing.T) {
		t.Setenv("DOTFILES_REPO_DIR", worktree)
		sys := newSys(nil, nil, nil)
		sys.CommandOutputEnv = func(env []string, name string, args ...string) (string, error) {
			for _, entry := range env {
				if strings.HasPrefix(entry, "GIT_DIR=") || strings.HasPrefix(entry, "GIT_WORK_TREE=") {
					t.Fatalf("repository-local Git override leaked into probe: %q", entry)
				}
			}
			return worktree + "\n", nil
		}

		var buf bytes.Buffer
		rep := capture(&buf)
		checkRepoDirResolves(sys, rep)
		if rep.Failures() != 0 {
			t.Fatalf("valid worktree rejected under inherited Git overrides:\n%s", buf.String())
		}
	})
}
