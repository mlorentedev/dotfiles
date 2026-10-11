package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/gitconfig"
)

type gitExitErr int

func (gitExitErr) Error() string   { return "exit status 1" }
func (e gitExitErr) ExitCode() int { return int(e) }

// gitSys is a machine whose global git config holds the repo's old bare
// helper and no include, with git and gh on PATH; gh is logged in when
// loggedIn, and its setup-git writes ghPath as the helper.
func gitSys(t *testing.T, loggedIn bool) (*System, map[string][]string) {
	t.Helper()
	home := t.TempDir()
	gh := filepath.Join(home, "bin", "gh")
	writeFile(t, gh, "")
	writeFile(t, gitconfig.IncludeFile(home), "")
	cfg := map[string][]string{
		"credential.https://github.com.helper":      {"", "!gh auth git-credential"},
		"credential.https://gist.github.com.helper": {"", "!gh auth git-credential"},
	}
	s := newSys(map[string]string{"HOME": home}, []string{"git", "gh"}, nil)
	s.CommandStdoutDir = func(_, name string, args ...string) (string, error) {
		return gitFake(t, cfg, gh, loggedIn, name, args...)
	}
	// The login question is asked without the environment's tokens, so it
	// reaches gh through CommandOutputEnv; same answers here.
	s.CommandOutputEnv = func(_ []string, name string, args ...string) (string, error) {
		return gitFake(t, cfg, gh, loggedIn, name, args...)
	}
	return s, cfg
}

func gitFake(t *testing.T, cfg map[string][]string, gh string, loggedIn bool, name string, args ...string) (string, error) {
	t.Helper()
	cmd := name + " " + strings.Join(args, " ")
	switch {
	case strings.HasPrefix(cmd, "git config --global --get-all "):
		v, ok := cfg[args[3]]
		if !ok {
			return "", gitExitErr(1)
		}
		return strings.Join(v, "\n") + "\n", nil
	case strings.HasPrefix(cmd, "git config --global --add "):
		cfg[args[3]] = append(cfg[args[3]], args[4])
	case cmd == "gh auth status --hostname github.com":
		if !loggedIn {
			return "", gitExitErr(1)
		}
	case cmd == "gh auth setup-git":
		for _, h := range gitconfig.CredentialHosts {
			cfg["credential."+h+".helper"] = []string{"", "!" + gh + " auth git-credential"}
		}
	default:
		t.Fatalf("unexpected command: %s", cmd)
	}
	return "", nil
}

// A login that exists only as a token in doctor's environment is not one the
// helper can use from a GUI app or a scheduled task, so the helper is blocked
// (WARN), not repairable (FAIL). CI's doctor gate has GH_TOKEN and its setup
// step does not; asking with the token went red on test-windows (#2319).
func TestCheckGitConfig_AnEnvironmentTokenIsNotALogin(t *testing.T) {
	s, _ := gitSys(t, true)
	var asked []string
	s.CommandOutputEnv = func(env []string, name string, args ...string) (string, error) {
		for _, kv := range env {
			if strings.HasPrefix(kv, "GH_TOKEN=") || strings.HasPrefix(kv, "GITHUB_TOKEN=") {
				t.Errorf("gh was asked with %s in its environment", strings.SplitN(kv, "=", 2)[0])
			}
		}
		asked = append(asked, name+" "+strings.Join(args, " "))
		return "", gitExitErr(1) // no stored login
	}
	t.Setenv("GH_TOKEN", "x")
	var b bytes.Buffer
	rep := capture(&b)
	checkGitConfig(s, rep, false)
	if rep.Failures() != 1 || rep.Warnings() != 1 || !strings.Contains(b.String(), "gh auth login") {
		t.Errorf("want the include FAIL and a helper WARN naming the login\n%s", b.String())
	}
	if len(asked) != 1 || asked[0] != "gh auth status --hostname github.com" {
		t.Errorf("asked %v, want one stored-login question", asked)
	}
}

func TestCheckGitConfig_ABareHelperAndNoIncludeFailNamingTheFix(t *testing.T) {
	s, cfg := gitSys(t, true)
	var b bytes.Buffer
	rep := capture(&b)
	checkGitConfig(s, rep, false)
	out := b.String()
	if rep.Failures() != 2 || !strings.Contains(out, gitconfig.IncludePath) || !strings.Contains(out, "!gh auth git-credential") || !strings.Contains(out, "dotf doctor --fix") {
		t.Errorf("want two FAILs naming the include, the helper and the fix\n%s", out)
	}
	if len(cfg["include.path"]) != 0 {
		t.Error("without --fix nothing is written")
	}
}

func TestCheckGitConfig_FixConvergesAndASecondRunPasses(t *testing.T) {
	s, cfg := gitSys(t, true)
	var b bytes.Buffer
	rep := capture(&b)
	checkGitConfig(s, rep, true)
	if rep.Failures() != 0 || rep.Warnings() != 0 || !strings.Contains(b.String(), "gh auth setup-git") {
		t.Fatalf("want a clean repair\n%s", b.String())
	}
	if got := cfg["include.path"]; len(got) != 1 || got[0] != gitconfig.IncludePath {
		t.Errorf("include.path = %v", got)
	}
	b.Reset()
	rep = capture(&b)
	checkGitConfig(s, rep, true)
	if rep.Failures() != 0 || rep.Warnings() != 0 || strings.Contains(b.String(), "FIX") || len(cfg["include.path"]) != 1 {
		t.Errorf("second run must be a plain pass\n%s", b.String())
	}
}

// gh not logged in: the include is still repaired; the helper is a WARN that
// names the login, because doctor cannot log in for the user.
func TestCheckGitConfig_AHelperOnlyALoginCanFixWarns(t *testing.T) {
	s, cfg := gitSys(t, false)
	var b bytes.Buffer
	rep := capture(&b)
	checkGitConfig(s, rep, true)
	if rep.Failures() != 0 || rep.Warnings() != 1 || !strings.Contains(b.String(), "gh auth login") {
		t.Errorf("want one WARN naming the login\n%s", b.String())
	}
	if len(cfg["include.path"]) != 1 {
		t.Error("the include does not need gh and must be repaired")
	}
}

func TestCheckGitConfig_AnUndeployedIncludeTargetFailsNamingDeploy(t *testing.T) {
	s, _ := gitSys(t, true)
	if err := os.Remove(gitconfig.IncludeFile(s.home())); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	rep := capture(&b)
	checkGitConfig(s, rep, true)
	if rep.Failures() != 1 || !strings.Contains(b.String(), "run: dotf deploy") || strings.Contains(b.String(), "runs gh by absolute path") {
		t.Errorf("want one FAIL naming dotf deploy and no PASS\n%s", b.String())
	}
}

func TestCheckGitConfig_SkipsWithoutGit(t *testing.T) {
	s := newSys(map[string]string{"HOME": os.TempDir()}, nil, nil)
	var b bytes.Buffer
	rep := capture(&b)
	checkGitConfig(s, rep, false)
	if rep.Failures() != 0 || !strings.Contains(b.String(), "git not on PATH") {
		t.Errorf("want a skip\n%s", b.String())
	}
}
