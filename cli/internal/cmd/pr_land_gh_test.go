package cmd

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// pr land's real gh runner keeps gh's own reason (#2087). A queue stopped on
// "gh pr update-branch: exit status 1" with the reason gone; the other tests
// replace prLandOptions, so only a real failing process pins this call site.
func TestPrLandGhRunnerKeepsGhsStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake gh is a shell script")
	}
	bin := t.TempDir()
	script := "#!/bin/sh\necho 'GraphQL: Head branch was modified' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	_, err := prLandOptions("o/r", "").Run(context.Background(), "pr", "update-branch", "1")
	if err == nil || !strings.Contains(err.Error(), "GraphQL: Head branch was modified") {
		t.Errorf("pr land's gh runner error = %v; want gh's stderr in it", err)
	}
}
