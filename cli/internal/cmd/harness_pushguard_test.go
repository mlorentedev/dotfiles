package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// pushGuardRepo builds a real clone on branch feat/x whose remote "origin" has
// default branch main (origin/HEAD set), and returns the clone's directory and
// the sha of its feat/x tip. Real git answers the guard's two questions, so the
// test sees what the hook sees.
func pushGuardRepo(t *testing.T) (dir, sha string) {
	t.Helper()
	root := t.TempDir()
	empty := filepath.Join(root, "gitconfig")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", empty)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	run := func(wd string, args ...string) string {
		t.Helper()
		c := exec.Command("git", append([]string{"-c", "commit.gpgsign=false", "-c", "core.hooksPath=" + os.DevNull,
			"-c", "user.name=t", "-c", "user.email=t@example.invalid", "-c", "init.defaultBranch=main"}, args...)...)
		c.Dir = wd
		out, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	remote := filepath.Join(root, "remote.git")
	run(root, "init", "--bare", remote)
	dir = filepath.Join(root, "clone")
	run(root, "init", dir)
	run(dir, "commit", "--allow-empty", "-m", "base")
	run(dir, "remote", "add", "origin", remote)
	run(dir, "push", "origin", "main")
	run(dir, "remote", "set-head", "origin", "main")
	run(dir, "checkout", "-b", "feat/x")
	run(dir, "commit", "--allow-empty", "-m", "work")
	return dir, run(dir, "rev-parse", "HEAD")
}

func pushGuardPayloadFor(tool, cwd, command string) string {
	b, _ := json.Marshal(map[string]any{
		"session_id": "s", "tool_name": tool, "cwd": cwd,
		"tool_input": map[string]any{"command": command},
	})
	return string(b)
}

// runPushGuard drives the real command in process and returns the exit status
// the harness would see plus what it would show the model.
func runPushGuardCmd(t *testing.T, payload string) (int, string) {
	t.Helper()
	c := newHarnessPushGuardCmd()
	var errb bytes.Buffer
	c.SetIn(strings.NewReader(payload))
	c.SetOut(io.Discard)
	c.SetErr(&errb)
	c.SetArgs(nil)
	err := c.Execute()
	// stderr only: the binary silences the returned error, so the reason must
	// already be on the stream (a refusal with an empty stderr shows the model nothing).
	return ExitCode(err), errb.String()
}

func TestPushGuard_AgainstARealRepository(t *testing.T) {
	dir, sha := pushGuardRepo(t)
	lease := func(b string) string { return "--force-with-lease=" + b + ":" + sha }
	cases := []struct {
		name string
		tool string
		cmd  string
		want int
		why  string
	}{
		{"lease on the checked-out branch", "Bash", "git push origin " + lease("feat/x") + " feat/x", 0, ""},
		{"lease through git -C", "Bash", "git -C " + dir + " push origin " + lease("feat/x") + " HEAD:refs/heads/feat/x", 0, ""},
		{"lease after a cd", "Bash", "cd " + dir + " && git push origin " + lease("feat/x") + " feat/x", 0, ""},
		{"powershell tool", "PowerShell", "git push origin " + lease("feat/x") + " feat/x", 0, ""},
		{"plain push", "Bash", "git push origin feat/x", 0, ""},
		{"not a shell tool", "Edit", "git push --force origin main", 0, ""},
		{"force", "Bash", "git push --force origin feat/x", 2, "force"},
		{"short force", "Bash", "git -C " + dir + " push -f origin feat/x", 2, "force"},
		{"plus refspec", "Bash", "git push origin +main", 2, "+"},
		{"lease on main", "Bash", "git push origin " + lease("main") + " HEAD:refs/heads/main", 2, "default branch"},
		{"lease naming another branch", "Bash", "git push origin " + lease("other") + " other", 2, "checked-out"},
		{"lease without a sha", "Bash", "git push origin --force-with-lease=feat/x feat/x", 2, "sha"},
		{"landing mode", "Bash", "npm run land:push", 2, "land:push"},
		{"powershell force", "PowerShell", "git push -f origin x", 2, "force"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, msg := runPushGuardCmd(t, pushGuardPayloadFor(c.tool, dir, c.cmd))
			if code != c.want {
				t.Fatalf("exit %d, want %d (%s)", code, c.want, msg)
			}
			if c.want == 2 && !strings.Contains(strings.ToLower(msg), c.why) {
				t.Fatalf("message %q does not mention %q", msg, c.why)
			}
		})
	}
}

