package doctor

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

func presenceDoctorRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "harness", "manifest.json"), `{"version":1,"agents":{"record_dir":"harness/agents","presence":[
  {"agent":"claude","file":".claude/CLAUDE.md"},
  {"agent":"pi","file":".pi/agent/AGENTS.md"},
  {"agent":"copilot","file":".copilot/copilot-instructions.md","requires_command":"copilot"}]}}`)
	writeFile(t, filepath.Join(repo, "harness", "agents", "curator", "AGENT.md"),
		"---\nname: curator\ndescription: x\nkind: invocable\nskills: [vault-doctor]\n---\n")
	return repo
}

func runCheckAgentPresence(t *testing.T, repo, home string, onPath []string) string {
	t.Helper()
	sys := newSys(map[string]string{"HOME": home, "USERPROFILE": home, "DOTFILES_REPO_DIR": repo}, onPath, nil)
	var buf bytes.Buffer
	rep := NewReport(&buf, true)
	checkAgentPresence(sys, rep)
	return buf.String()
}

// AC4 (HARNESS-092, #1326): current → PASS; a file with no region → WARN naming
// it and the remedy; a region whose sha predates the roster → WARN "stale"; a
// copilot target on a box without copilot is not compared.
func TestCheckAgentPresence_ByStatus(t *testing.T) {
	t.Run("all injected → PASS", func(t *testing.T) {
		repo, home := presenceDoctorRepo(t), t.TempDir()
		for _, f := range []string{".claude/CLAUDE.md", ".pi/agent/AGENTS.md"} {
			writeFile(t, filepath.Join(home, filepath.FromSlash(f)), "intro\n")
		}
		if _, err := harness.DeployPresence(repo, home); err != nil {
			t.Fatal(err)
		}
		out := runCheckAgentPresence(t, repo, home, nil)
		if got := statusOfLine(out, "presence current in 2"); got != StatusPass {
			t.Errorf("want PASS, got %v\n%s", got, out)
		}
	})

	t.Run("no region → WARN naming the file and dotf harness presence", func(t *testing.T) {
		repo, home := presenceDoctorRepo(t), t.TempDir()
		writeFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), "intro\n")
		out := runCheckAgentPresence(t, repo, home, nil)
		if got := statusOfLine(out, "no presence region in .claude/CLAUDE.md"); got != StatusWarn {
			t.Errorf("want WARN, got %v\n%s", got, out)
		}
		if statusOfLine(out, "presence current") == StatusPass {
			t.Errorf("a missing region must not PASS:\n%s", out)
		}
	})

	t.Run("roster changed after injection → WARN stale", func(t *testing.T) {
		repo, home := presenceDoctorRepo(t), t.TempDir()
		writeFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), "intro\n")
		if _, err := harness.DeployPresence(repo, home); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(repo, "harness", "agents", "curator", "AGENT.md"),
			"---\nname: curator\ndescription: x\nkind: invocable\nskills: [vault-doctor, crystallize]\n---\n")
		out := runCheckAgentPresence(t, repo, home, nil)
		if got := statusOfLine(out, "stale presence in .claude/CLAUDE.md"); got != StatusWarn {
			t.Errorf("want WARN stale, got %v\n%s", got, out)
		}
	})

	t.Run("copilot target without copilot is not compared; with it, it is", func(t *testing.T) {
		repo, home := presenceDoctorRepo(t), t.TempDir()
		writeFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), "intro\n")
		writeFile(t, filepath.Join(home, ".copilot", "copilot-instructions.md"), "intro\n")
		if _, err := harness.DeployPresence(repo, home); err != nil {
			t.Fatal(err)
		}
		// Break copilot's region only.
		writeFile(t, filepath.Join(home, ".copilot", "copilot-instructions.md"), "intro\n")
		out := runCheckAgentPresence(t, repo, home, nil)
		if got := statusOfLine(out, "presence current"); got != StatusPass {
			t.Errorf("without copilot on PATH its file is not a surface; want PASS, got %v\n%s", got, out)
		}
		out = runCheckAgentPresence(t, repo, home, []string{"copilot"})
		if got := statusOfLine(out, "no presence region in .copilot/copilot-instructions.md"); got != StatusWarn {
			t.Errorf("with copilot on PATH the missing region is drift; got %v\n%s", got, out)
		}
	})

	t.Run("no repo → SKIP", func(t *testing.T) {
		t.Chdir(t.TempDir())
		sys := newSys(map[string]string{"HOME": t.TempDir()}, nil, nil)
		var buf bytes.Buffer
		rep := NewReport(&buf, true)
		checkAgentPresence(sys, rep)
		if got := statusOfLine(buf.String(), "repo not found"); got != StatusSkip {
			t.Errorf("want SKIP, got %v\n%s", got, buf.String())
		}
	})
}
