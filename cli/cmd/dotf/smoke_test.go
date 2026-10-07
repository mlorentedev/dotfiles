package main

// The clean-HOME smoke (CLI-100, #2081): the BUILT binary, run the way a user
// outside this repository runs it. Every other test is in-process, from inside
// the checkout, with a HOME the test shaped; none of them sees a command whose
// construction needs a machine.json, a vault or a checkout and fails without
// one. This runs only when DOTF_SMOKE_BIN names a binary, so `go test ./...`
// stays in-process; the cli-smoke workflow builds one and sets it.

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/cmd"
	"github.com/spf13/cobra"
)

func TestCleanHomeSmoke(t *testing.T) {
	bin := os.Getenv("DOTF_SMOKE_BIN")
	if bin == "" {
		// In the workflow that exists to run it, a skip is a green job that
		// smoke-tested nothing: a renamed variable must fail, not pass.
		if os.Getenv("GITHUB_WORKFLOW") == "cli-smoke" {
			t.Fatal("DOTF_SMOKE_BIN is unset inside the cli-smoke workflow")
		}
		t.Skip("DOTF_SMOKE_BIN is unset: the clean-HOME smoke runs a built binary (#2081)")
	}
	bin, err := filepath.Abs(bin)
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	work := t.TempDir()
	if repo := enclosingGitRepo(work); repo != "" {
		t.Fatalf("the working directory %s is inside the git repository %s: the smoke would not run outside a checkout", work, repo)
	}
	env := cleanEnv(t, home)

	runs := [][]string{{"version"}, {"--help"}}
	for _, path := range commandPaths(cmd.New("smoke", "")) {
		runs = append(runs, append(path, "--help"))
	}
	if len(runs) < 3 {
		t.Fatalf("the command tree enumerated no subcommands; a loop over nothing passes")
	}
	for _, args := range runs {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			c := exec.CommandContext(ctx, bin, args...) // #nosec G204 -- the binary under test, args from its own command tree
			c.Dir = work
			c.Env = env
			out, err := c.CombinedOutput()
			if err != nil {
				t.Errorf("dotf %s failed under a clean HOME: %v\n%s", strings.Join(args, " "), err, out)
			}
			if s := string(out); strings.Contains(s, "panic:") || strings.Contains(s, "goroutine ") {
				t.Errorf("dotf %s panicked under a clean HOME:\n%s", strings.Join(args, " "), s)
			}
		})
	}
}

// commandPaths lists every non-hidden command below root, as argument paths.
func commandPaths(root *cobra.Command) [][]string {
	var paths [][]string
	var walk func(c *cobra.Command, prefix []string)
	walk = func(c *cobra.Command, prefix []string) {
		for _, sub := range c.Commands() {
			if sub.Hidden {
				continue
			}
			p := append(append([]string{}, prefix...), sub.Name())
			paths = append(paths, p)
			walk(sub, p)
		}
	}
	walk(root, nil)
	return paths
}

// cleanEnv is a fresh user's environment, built rather than inherited:
// inheriting the runner's is the gap this test exists to close. On Windows a
// fresh profile still has its AppData directories and SystemRoot, which the
// standard library reads.
func cleanEnv(t *testing.T, home string) []string {
	t.Helper()
	tmp := filepath.Join(home, "tmp")
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		root := os.Getenv("SystemRoot")
		return []string{
			"USERPROFILE=" + home, "HOME=" + home,
			"APPDATA=" + filepath.Join(home, "AppData", "Roaming"),
			"LOCALAPPDATA=" + filepath.Join(home, "AppData", "Local"),
			"TEMP=" + tmp, "TMP=" + tmp,
			"SystemRoot=" + root, "PATH=" + filepath.Join(root, "System32"),
		}
	}
	return []string{"HOME=" + home, "TMPDIR=" + tmp, "PATH=/usr/bin:/bin"}
}

// enclosingGitRepo is the nearest ancestor of dir holding a .git, or "".
func enclosingGitRepo(dir string) string {
	for d := dir; ; {
		if _, err := os.Lstat(filepath.Join(d, ".git")); err == nil {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}