// With no origin/HEAD the default branch cannot be resolved, so main and master
// stay refused by name; with it pointing elsewhere, that branch is refused too.
func TestPushGuard_DefaultBranchResolution(t *testing.T) {
	dir, sha := pushGuardRepo(t)
	run := func(args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	lease := func(b string) string { return "--force-with-lease=" + b + ":" + sha }

	run("remote", "set-head", "origin", "-d")
	if code, msg := runPushGuardCmd(t, pushGuardPayloadFor("Bash", dir, "git push origin "+lease("main")+" main")); code != 2 {
		t.Errorf("main with origin/HEAD unset: exit %d, want 2 (%s)", code, msg)
	}

	// Make the resolved default something else: push develop, point origin/HEAD at it.
	run("push", "origin", "feat/x:develop")
	run("fetch", "origin")
	run("remote", "set-head", "origin", "develop")
	run("checkout", "-b", "develop", "feat/x")
	if code, msg := runPushGuardCmd(t, pushGuardPayloadFor("Bash", dir, "git push origin "+lease("develop")+" develop")); code != 2 || !strings.Contains(msg, "default branch") {
		t.Errorf("resolved default develop: exit %d, want 2 naming the default branch (%s)", code, msg)
	}
	// ...and feat/x is an ordinary branch again once develop is the default.
	run("checkout", "feat/x")
	if code, msg := runPushGuardCmd(t, pushGuardPayloadFor("Bash", dir, "git push origin "+lease("feat/x")+" feat/x")); code != 0 {
		t.Errorf("own branch with develop as default: exit %d, want 0 (%s)", code, msg)
	}
}

// The status a hook returns is the whole contract: 0 or 2, never 1. Claude reads
// any other status as a non-blocking error, which would be "allowed" in disguise.
func TestPushGuard_NeverExitsOne(t *testing.T) {
	dir := t.TempDir() // not a repository: git answers nothing, which must never turn into an exit 1
	payloads := []string{
		"", "   ", "not json", "[]", "null", `{"tool_name":"Bash"}`, `{"tool_name":"Bash","tool_input":{}}`,
		`{"tool_name":"Bash","tool_input":{"command":42}}`,
		`{"tool_name":123}`,
		pushGuardPayloadFor("Bash", "", "git push --force-with-lease=a:b c d"),
		pushGuardPayloadFor("Bash", "/does/not/exist", "git push origin --force-with-lease=feat/x:"+strings.Repeat("a", 40)+" feat/x"),
		pushGuardPayloadFor("Bash", dir, "git push --force \"origin"),
		pushGuardPayloadFor("Bash", dir, strings.Repeat("(", 500)+"git push -f"),
		pushGuardPayloadFor("Bash", dir, strings.Repeat("bash -c '", 10)+"git push -f"),
	}
	for _, p := range payloads {
		if code, msg := runPushGuardCmd(t, p); code != 0 && code != 2 {
			t.Errorf("payload %.60q: exit %d (%s), want 0 or 2", p, code, msg)
		}
	}
}

// An unresolvable directory refuses a lease push rather than allowing it.
func TestPushGuard_UnresolvableDirectoryRefusesTheLease(t *testing.T) {
	cmdline := "git push origin --force-with-lease=feat/x:" + strings.Repeat("a", 40) + " feat/x"
	code, msg := runPushGuardCmd(t, pushGuardPayloadFor("Bash", filepath.Join(t.TempDir(), "nowhere"), cmdline))
	if code != 2 || !strings.Contains(msg, "checked-out") {
		t.Fatalf("exit %d (%s), want 2 about the checked-out branch", code, msg)
	}
}

// Subagents send `cd /c/Users/... && git push` and a native git cannot resolve
// that spelling; the conversion is what keeps every lease push from being refused
// on Windows while Linux CI stays green.
func TestNativeDir(t *testing.T) {
	cases := []struct{ goos, in, want string }{
		{"windows", "/c/Users/u/repo", "C:/Users/u/repo"},
		{"windows", "/C/Users/u/repo", "C:/Users/u/repo"},
		{"windows", "/c", "C:/"},
		{"windows", "/cygdrive/d/work", "D:/work"},
		{"windows", `C:\Users\u\repo`, "C:/Users/u/repo"},
		{"windows", "C:/Users/u/repo", "C:/Users/u/repo"},
		{"windows", "/tmp/x", "/tmp/x"},
		{"windows", "relative/dir", "relative/dir"},
		{"linux", "/c/Users/u/repo", "/c/Users/u/repo"},
		{"darwin", "/c/x", "/c/x"},
	}
	for _, c := range cases {
		if got := nativeDir(c.goos, c.in); got != c.want {
			t.Errorf("nativeDir(%q, %q) = %q, want %q", c.goos, c.in, got, c.want)
		}
	}
}

func TestSafeRemoteName(t *testing.T) {
	for in, want := range map[string]bool{
		"origin": true, "up-stream": true, "a.b": true,
		"": false, "-o": false, "--upload-pack=x": false, "https://h/r.git": false, "a b": false, "a..b": false, "../x": false,
	} {
		if got := safeRemoteName(in); got != want {
			t.Errorf("safeRemoteName(%q) = %v, want %v", in, got, want)
		}
	}
}

// Every `dotf harness <x>` the manifest binds for Claude must be a command this
// binary has. An older binary lacking push-guard would run the hook as an error
// on every tool call; this catches the manifest naming a subcommand that is not
// registered at all.
func TestBoundHarnessHooksResolveToRealSubcommands(t *testing.T) {
	targets, err := harness.LoadBindTargets(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	root := newHarnessCmd()
	seen := 0
	for _, tg := range targets {
		for _, h := range tg.EmitHooks {
			f := strings.Fields(h.Command)
			if len(f) < 2 || f[0] != "harness" {
				continue
			}
			seen++
			found, _, err := root.Find(f[1:2])
			if err != nil || found == root || found.Name() != f[1] {
				t.Errorf("%s hook %q names `dotf harness %s`, which is not a registered subcommand", tg.Agent, h.ID, f[1])
			}
		}
	}
	if seen == 0 {
		t.Fatal("no harness hook was checked: the manifest moved")
	}
}

// A POSIX shell on Windows sends /c/... for a drive path; the guard must read the
// branch from that directory through the real git, not only through a unit.
func TestPushGuard_MsysDriveDirectoryAgainstARealRepository(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("drive-letter spellings exist only on Windows")
	}
	dir, sha := pushGuardRepo(t)
	if len(dir) < 3 || dir[1] != ':' {
		t.Skipf("repository path %q has no drive letter", dir)
	}
	msys := "/" + strings.ToLower(dir[:1]) + filepath.ToSlash(dir[2:])
	lease := "--force-with-lease=feat/x:" + sha
	for _, c := range []struct {
		name, cmd string
		want      int
	}{
		{"lease after cd", "cd " + msys + " && git push origin " + lease + " feat/x", 0},
		{"lease through -C", "git -C " + msys + " push origin " + lease + " feat/x", 0},
		{"force after cd", "cd " + msys + " && git push -f origin feat/x", 2},
	} {
		t.Run(c.name, func(t *testing.T) {
			code, msg := runPushGuardCmd(t, pushGuardPayloadFor("Bash", "/nowhere", c.cmd))
			if code != c.want {
				t.Fatalf("exit %d, want %d (%s)", code, c.want, msg)
			}
		})
	}
}
