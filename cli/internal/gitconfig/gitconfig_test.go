package gitconfig

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type exitErr int

func (e exitErr) Error() string { return fmt.Sprintf("exit status %d", int(e)) }
func (e exitErr) ExitCode() int { return int(e) }

// fakeGit answers `git config --global` from a key -> values map, as git does:
// absent keys exit 1. `--add` appends. gh answers from ghOK and records
// setup-git calls, which write the absolute helper it would write.
type fakeGit struct {
	cfg      map[string][]string
	ghOK     bool
	ghPath   string
	setupRan int
}

func (f *fakeGit) run(name string, args ...string) ([]byte, error) {
	cmd := name + " " + strings.Join(args, " ")
	switch {
	case strings.HasPrefix(cmd, "git config --global --get-all "):
		vals, ok := f.cfg[args[3]]
		if !ok {
			return nil, exitErr(1)
		}
		return []byte(strings.Join(vals, "\n") + "\n"), nil
	case strings.HasPrefix(cmd, "git config --global --add "):
		f.cfg[args[3]] = append(f.cfg[args[3]], args[4])
		return nil, nil
	case cmd == "gh auth status --hostname github.com":
		if !f.ghOK {
			return nil, exitErr(1)
		}
		return nil, nil
	case cmd == "gh auth setup-git":
		f.setupRan++
		for _, h := range CredentialHosts {
			f.cfg["credential."+h+".helper"] = []string{"", "!" + f.ghPath + " auth git-credential"}
		}
		return nil, nil
	}
	return nil, errors.New("unexpected command: " + cmd)
}

func machine(f *fakeGit, ghOnPath bool) Machine {
	return Machine{
		Home:   "/home/u",
		Run:    f.run,
		OnPath: func(string) bool { return ghOnPath },
		Exists: func(p string) bool { return p == f.ghPath || p == IncludeFile("/home/u") },
	}
}

// The repo's old ~/.gitconfig: a bare helper and no include.
func bareMachine() *fakeGit {
	return &fakeGit{ghOK: true, ghPath: "/opt/homebrew/bin/gh", cfg: map[string][]string{
		"credential.https://github.com.helper":      {"", "!gh auth git-credential"},
		"credential.https://gist.github.com.helper": {"", "!gh auth git-credential"},
	}}
}

func TestInspect_ABareHelperAndNoIncludeAreBothReported(t *testing.T) {
	f := bareMachine()
	st, err := Inspect(machine(f, true))
	if err != nil {
		t.Fatal(err)
	}
	if !st.IncludeMissing || len(st.BadHelpers) != 2 || st.Blocked != "" || st.Repairable() != 2 {
		t.Fatalf("want include missing and two bad helpers, repairable: %+v", st)
	}
	if !strings.Contains(st.BadHelpers[0], "!gh auth git-credential") {
		t.Errorf("a bad helper must name what git would run: %q", st.BadHelpers[0])
	}
}

func TestApply_ConvergesAndASecondRunHasNothingToDo(t *testing.T) {
	f := bareMachine()
	m := machine(f, true)
	st, _ := Inspect(m)
	if err := Apply(m, st); err != nil {
		t.Fatal(err)
	}
	after, err := Inspect(m)
	if err != nil {
		t.Fatal(err)
	}
	if !after.Converged() {
		t.Fatalf("after apply: %+v", after)
	}
	if got := f.cfg["include.path"]; len(got) != 1 || got[0] != IncludePath {
		t.Errorf("include.path = %v", got)
	}
	if err := Apply(m, after); err != nil || f.setupRan != 1 || len(f.cfg["include.path"]) != 1 {
		t.Errorf("a converged machine must not be written again: setup-git ran %d time(s), include %v", f.setupRan, f.cfg["include.path"])
	}
}

