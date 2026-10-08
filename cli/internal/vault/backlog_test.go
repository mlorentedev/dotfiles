package vault

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/mem"
)

// integrityFixtures cover what the awk in check-backlog-integrity.sh decides:
// the status column, the bold full id, number reuse, and how lines are read.
var integrityFixtures = map[string]string{
	"empty":            "",
	"clean":            "- [ ] **A-1** x\n- [x] **A-2** y\n",
	"duplicate":        "- [ ] **A-1** x\n- [ ] **A-1** again\n",
	"contradiction":    "- [ ] **A-1** x\n- [X] **A-1** done\n",
	"tilde and dash":   "- [~] **A-1** x\n- [-] **A-1** y\n- [-] **A-1** z\n",
	"number reuse":     "- [ ] **BUG-020-pwsh** a\n- [x] **BUG-020-split** b\n- [ ] **WIN-002** c\n- [ ] **WIN-002a** d\n- [ ] **WIN-002** again\n",
	"no final newline": "- [ ] **A-1** x\n- [ ] **A-1** y",
	"crlf":             "- [ ] **A-1** x\r\n- [x] **A-1** y\r\n",
	"not entries":      "  - [ ] **A-1** indented\n- [ ] A-1 not bold\n- [y] **A-1** bad status\n* [ ] **A-1** star\n- [ ] **A-1** real\n",
	"order":            "- [ ] **B-2**\n- [ ] **A-1**\n- [x] **A-1**\n- [ ] **B-2**\n- [ ] **B-2-x**\n",
	"word in a note":   "- [ ] **X-1-DUPLICATE** a\n- [ ] **X-1-b** b\n",
}

func TestBacklogIntegrity(t *testing.T) {
	cases := map[string]struct {
		report string
		drift  bool
	}{
		"empty":         {"", false},
		"clean":         {"", false},
		"duplicate":     {"  DUPLICATE: A-1 — 2 entries\n", true},
		"contradiction": {"  CONTRADICTION: A-1 — 2 entries, marked BOTH open and done\n", true},
		"number reuse": {"  DUPLICATE: WIN-002 — 2 entries\n" +
			"  NOTE: number BUG-020 reused by 2 different tickets (BUG-020-pwsh, BUG-020-split) — advisory, not drift\n" +
			"  NOTE: number WIN-002 reused by 2 different tickets (WIN-002, WIN-002a) — advisory, not drift\n", true},
		"word in a note": {"  NOTE: number X-1 reused by 2 different tickets (X-1-DUPLICATE, X-1-b) — advisory, not drift\n", true},
	}
	dir := t.TempDir()
	for name, want := range cases {
		path := filepath.Join(dir, name+".md")
		if err := os.WriteFile(path, []byte(integrityFixtures[name]), 0o600); err != nil {
			t.Fatal(err)
		}
		report, drift, err := BacklogIntegrity(path)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if want.report != "" {
			want.report = path + ":\n" + want.report
		}
		if report != want.report || drift != want.drift {
			t.Errorf("%s: got (%q, %v), want (%q, %v)", name, report, drift, want.report, want.drift)
		}
	}
}

