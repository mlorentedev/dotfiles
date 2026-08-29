package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A manifest entry that `requires` a command is skipped — and said so — when
// the command is absent (AI-039, #1322). The integration guard #1312 asserts
// ~/.copilot never appears on a box without copilot; a bare `dotf deploy` that
// wrote Copilot's settings everywhere would have broken it, and would have
// left a file nobody reads (#843).
func TestDeployCmd_SkipsAnEntryWhoseRequiredCommandIsAbsent(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	writeMirrorFixture(t, filepath.Join(repo, "ai", "deploy.json"), `{
  "version": 2,
  "configs": [
    {"name": "always", "src": "ai/always.json", "dst": "{HOME}/.always/config.json", "render": false, "mode": "0644"},
    {"name": "gated", "src": "ai/gated.json", "dst": "{HOME}/.gated/settings.json", "render": false, "mode": "0644", "strategy": "merge", "requires": "gatedtool"}
  ]
}`)
	writeMirrorFixture(t, filepath.Join(repo, "ai", "always.json"), `{"a":true}`)
	writeMirrorFixture(t, filepath.Join(repo, "ai", "gated.json"), `{"model":"m"}`)

	orig := deployCommandAvailable
	t.Cleanup(func() { deployCommandAvailable = orig })
	deployCommandAvailable = func(string) bool { return false }

	out, err := runDeploy(t, repo, home, nil)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "skipped   gated") || !strings.Contains(out, "gatedtool not installed") {
		t.Errorf("the skip must be reported by name and reason:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".gated")); !os.IsNotExist(err) {
		t.Error("a skipped entry must not create its destination directory")
	}
	if _, err := os.Stat(filepath.Join(home, ".always", "config.json")); err != nil {
		t.Errorf("the ungated entry must still deploy: %v", err)
	}

	deployCommandAvailable = func(string) bool { return true }
	out, err = runDeploy(t, repo, home, []string{"gated"})
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "deployed  gated") {
		t.Errorf("with the command present the entry deploys:\n%s", out)
	}
}
