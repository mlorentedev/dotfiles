package prtriage

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The real runner keeps gh's own reason (#2087). The other tests reach the
// fetch path through a stubbed ghRunner, which never produces an
// *exec.ExitError, so only a real failing process can pin this call site.
func TestExecGHKeepsGhsStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake gh is a shell script")
	}
	bin := t.TempDir()
	script := "#!/bin/sh\necho 'HTTP 502: upstream unavailable' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	_, err := execGH(context.Background(), "api", "x")
	if err == nil || !strings.Contains(err.Error(), "HTTP 502: upstream unavailable") {
		t.Errorf("execGH error = %v; want gh's stderr in it", err)
	}
}