func TestBacklogIntegrityUnreadableFileIsAnError(t *testing.T) {
	if _, _, err := BacklogIntegrity(filepath.Join(t.TempDir(), "missing.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("err = %v, want not-exist", err)
	}
}

// mergedRepo is a repository with archived specs for A-1 and B-2.
func mergedRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	for _, id := range []string{"A-1", "B-2"} {
		if err := os.MkdirAll(filepath.Join(repo, "specs", "archive", id), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

const mergedTasks = "- [ ] **A-1** open\n- [x] **B-2** done\n- [X] **B-2** done too\n- [~] **A-1** again\n- [-] **C-3** not archived\n- [ ] **B-2-x** other id\n"

func TestBacklogMerged(t *testing.T) {
	tasks := filepath.Join(t.TempDir(), "11-tasks.md")
	if err := os.WriteFile(tasks, []byte(mergedTasks), 0o600); err != nil {
		t.Fatal(err)
	}
	report, stale, err := BacklogMerged(tasks, mergedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	line := "  STALE-OPEN: A-1 — archived spec exists (specs/archive/A-1/); verify + tick [x]\n"
	if report != line+line || !stale {
		t.Errorf("got (%q, %v), want A-1 twice: each open entry is reported, and a done B-2 is not", report, stale)
	}

	if report, stale, _ := BacklogMerged(tasks, t.TempDir()); report != "" || stale {
		t.Errorf("a repo with no specs/archive has nothing to cross-reference, got (%q, %v)", report, stale)
	}
}

func TestBacklogMergedInfersTheRepoFromHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "Projects", "demo", "specs", "archive", "A-1"), 0o755); err != nil {
		t.Fatal(err)
	}
	projDir := filepath.Join(t.TempDir(), "10_projects", "demo")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tasks := filepath.Join(projDir, "11-tasks.md")
	if err := os.WriteFile(tasks, []byte("- [ ] **A-1** open\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, stale, err := BacklogMerged(tasks, ""); err != nil || !stale {
		t.Errorf("stale = %v, err = %v: $HOME/Projects/demo/specs/archive/A-1 exists", stale, err)
	}
}

// TestBacklogChecksMatchTheScripts runs each script and its port on the same
// input and compares stdout and exit status. It holds only while the scripts
// exist (scripts/vault.sh still exposes them); delete it with them.
func TestBacklogChecksMatchTheScripts(t *testing.T) {
	bash := mem.ResolveBash()
	if bash == "" {
		t.Skip("bash not found: the scripts cannot run here")
	}
	scripts := filepath.Join(goldenRepoRoot(t), "scripts")
	run := func(t *testing.T, args ...string) (string, int) {
		t.Helper()
		var stdout bytes.Buffer
		cmd := exec.Command(bash, args...)
		cmd.Stdout = &stdout
		cmd.Env = os.Environ() // carries a HOME set with t.Setenv
		err := cmd.Run()
		var ee *exec.ExitError
		switch {
		case err == nil:
			return stdout.String(), 0
		case errors.As(err, &ee):
			return stdout.String(), ee.ExitCode()
		}
		t.Fatalf("run %v: %v", args, err)
		return "", 0
	}
	code := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}

	dir := t.TempDir()
	for name, body := range integrityFixtures {
		t.Run("integrity/"+name, func(t *testing.T) {
			path := filepath.Join(dir, filepath.Base(t.Name())+".md")
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			wantOut, wantCode := run(t, filepath.Join(scripts, "check-backlog-integrity.sh"), path)
			report, drift, err := BacklogIntegrity(path)
			if err != nil {
				t.Fatal(err)
			}
			if report != wantOut || code(drift) != wantCode {
				t.Errorf("Go (%q, exit %d) != shell (%q, exit %d)", report, code(drift), wantOut, wantCode)
			}
		})
	}

	t.Run("merged", func(t *testing.T) {
		tasks := filepath.Join(dir, "11-tasks.md")
		if err := os.WriteFile(tasks, []byte(mergedTasks), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, repo := range []string{mergedRepo(t), t.TempDir()} {
			wantOut, wantCode := run(t, filepath.Join(scripts, "check-backlog-merged.sh"), tasks, "--repo", repo)
			report, stale, err := BacklogMerged(tasks, repo)
			if err != nil {
				t.Fatal(err)
			}
			if report != wantOut || code(stale) != wantCode {
				t.Errorf("repo %s: Go (%q, exit %d) != shell (%q, exit %d)", repo, report, code(stale), wantOut, wantCode)
			}
		}
	})

	// The path health takes: no --repo, so both infer $HOME/Projects/<proj>.
	t.Run("merged/repo from HOME", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		if err := os.MkdirAll(filepath.Join(home, "Projects", "demo", "specs", "archive", "A-1"), 0o755); err != nil {
			t.Fatal(err)
		}
		projDir := filepath.Join(t.TempDir(), "10_projects", "demo")
		if err := os.MkdirAll(projDir, 0o755); err != nil {
			t.Fatal(err)
		}
		tasks := filepath.Join(projDir, "11-tasks.md")
		if err := os.WriteFile(tasks, []byte(mergedTasks), 0o600); err != nil {
			t.Fatal(err)
		}
		wantOut, wantCode := run(t, filepath.Join(scripts, "check-backlog-merged.sh"), tasks)
		report, stale, err := BacklogMerged(tasks, "")
		if err != nil {
			t.Fatal(err)
		}
		if wantCode != 1 || report != wantOut || code(stale) != wantCode {
			t.Errorf("Go (%q, exit %d) != shell (%q, exit %d); the shell must find the archive", report, code(stale), wantOut, wantCode)
		}
	})
}
