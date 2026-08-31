package initrepo

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/shellsafe"
)

func TestParseOriginRepo(t *testing.T) {
	cases := []struct {
		url     string
		want    string
		wantErr bool
	}{
		{"git@github.com:mlorentedev/dotfiles.git", "mlorentedev/dotfiles", false},
		{"git@github.com:mlorentedev/dotfiles", "mlorentedev/dotfiles", false},
		{"https://github.com/mlorentedev/dotfiles.git", "mlorentedev/dotfiles", false},
		{"https://github.com/mlorentedev/dotfiles", "mlorentedev/dotfiles", false},
		{"ssh://git@github.com/mlorentedev/dotfiles.git", "mlorentedev/dotfiles", false},
		{"https://github.com/mlorentedev/dotfiles\n", "mlorentedev/dotfiles", false}, // trailing newline from git
		{"git@gitlab.com:owner/repo.git", "", true},                                  // not github
		{"https://github.com/owner", "", true},                                       // no repo segment
		{"", "", true},
		{"not a url", "", true},
	}
	for _, c := range cases {
		got, err := ParseOriginRepo(c.url)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseOriginRepo(%q) = %q, want error", c.url, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseOriginRepo(%q) errored: %v", c.url, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseOriginRepo(%q) = %q, want %q", c.url, got, c.want)
		}
	}
}

func TestApplyDeleteBranchOnMergeSkipsWithoutGh(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // a PATH with no gh
	res, err := ApplyDeleteBranchOnMerge("owner/repo", false)
	if err != nil {
		t.Fatalf("expected graceful skip, got error: %v", err)
	}
	if res.Action != "skipped" {
		t.Errorf("Action = %q, want %q", res.Action, "skipped")
	}
	if !strings.Contains(strings.ToLower(res.Reason), "gh") {
		t.Errorf("skip Reason should mention gh, got %q", res.Reason)
	}
}

func TestApplyDeleteBranchOnMergeDryRunDoesNotPatch(t *testing.T) {
	calls := stubGh(t, "false")
	res, err := ApplyDeleteBranchOnMerge("owner/repo", true)
	if err != nil {
		t.Fatalf("ApplyDeleteBranchOnMerge: %v", err)
	}
	if res.Action != "dry-run" {
		t.Errorf("Action = %q, want %q", res.Action, "dry-run")
	}
	if patched(t, calls) {
		t.Errorf("--dry-run must not PATCH; calls:\n%s", readFile(t, calls))
	}
}

func TestApplyDeleteBranchOnMergeEnables(t *testing.T) {
	calls := stubGh(t, "false")
	res, err := ApplyDeleteBranchOnMerge("owner/repo", false)
	if err != nil {
		t.Fatalf("ApplyDeleteBranchOnMerge: %v", err)
	}
	if res.Action != "enabled" {
		t.Errorf("Action = %q, want %q", res.Action, "enabled")
	}
	if !patched(t, calls) {
		t.Errorf("expected a PATCH call; calls:\n%s", readFile(t, calls))
	}
}

func TestApplyDeleteBranchOnMergeAlreadyEnabled(t *testing.T) {
	calls := stubGh(t, "true")
	res, err := ApplyDeleteBranchOnMerge("owner/repo", false)
	if err != nil {
		t.Fatalf("ApplyDeleteBranchOnMerge: %v", err)
	}
	if res.Action != "already-enabled" {
		t.Errorf("Action = %q, want %q", res.Action, "already-enabled")
	}
	if patched(t, calls) {
		t.Errorf("already-enabled must be a no-op (no PATCH); calls:\n%s", readFile(t, calls))
	}
}

func TestOriginRepoFromGitRemote(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir, "git@github.com:mlorentedev/dotfiles.git")
	got, err := OriginRepo(dir)
	if err != nil {
		t.Fatalf("OriginRepo: %v", err)
	}
	if got != "mlorentedev/dotfiles" {
		t.Errorf("OriginRepo = %q, want %q", got, "mlorentedev/dotfiles")
	}
}

// --- helpers ---

// stubGh installs a fake `gh` on PATH that logs every invocation to the returned
// calls file, answers `gh api /repos/...` GETs with state ("true"/"false"), and
// returns success JSON for a `-X PATCH`. POSIX-only.
func stubGh(t *testing.T, state string) (callsFile string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("gh stub uses a POSIX shell script")
	}
	dir := t.TempDir()
	callsFile = filepath.Join(dir, "calls.log")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> " + shq(callsFile) + "\n" +
		"case \"$*\" in\n" +
		"  *'-X PATCH'*) printf '{\"delete_branch_on_merge\":true}' ;;\n" +
		"  *) printf '%s' " + shq(state) + " ;;\n" +
		"esac\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	return callsFile
}

func patched(t *testing.T, callsFile string) bool {
	t.Helper()
	b, err := os.ReadFile(callsFile)
	if err != nil {
		return false
	}
	return strings.Contains(string(b), "-X PATCH")
}

func gitInit(t *testing.T, dir, originURL string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	run("remote", "add", "origin", originURL)
}

func shq(s string) string {
	return shellsafe.Bash(s)
}
