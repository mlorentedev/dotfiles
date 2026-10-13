package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// TestRun_DockerComposeIsWiredIn proves doctor.Run still calls the compose
// check: a test of the function alone stays green when the call is deleted
// (the OPS-043 review showed it by removing the call from doctor.go).
func TestRun_DockerComposeIsWiredIn(t *testing.T) {
	home := t.TempDir()
	dotfiles := filepath.Join(home, ".dotfiles")
	writeFile(t, filepath.Join(dotfiles, "env-contract.json"),
		`{"env_vars":[],"required_path_entries":{"linux":[]},"required_binaries":[],"optional_binaries":[]}`)
	writeFile(t, filepath.Join(dotfiles, "versions.conf"), "GO_VERSION=1.26.0\n")

	var buf bytes.Buffer
	sys := newSys(map[string]string{"HOME": home, "DOTFILES_DIR": dotfiles},
		[]string{"docker"}, map[string]string{"docker compose version": "Docker Compose version v2.39.1"})
	if _, err := Run(Options{Out: &buf, System: sys, StartDir: home, Verbose: true}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(buf.String(), "compose v2 plugin: Docker Compose version v2.39.1") {
		t.Fatalf("the compose check did not run — is it still called from doctor.Run?\n%s", buf.String())
	}
}
