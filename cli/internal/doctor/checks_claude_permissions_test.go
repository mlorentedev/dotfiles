package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The rules #2349 corrected in the template, as an existing box still has them
// after a merge deploy (#2350).
const settingsWithMalformedRules = `{
  "permissions": {
    "deny": ["Bash(git push *--force*:*)", "Bash(git push *--force*)", "Bash(rm -rf ~/*:*)"],
    "allow": ["Bash(git status:*)"]
  },
  "model": "opus"
}`

func claudePermissionsBox(t *testing.T, settings string) (*System, string) {
	t.Helper()
	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cfg := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", cfg)
	dst := filepath.Join(cfg, "settings.json")
	if settings != "" {
		if err := os.WriteFile(dst, []byte(settings), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return newSys(map[string]string{"DOTFILES_REPO_DIR": repo, "HOME": t.TempDir()}, []string{"claude"}, nil), dst
}

func TestCheckClaudePermissionRules_NamesEachMalformedRuleAndChangesNothing(t *testing.T) {
	sys, dst := claudePermissionsBox(t, settingsWithMalformedRules)
	var buf bytes.Buffer
	checkClaudePermissionRules(sys, capture(&buf), false)

	out := buf.String()
	for _, r := range []string{"deny: Bash(git push *--force*:*)", "deny: Bash(rm -rf ~/*:*)"} {
		if !strings.Contains(out, "[FAIL] "+r) {
			t.Errorf("no FAIL for %s:\n%s", r, out)
		}
	}
	if strings.Contains(out, "Bash(git push *--force*) ") || strings.Contains(out, "git status") {
		t.Errorf("a well-formed rule was reported:\n%s", out)
	}
	if got, _ := os.ReadFile(dst); string(got) != settingsWithMalformedRules {
		t.Errorf("a check without --fix rewrote the file:\n%s", got)
	}
}

func TestCheckClaudePermissionRules_FixRemovesThemAndASecondRunPasses(t *testing.T) {
	sys, dst := claudePermissionsBox(t, settingsWithMalformedRules)
	var buf bytes.Buffer
	checkClaudePermissionRules(sys, capture(&buf), true)
	if out := buf.String(); strings.Count(out, "] removed ") != 2 || strings.Contains(out, "[FAIL]") {
		t.Fatalf("--fix should report two removals and no FAIL:\n%s", out)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	for _, keep := range []string{`"Bash(git push *--force*)"`, `"Bash(git status:*)"`, `"model": "opus"`} {
		if !strings.Contains(string(got), keep) {
			t.Errorf("--fix dropped %s:\n%s", keep, got)
		}
	}
	for _, gone := range []string{"*--force*:*", "~/*:*"} {
		if strings.Contains(string(got), gone) {
			t.Errorf("malformed rule %s survived:\n%s", gone, got)
		}
	}

	buf.Reset()
	checkClaudePermissionRules(sys, capture(&buf), true)
	if out := buf.String(); !strings.Contains(out, "[ OK ] no permission rule") || strings.Contains(out, "removed") {
		t.Errorf("a second run should pass with nothing to fix:\n%s", out)
	}
}

func TestCheckClaudePermissionRules_SkipsWithoutClaudeOrADeployedFile(t *testing.T) {
	sys, _ := claudePermissionsBox(t, "")
	var buf bytes.Buffer
	checkClaudePermissionRules(sys, capture(&buf), true)
	if out := buf.String(); !strings.Contains(out, "not deployed") {
		t.Errorf("an absent settings.json should be a SKIP:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("CLAUDE_CONFIG_DIR"), "settings.json")); !os.IsNotExist(err) {
		t.Error("--fix created a settings.json")
	}

	sys, _ = claudePermissionsBox(t, settingsWithMalformedRules)
	sys.LookPath = func(string) (string, error) { return "", os.ErrNotExist }
	buf.Reset()
	checkClaudePermissionRules(sys, capture(&buf), true)
	if out := buf.String(); !strings.Contains(out, "claude not in PATH") {
		t.Errorf("no claude should be a SKIP:\n%s", out)
	}
}

// A --fix that cannot write does not send the operator back to --fix: each
// FAIL names why the rewrite failed instead.
func TestCheckClaudePermissionRules_AFixThatCannotWriteSaysWhy(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs POSIX directory modes, which root ignores")
	}
	sys, dst := claudePermissionsBox(t, settingsWithMalformedRules)
	dir := filepath.Dir(dst)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	var buf bytes.Buffer
	checkClaudePermissionRules(sys, capture(&buf), true)
	out := buf.String()
	if strings.Count(out, "[FAIL]") != 2 || !strings.Contains(out, "--fix could not rewrite the file") {
		t.Errorf("want two FAILs naming the failed rewrite:\n%s", out)
	}
	if strings.Contains(out, "run: dotf doctor --fix") || strings.Contains(out, "removed") {
		t.Errorf("a failed --fix pointed back at --fix or claimed a removal:\n%s", out)
	}
}
