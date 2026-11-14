//go:build !windows

package lane

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Exec replaces the process, so the effect on the process it starts is seen from
// outside: a child test binary calls Exec, and the `dotf` it finds on PATH, a
// script standing in for the wrapper, writes the HERDR_AGENT it was started with.
// herdr reads that value from the environment the foreground process was
// started with, which is why Exec must hand it to the image it replaces itself
// with (lesson 493, #3430).
func TestExec_StartsTheWrapperWithTheAgentNamedForHerdr(t *testing.T) {
	if os.Getenv("LANE_EXEC_ENV_HELPER") == "1" {
		_ = Exec([]string{"dotf", "secrets", "run", "--only", "NAN_API_KEY", "--", "opencode", "--prompt", "hi"})
		os.Exit(3) // Exec returned: the launch failed
	}
	dir := t.TempDir()
	out := filepath.Join(dir, "seen")
	script := "#!/bin/sh\nprintf '%s' \"$HERDR_AGENT\" > \"$LANE_EXEC_ENV_OUT\"\n"
	if err := os.WriteFile(filepath.Join(dir, "dotf"), []byte(script), 0o755); err != nil { //nolint:gosec // a fixture that must be executable
		t.Fatal(err)
	}
	for _, inherited := range []string{"", "stale"} {
		_ = os.Remove(out)
		cmd := exec.Command(os.Args[0], "-test.run=^TestExec_StartsTheWrapperWithTheAgentNamedForHerdr$")
		cmd.Env = append(os.Environ(), "LANE_EXEC_ENV_HELPER=1", "LANE_EXEC_ENV_OUT="+out,
			"PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
		if inherited != "" {
			cmd.Env = append(cmd.Env, "HERDR_AGENT="+inherited)
		}
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("helper: %v\n%s", err, b)
		}
		got, err := os.ReadFile(out)
		if err != nil || strings.TrimSpace(string(got)) != "opencode" {
			t.Errorf("inherited %q: the wrapper saw HERDR_AGENT=%q (err=%v), want opencode", inherited, got, err)
		}
	}
}
