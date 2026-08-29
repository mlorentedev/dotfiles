package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func presenceFixture(t *testing.T) (repo, home string) {
	t.Helper()
	repo, home = t.TempDir(), t.TempDir()
	writeMirrorFixture(t, filepath.Join(repo, "harness", "manifest.json"), `{"version":1,"agents":{"record_dir":"harness/agents","presence":[
  {"agent":"claude","file":".claude/CLAUDE.md"},
  {"agent":"opencode","file":".config/opencode/AGENTS.md"}]}}`)
	writeMirrorFixture(t, filepath.Join(repo, "harness", "agents", "curator", "AGENT.md"),
		"---\nname: curator\ndescription: x\nkind: invocable\nskills: [vault-doctor]\n---\n")
	writeMirrorFixture(t, filepath.Join(home, ".claude", "CLAUDE.md"), "intro\n")
	return repo, home
}

func runPresence(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	var out, errb bytes.Buffer
	cmd := newHarnessPresenceCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errb.String(), err
}

// The command reports per file what it did, on the stream the setup scripts
// read: injections and "current" on stdout, skips on stderr.
func TestHarnessPresenceCmd_ReportsEachTarget(t *testing.T) {
	repo, home := presenceFixture(t)

	out, errs, err := runPresence(t, "--repo-root", repo, "--home", home)
	if err != nil {
		t.Fatalf("%v\n%s%s", err, out, errs)
	}
	if !strings.Contains(out, "[deploy] presence -> "+filepath.Join(home, ".claude", "CLAUDE.md")+" (claude)") {
		t.Errorf("injection not reported:\n%s", out)
	}
	if !strings.Contains(errs, "presence target absent, skipping: "+filepath.Join(home, ".config", "opencode", "AGENTS.md")) {
		t.Errorf("absent target not reported on stderr:\n%s", errs)
	}
	raw, _ := os.ReadFile(filepath.Join(home, ".claude", "CLAUDE.md"))
	if !strings.Contains(string(raw), "vault-doctor") {
		t.Errorf("roster not written:\n%s", raw)
	}

	out, _, err = runPresence(t, "--repo-root", repo, "--home", home)
	if err != nil || !strings.Contains(out, "presence current") {
		t.Errorf("second run must report current: %v\n%s", err, out)
	}
}

func TestHarnessPresenceCmd_FailsOnABrokenRecord(t *testing.T) {
	repo, home := presenceFixture(t)
	writeMirrorFixture(t, filepath.Join(repo, "harness", "agents", "broken", "AGENT.md"), "---\nname: broken\nkind: invocable\nskills: [unterminated\n---\n")

	if _, _, err := runPresence(t, "--repo-root", repo, "--home", home); err == nil {
		t.Error("an unparseable skills list must fail the deploy, not render 'none'")
	}
}
