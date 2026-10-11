package converge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/gitconfig"
)

type gitExit int

func (gitExit) Error() string   { return "exit status 1" }
func (e gitExit) ExitCode() int { return int(e) }

// gitFake is a global git config with a bare helper and no include, and a gh
// that is logged in and writes its absolute helper on setup-git.
func gitFake(ghPath string) (map[string][]string, *int, func(string, ...string) ([]byte, error)) {
	cfg := map[string][]string{
		"credential.https://github.com.helper":      {"", "!gh auth git-credential"},
		"credential.https://gist.github.com.helper": {"", "!gh auth git-credential"},
	}
	setups := 0
	run := func(name string, args ...string) ([]byte, error) {
		cmd := name + " " + strings.Join(args, " ")
		switch {
		case strings.HasPrefix(cmd, "git config --global --get-all "):
			v, ok := cfg[args[3]]
			if !ok {
				return nil, gitExit(1)
			}
			return []byte(strings.Join(v, "\n") + "\n"), nil
		case strings.HasPrefix(cmd, "git config --global --add "):
			cfg[args[3]] = append(cfg[args[3]], args[4])
		case cmd == "gh auth setup-git":
			setups++
			for _, h := range []string{"https://github.com", "https://gist.github.com"} {
				cfg["credential."+h+".helper"] = []string{"", "!" + ghPath + " auth git-credential"}
			}
		}
		return nil, nil
	}
	return cfg, &setups, run
}

// deployedHome is a home where `dotf deploy` has written the include target.
func deployedHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	dst := gitconfig.IncludeFile(home)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	return home
}

func TestGitConfig_PlanChangesNothingAndApplyConverges(t *testing.T) {
	home := deployedHome(t)
	gh := filepath.Join(home, "gh")
	if err := os.WriteFile(gh, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, setups, run := gitFake(gh)
	r := gitConfig{run: run, has: func(string) bool { return true }}
	env := Env{Home: home}

	plan, err := r.Reconcile(env, true)
	if err != nil || plan.Changes != 2 || !strings.HasPrefix(plan.Detail, "to apply:") {
		t.Fatalf("plan: %+v %v", plan, err)
	}
	if len(cfg["include.path"]) != 0 || *setups != 0 {
		t.Fatal("a plan must write nothing")
	}
	if _, err := r.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	if len(cfg["include.path"]) != 1 || *setups != 1 {
		t.Errorf("apply: include %v, setup-git ran %d", cfg["include.path"], *setups)
	}
	if err := r.Probe(env); err != nil {
		t.Errorf("probe after apply: %v", err)
	}
	again, err := r.Reconcile(env, false)
	if err != nil || again.Changes != 0 || *setups != 1 {
		t.Errorf("second run must change nothing: %+v %v, setup-git ran %d", again, err, *setups)
	}
}

func TestGitConfig_SkippedWithoutGit(t *testing.T) {
	_, _, run := gitFake("/nowhere/gh")
	r := gitConfig{run: run, has: func(n string) bool { return n != "git" }}
	res, err := r.Reconcile(Env{Home: t.TempDir()}, false)
	if err != nil || res.Skip == "" {
		t.Errorf("want a skip naming git, got %+v %v", res, err)
	}
}

// gh absent: the include converges and the probe passes; the detail names
// what is left and why, instead of failing a run converge cannot fix.
func TestGitConfig_ABlockedHelperIsNamedNotFailed(t *testing.T) {
	cfg, setups, run := gitFake("/nowhere/gh")
	r := gitConfig{run: run, has: func(n string) bool { return n == "git" }}
	env := Env{Home: deployedHome(t)}
	res, err := r.Reconcile(env, false)
	if err != nil || res.Changes != 1 || !strings.Contains(res.Detail, "gh is not on PATH") {
		t.Fatalf("got %+v %v", res, err)
	}
	if *setups != 0 || len(cfg["include.path"]) != 1 {
		t.Errorf("include %v, setup-git ran %d", cfg["include.path"], *setups)
	}
	if err := r.Probe(env); err != nil {
		t.Errorf("a helper only gh can fix must not fail the probe: %v", err)
	}
}

// The include converges but names a file `dotf deploy` never wrote: git
// ignores it silently, so the post-condition fails and names the remedy.
func TestGitConfig_AnIncludeOfAnUndeployedFileFailsTheProbe(t *testing.T) {
	home := t.TempDir()
	gh := filepath.Join(home, "gh")
	if err := os.WriteFile(gh, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, _, run := gitFake(gh)
	r := gitConfig{run: run, has: func(string) bool { return true }}
	env := Env{Home: home}
	if _, err := r.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	if len(cfg["include.path"]) != 1 {
		t.Fatalf("the include must still be added: %v", cfg["include.path"])
	}
	if err := r.Probe(env); err == nil || !strings.Contains(err.Error(), "dotf deploy") {
		t.Errorf("want the probe to fail naming dotf deploy, got %v", err)
	}
}

// Nothing repairable: the include is there, gh is absent and the deployed
// file is missing. The run changes nothing, so its detail must not say
// "applied", and it names both remedies.
func TestGitConfig_ARunThatChangesNothingDoesNotSayApplied(t *testing.T) {
	cfg, _, run := gitFake("/nowhere/gh")
	cfg["include.path"] = []string{gitconfig.IncludePath}
	r := gitConfig{run: run, has: func(n string) bool { return n == "git" }}
	res, err := r.Reconcile(Env{Home: t.TempDir()}, false)
	if err != nil || res.Changes != 0 {
		t.Fatalf("got %+v %v", res, err)
	}
	if strings.Contains(res.Detail, "applied") || !strings.Contains(res.Detail, "gh is not on PATH") || !strings.Contains(res.Detail, "dotf deploy") {
		t.Errorf("detail must name what is left, not claim an action: %q", res.Detail)
	}
}

func TestSelect_KeepsRegistryOrderAndRefusesAnUnknownName(t *testing.T) {
	reg := Registry(Options{})
	got, err := Select(reg, []string{"git-config", "records-mirror"})
	if err != nil || len(got) != 2 || got[0].Name() != "records-mirror" || got[1].Name() != "git-config" {
		t.Fatalf("got %v %v", got, err)
	}
	if _, err := Select(reg, []string{"git-confg"}); err == nil || !strings.Contains(err.Error(), "git-confg") || !strings.Contains(err.Error(), "known: checkout, records-mirror") {
		t.Errorf("want an error naming the typo and the known names, got %v", err)
	}
}
