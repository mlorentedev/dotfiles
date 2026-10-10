package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func hookBindingRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "harness", "manifest.json"), `{"agents":{"bind":[
  {"agent":"claude","file":".claude/settings.json","format":"command-hook","matcher":true,
   "emit_hooks":[{"id":"gate","event":"PreToolUse","command":"harness gate --harness claude","timeout":5}]},
  {"agent":"pi","file":".pi/agent/extensions/dotfiles-gate.ts","format":"ts-extension","emit":false}]}}`)
	return repo
}

func runCheckHookBinding(t *testing.T, repo, home string, fix bool) string {
	t.Helper()
	sys := newSys(map[string]string{"HOME": home, "USERPROFILE": home, "DOTFILES_REPO_DIR": repo}, nil, nil)
	var buf bytes.Buffer
	checkHookBinding(sys, NewReport(&buf, true), fix)
	return buf.String()
}

// #2232: a settings file without the gate hook FAILs naming the harness and the
// remedy, and writes nothing; --fix binds it, keeping the rest of the file, and
// the next check passes. A target that does not emit is reported skipped.
func TestCheckHookBinding_FailsOnDriftAndFixBindsIt(t *testing.T) {
	repo, home := hookBindingRepo(t), t.TempDir()
	settings := filepath.Join(home, ".claude", "settings.json")
	writeFile(t, settings, `{"model":"opus"}`)

	out := runCheckHookBinding(t, repo, home, false)
	if got := statusOfLine(out, "claude: hooks in .claude/settings.json differ"); got != StatusFail {
		t.Errorf("want FAIL, got %v\n%s", got, out)
	}
	if !strings.Contains(out, "dotf doctor --fix") {
		t.Errorf("the FAIL must name its remedy:\n%s", out)
	}
	if got := statusOfLine(out, "pi: declared emit:false"); got != StatusSkip {
		t.Errorf("want pi SKIP, got %v\n%s", got, out)
	}
	if raw, _ := os.ReadFile(settings); string(raw) != `{"model":"opus"}` {
		t.Fatalf("the check wrote without --fix:\n%s", raw)
	}

	out = runCheckHookBinding(t, repo, home, true)
	if got := statusOfLine(out, "claude: hooks bound in .claude/settings.json"); got != StatusFix {
		t.Errorf("want FIX, got %v\n%s", got, out)
	}
	raw, _ := os.ReadFile(settings)
	if !strings.Contains(string(raw), "harness gate --harness claude") || !strings.Contains(string(raw), `"opus"`) {
		t.Errorf("--fix did not bind the gate, or lost a key:\n%s", raw)
	}

	out = runCheckHookBinding(t, repo, home, false)
	if got := statusOfLine(out, "claude: hooks current in .claude/settings.json"); got != StatusPass {
		t.Errorf("after --fix want PASS, got %v\n%s", got, out)
	}
}

func TestCheckHookBinding_SkipsWithoutAHome(t *testing.T) {
	sys := newSys(map[string]string{"DOTFILES_REPO_DIR": hookBindingRepo(t)}, nil, nil)
	var buf bytes.Buffer
	checkHookBinding(sys, NewReport(&buf, true), true)
	if got := statusOfLine(buf.String(), "HOME is not set"); got != StatusSkip {
		t.Errorf("want SKIP, got %v\n%s", got, buf.String())
	}
}
