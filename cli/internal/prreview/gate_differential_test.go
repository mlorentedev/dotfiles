package prreview

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

// Until scripts/pr-agent-push-gate.sh is deleted, it is the specification: the
// same two files through the script and through Gate must print the same three
// lines and exit 0. The script goes when the workflow switches to the command,
// and this test goes with it. TestGate_Divergences' cases are not here: on those
// the port differs on purpose.
func TestGateMatchesTheScript(t *testing.T) {
	script := requireScript(t, "pr-agent-push-gate.sh", "bash", "jq")
	for _, c := range gateCases() {
		t.Run(c.name, func(t *testing.T) {
			f := &gateFixture{}
			c.build(f)
			comments, commits := f.files(t)
			dir := t.TempDir()
			args := []string{script, "--comments", writeGateFile(t, dir, "comments.json", comments),
				"--commits", writeGateFile(t, dir, "commits.json", commits),
				"--threshold", strconv.Itoa(c.limit()), "--sender-type", c.sender}

			var stdout bytes.Buffer
			cmd := exec.CommandContext(t.Context(), "bash", args...)
			cmd.Stdout = &stdout
			if err := cmd.Run(); err != nil {
				var ee *exec.ExitError
				if !errors.As(err, &ee) {
					t.Fatalf("running the script: %v", err)
				}
				t.Fatalf("the script exited %d", ee.ExitCode())
			}

			got := Gate(GateInputs{Comments: comments, Commits: commits, Threshold: c.limit(), SenderType: c.sender}).Output()
			if want := stdout.String(); got != want {
				t.Errorf("the port differs from the script\n--- script ---\n%s--- port ---\n%s", want, got)
			}
		})
	}
}

// writeGateFile writes one input, or leaves no file for nil, and returns its path.
func writeGateFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if content != nil {
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return path
}
