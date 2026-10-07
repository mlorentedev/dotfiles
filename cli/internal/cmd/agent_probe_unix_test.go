//go:build !windows

package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// With no --backend the probe chooses by which harness binary is on PATH, and
// the command reaches it through the real fork/exec. The negative half (nothing
// on PATH exhausts the chain) is TestAgentRun_NoHarnessOnPathExhaustsTheChain...;
// this is the positive half, with a stub standing in for the harness so it
// spends no quota. It is unix-only for the same reason internal/agent's
// subprocess tests are: the stub is a shell script.
func TestAgentRun_TheProbeSelectsABackendByTheHarnessBinaryOnPath(t *testing.T) {
	root := repoRootForTest(t)
	declareIdentity(t) // isolates PATH; the stub dir replaces it below

	stubs := t.TempDir()
	script := "#!/bin/sh\nprintf 'answered by the claude stub'\n"
	if err := os.WriteFile(filepath.Join(stubs, "claude"), []byte(script), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	t.Setenv("PATH", stubs)

	stdout, _, err := captureRealStreams(t,
		"agent", "run", "--role", "r", "--task", "t", "--tier", "mid",
		"--timeout", "1m", "--repo-root", root, "--semaphore-dir", t.TempDir(),
	)
	if err != nil {
		t.Fatalf("agent run: %v", err)
	}
	var rec struct {
		Status string `json:"status"`
		Pool   string `json:"pool"`
		Output string `json:"output"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &rec); jsonErr != nil {
		t.Fatalf("stdout is not a record: %v (%q)", jsonErr, stdout)
	}
	if rec.Status != "ok" {
		t.Errorf("status = %q, want ok", rec.Status)
	}
	if rec.Pool != "claude" {
		t.Errorf("pool = %q, want claude: it is the only harness on PATH", rec.Pool)
	}
	if !strings.Contains(rec.Output, "claude stub") {
		t.Errorf("output = %q, want the stub's answer: the dispatch never reached the process", rec.Output)
	}
}
