//go:build unix

package cmd

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/evals"
)

// The command's real composition, end to end, with PATH holding only stubs: a
// `dotf` that verifies every secret and runs what follows `--`, and a `pi`
// that answers in the file the prompt names. Nothing real can be reached, so
// this is the one test that runs `dotf eval run` without --plan.
func TestEvalRunLaunchesThroughStubRunners(t *testing.T) {
	bin := t.TempDir()
	if git, err := exec.LookPath("git"); err == nil {
		if err := os.Symlink(git, filepath.Join(bin, "git")); err != nil {
			t.Fatal(err)
		}
	}
	stub := func(name, body string) {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	stub("dotf", `if [ "$1 $2" = "secrets verify" ]; then exit 0; fi
while [ "$#" -gt 0 ] && [ "$1" != "--" ]; do shift; done
shift
exec "$@"
`)
	cwdFile := filepath.Join(t.TempDir(), "pi-cwd")
	stub("pi", `pwd -P > '`+cwdFile+`'
printf '%s\n' '{"type":"message_end"}'
printf '%s\n' '| # | Severity | Finding |' '|---|---|---|' '| 1 | Major | count never increments |' > .dotf-eval/review.md
`)
	t.Setenv("PATH", bin)
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)

	stdout, _, err := execute(t, "eval", "run", "--repo-root", evalRepoRoot(t),
		"--candidate", "nan/glm5.3-flash", "--case", "bash-count-increment", "--epochs", "1")
	if err != nil {
		t.Fatalf("eval run: %v\n%s", err, stdout)
	}
	runs, err := filepath.Glob(filepath.Join(state, "dotfiles", "evals", "*", "runs", "*", "bash-count-increment", "1", "record.json"))
	if err != nil || len(runs) != 1 {
		t.Fatalf("record.json: %v %v\n%s", runs, err, stdout)
	}
	raw, err := os.ReadFile(runs[0])
	if err != nil {
		t.Fatal(err)
	}
	var rec evals.Record
	if err := json.Unmarshal(raw, &rec); err != nil || rec.Completion != evals.CompletionAnswered {
		t.Errorf("record = %s (%v)", raw, err)
	}
	// The tree is the directory the runner actually ran in, as the stub saw
	// it, so the check cannot pass by looking somewhere the tree never was.
	cwd, err := os.ReadFile(cwdFile)
	if err != nil || strings.TrimSpace(string(cwd)) == "" {
		t.Fatalf("the stub runner recorded no working directory: %v", err)
	}
	if _, err := os.Stat(strings.TrimSpace(string(cwd))); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the attempt's tree was left behind: %s (%v)", cwd, err)
	}
	if !strings.Contains(stdout, "1 attempts: 1 answered") {
		t.Errorf("the report lacks the count:\n%s", stdout)
	}
}