// gh not logged in or not on PATH: the include is still repaired, the helper
// is left alone, and Blocked names the remedy.
func TestInspect_AHelperGHCannotRepairIsBlockedNotApplied(t *testing.T) {
	for name, tc := range map[string]struct {
		onPath, loggedIn bool
		want             string
	}{
		"gh absent":        {false, false, "not on PATH"},
		"gh not logged in": {true, false, "gh auth login"},
	} {
		t.Run(name, func(t *testing.T) {
			f := bareMachine()
			f.ghOK = tc.loggedIn
			m := machine(f, tc.onPath)
			st, _ := Inspect(m)
			if !strings.Contains(st.Blocked, tc.want) || st.Repairable() != 1 {
				t.Fatalf("want blocked naming %q, one repairable: %+v", tc.want, st)
			}
			if err := Apply(m, st); err != nil {
				t.Fatal(err)
			}
			if f.setupRan != 0 || len(f.cfg["include.path"]) != 1 {
				t.Errorf("setup-git ran %d time(s), include %v", f.setupRan, f.cfg["include.path"])
			}
		})
	}
}

// The include is only a pointer: with the deployed file absent, git reads
// none of the dotfiles' settings, so the state is not converged. Apply cannot
// write that file (`dotf deploy` does), so it is not repairable either.
func TestInspect_AnIncludeWhoseFileIsNotDeployedIsNotConverged(t *testing.T) {
	f := bareMachine()
	m := machine(f, true)
	m.Exists = func(p string) bool { return p == f.ghPath }
	f.cfg["include.path"] = []string{IncludePath}
	f.cfg["credential.https://github.com.helper"] = []string{"", "!" + f.ghPath + " auth git-credential"}
	f.cfg["credential.https://gist.github.com.helper"] = []string{"", "!" + f.ghPath + " auth git-credential"}
	st, err := Inspect(m)
	if err != nil {
		t.Fatal(err)
	}
	if !st.TargetMissing || st.Converged() || st.Repairable() != 0 {
		t.Fatalf("want the missing target reported, not converged, nothing repairable: %+v", st)
	}
}

func TestInspect_AnIncludeSpelledAbsoluteCounts(t *testing.T) {
	f := bareMachine()
	f.cfg["include.path"] = []string{filepath.Join("/home/u", ".config", "git", "dotfiles.gitconfig")}
	st, _ := Inspect(machine(f, true))
	if st.IncludeMissing {
		t.Error("an absolute include of the same file is the include")
	}
}

func TestIsAbsoluteGHHelper(t *testing.T) {
	exists := func(string) bool { return true }
	for v, want := range map[string]bool{
		"!/opt/homebrew/bin/gh auth git-credential":                 true,
		"!/usr/bin/gh auth git-credential":                          true,
		`!'C:\Program Files\GitHub CLI\gh.exe' auth git-credential`: true,
		`!"C:/Program Files/GitHub CLI/gh.exe" auth git-credential`: true,
		"!gh auth git-credential":                                   false, // needs PATH
		"!/opt/homebrew/bin/gh auth token":                          false, // not the helper
		"!/usr/local/bin/hub auth git-credential":                   false, // not gh
		"/opt/homebrew/bin/gh auth git-credential":                  false, // not a shell command: git would run git-credential-/opt/...
		"osxkeychain": false,
		"!'/Applications/My Tools/gh' auth   git-credential":          true,
		"!/opt/homebrew/bin/gh auth git-credential --hostname x.test": false,
	} {
		if got := IsAbsoluteGHHelper(v, exists); got != want {
			t.Errorf("%q: got %v, want %v", v, got, want)
		}
	}
	if IsAbsoluteGHHelper("!/opt/homebrew/bin/gh auth git-credential", func(string) bool { return false }) {
		t.Error("a path to a gh that no longer exists is not a working helper")
	}
}

