package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CLI-093 (#1953): Orca rewrites ~/.copilot/hooks/orca.json with timeoutSec 5
// on every install, and only the Windows setup re-tuned it. A bare `dotf
// deploy`, which both setups run, now converges it on every OS.
const orcaHooks5s = `{"version": 1, "hooks": {"PreToolUse": [{"type": "command", "powershell": "y", "timeoutSec": 5}]}}`

func writeOrcaHooks(t *testing.T, home string) string {
	t.Helper()
	p := filepath.Join(home, ".copilot", "hooks", "orca.json")
	writeMirrorFixture(t, p, orcaHooks5s)
	return p
}

func TestDeployCmd_TunesOrcaHooks(t *testing.T) {
	repo, home := twoConfigRepo(t), t.TempDir()
	hooks := writeOrcaHooks(t, home)

	out, err := runDeploy(t, repo, home, nil)
	if err != nil {
		t.Fatalf("deploy: %v\n%s", err, out)
	}
	raw, _ := os.ReadFile(hooks)
	if strings.Contains(string(raw), `"timeoutSec": 5`) || !strings.Contains(string(raw), `"timeoutSec": 30`) {
		t.Errorf("orca.json not tuned:\n%s\noutput:\n%s", raw, out)
	}
	if !strings.Contains(out, "tuned") {
		t.Errorf("deploy did not report the tune:\n%s", out)
	}
}

func TestDeployCmd_DryRunReportsOrcaDriftWithoutWriting(t *testing.T) {
	repo, home := twoConfigRepo(t), t.TempDir()
	hooks := writeOrcaHooks(t, home)

	out, err := runDeploy(t, repo, home, []string{"--dry-run"})
	if err != nil {
		t.Fatalf("deploy --dry-run: %v\n%s", err, out)
	}
	raw, _ := os.ReadFile(hooks)
	if string(raw) != orcaHooks5s {
		t.Errorf("--dry-run wrote orca.json:\n%s", raw)
	}
	if !strings.Contains(out, "would tune") || !strings.Contains(out, "orca-hooks") {
		t.Errorf("--dry-run did not report the drift:\n%s", out)
	}
}

// Without Orca there is nothing to tune, and deploy says so in one line.
func TestDeployCmd_NoOrcaIsNothingToDo(t *testing.T) {
	repo, home := twoConfigRepo(t), t.TempDir()
	out, err := runDeploy(t, repo, home, nil)
	if err != nil {
		t.Fatalf("deploy: %v\n%s", err, out)
	}
	if !strings.Contains(out, "skipped   orca-hooks") {
		t.Errorf("no Orca line:\n%s", out)
	}
}

// `dotf deploy <name>` deploys that one config and nothing else.
func TestDeployCmd_NamedConfigLeavesOrcaAlone(t *testing.T) {
	repo, home := twoConfigRepo(t), t.TempDir()
	hooks := writeOrcaHooks(t, home)
	if out, err := runDeploy(t, repo, home, []string{"one"}); err != nil {
		t.Fatalf("deploy one: %v\n%s", err, out)
	}
	if raw, _ := os.ReadFile(hooks); string(raw) != orcaHooks5s {
		t.Errorf("a named deploy touched orca.json:\n%s", raw)
	}
}
