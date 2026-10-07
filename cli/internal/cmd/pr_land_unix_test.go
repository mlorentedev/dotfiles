//go:build !windows

package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The prland package tells "no checks yet" from a gh failure by gh's stderr
// (#2056), which only works if the real runner puts stderr in the error. This
// runs that runner against a gh stub rather than trusting a fake's format.
func TestPrLandRunner_ErrorCarriesGhStderr(t *testing.T) {
	stubs := t.TempDir()
	script := "#!/bin/sh\necho \"no checks reported on the 'feat/x' branch\" >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(stubs, "gh"), []byte(script), 0o755); err != nil { //nolint:gosec // an executable stub
		t.Fatalf("write stub: %v", err)
	}
	t.Setenv("PATH", stubs)

	out, err := prLandOptions("", "").Run(context.Background(), "pr", "checks", "30", "--json", "name,bucket")
	if len(out) != 0 || err == nil {
		t.Fatalf("want empty stdout and an error, got %q, %v", out, err)
	}
	if !strings.Contains(err.Error(), "no checks reported on the") {
		t.Errorf("error %q lost gh's stderr", err)
	}
}