func TestInspect_AGitErrorOtherThanAbsentIsReturned(t *testing.T) {
	m := Machine{Home: "/h", Run: func(string, ...string) ([]byte, error) { return nil, exitErr(3) }, OnPath: func(string) bool { return true }, Exists: func(string) bool { return true }}
	if _, err := Inspect(m); err == nil || !strings.Contains(err.Error(), "include.path") {
		t.Errorf("want the failing key named, got %v", err)
	}
}

// An empty value appended last resets git's list to nothing. git prints it as
// a blank last line, which a trim of every trailing newline would swallow,
// leaving the stale helper before it to read as converged.
func TestInspect_AResetAppendedLastLeavesNoHelper(t *testing.T) {
	f := bareMachine()
	for _, h := range CredentialHosts {
		f.cfg["credential."+h+".helper"] = []string{"", "!" + f.ghPath + " auth git-credential", ""}
	}
	f.cfg["include.path"] = []string{IncludePath}
	st, err := Inspect(machine(f, true))
	if err != nil {
		t.Fatal(err)
	}
	if len(st.BadHelpers) != 2 || !strings.Contains(st.BadHelpers[0], "no helper") {
		t.Fatalf("a trailing reset leaves no helper: %+v", st)
	}
}

// The fake speaks git's dialect only if git does: one run against the real
// binary, confined to a temporary global config (GIT_CONFIG_GLOBAL) so the
// developer's ~/.gitconfig is never touched.
func TestAgainstRealGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	home := t.TempDir()
	global := filepath.Join(home, "gitconfig")
	if err := os.WriteFile(global, []byte("[credential \"https://github.com\"]\n\thelper =\n\thelper = !gh auth git-credential\n[credential \"https://gist.github.com\"]\n\thelper = !/abs/gh auth git-credential\n\thelper =\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	m := Machine{
		Home: home,
		Run: func(name string, args ...string) ([]byte, error) {
			if name == "gh" {
				return nil, nil // logged in; setup-git is not exercised here
			}
			return exec.Command(name, args...).Output() //nolint:gosec // test-controlled git
		},
		OnPath: func(string) bool { return true },
		Exists: func(string) bool { return true },
	}
	st, err := Inspect(m)
	if err != nil {
		t.Fatal(err)
	}
	if !st.IncludeMissing || len(st.BadHelpers) != 2 || !strings.Contains(st.BadHelpers[0], "!gh auth git-credential") || !strings.HasSuffix(st.BadHelpers[1], "no helper") {
		t.Fatalf("real git read differently: %+v", st)
	}
	st.BadHelpers = nil // only the include is exercised against real git
	if err := Apply(m, st); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("git", "config", "--global", "--get-all", "include.path").Output()
	if err != nil || strings.TrimSpace(string(out)) != IncludePath {
		t.Errorf("include.path after apply = %q (%v)", out, err)
	}
}

// gh is asked about its stored login, not the token in this environment: a
// GH_TOKEN is not a login a GUI app or a scheduled task inherits (#2319).
func TestInspect_AnEnvironmentTokenIsNotALogin(t *testing.T) {
	f := bareMachine() // f.ghOK: Run's gh, which sees the token, is logged in
	m := machine(f, true)
	m.StoredAuth = func(name string, args ...string) ([]byte, error) { return nil, exitErr(1) }
	st, err := Inspect(m)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(st.Blocked, "gh auth login") || st.Repairable() != 1 {
		t.Errorf("want the helper blocked on the login and only the include repairable: %+v", st)
	}
}

func TestWithoutEnvTokens(t *testing.T) {
	got := WithoutEnvTokens([]string{"PATH=/bin", "GH_TOKEN=a", "GITHUB_TOKEN=b", "GH_ENTERPRISE_TOKEN=c", "GITHUB_ENTERPRISE_TOKEN=d", "gh_token=e", "Github_Token=f", "GH_HOST=x", "HOME=/h"})
	if want := []string{"PATH=/bin", "GH_HOST=x", "HOME=/h"}; strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("got %v, want %v", got, want)
	}
}
