package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/agy"
)

// The bare deploy writes agy's MCP configs (CLI-105): until it did, only the
// setup twins wrote them, and a Mac converged by dotf had none.
func TestDeployCmd_BareDeployWritesAgyMCPConfigs(t *testing.T) {
	withPath(t, "uv")
	repo, home := pluginRepo(t), t.TempDir()
	writeMirrorFixture(t, filepath.Join(repo, agy.MCPRel),
		`{"mcpServers":{"hive-vault":{"command":"uvx","args":["hive-vault"],"env":{"VAULT_PATH":"${VAULT_PATH}"}}}}`)
	fake := &fakeClaude{registered: map[string]string{"web": webRegistered}, listed: "a@m b@m"}

	out, err := runDeployWithClaude(t, repo, home, []string{"--dry-run"}, fake)
	if err != nil {
		t.Fatalf("dry run: %v\n%s", err, out)
	}
	master := filepath.Join(home, ".gemini", "config", "mcp_config.json")
	if !strings.Contains(out, "would deploy") || !strings.Contains(out, master) {
		t.Errorf("a plan must name the master it would write:\n%s", out)
	}
	if _, err := os.Stat(master); !os.IsNotExist(err) {
		t.Fatalf("a dry run wrote %s", master)
	}

	if out, err = runDeployWithClaude(t, repo, home, nil, fake); err != nil {
		t.Fatalf("apply: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(master)
	if err != nil {
		t.Fatalf("master not written: %v\n%s", err, out)
	}
	// Decoded, not grepped: JSON escapes a Windows path's backslashes.
	var doc struct {
		MCPServers map[string]struct {
			Env map[string]string `json:"env"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if got, want := doc.MCPServers["hive-vault"].Env["VAULT_PATH"], filepath.Join(home, "Projects", "knowledge"); got != want {
		t.Errorf("hive-vault VAULT_PATH = %q, want %q", got, want)
	}
	if !strings.Contains(out, "no vault at") {
		t.Errorf("a missing vault must be reported:\n%s", out)
	}

	if out, err = runDeployWithClaude(t, repo, home, nil, fake); err != nil {
		t.Fatalf("second apply: %v\n%s", err, out)
	}
	if !strings.Contains(out, "in sync") || !strings.Contains(out, "3 file(s)") {
		t.Errorf("a second deploy must leave agy's three files alone:\n%s", out)
	}
}
