package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/converge"
)

// runLogFixture is convergeFixture whose harness deploy and setup script run a
// real child process, the way uv, npm, winget and the setup script do: the
// child inherits dotf's stdout and prints what it was given. Both seams carry
// it because no single step runs on every OS: the harness deploy is linux and
// darwin, the setup script linux and windows.
func runLogFixture(t *testing.T, childSays string) (repo, state string, console *[]*os.File) {
	t.Helper()
	repo, home := convergeFixture(t)
	state = filepath.Join(home, "state")
	t.Setenv("XDG_STATE_HOME", state)
	seen := []*os.File{}
	base := convergeOptions
	convergeOptions = func() converge.Options {
		o := base()
		child := func(converge.Env) error {
			seen = append(seen, consoleOut())
			c := exec.Command(helperArgv()[0], helperArgv()[1:]...) //nolint:gosec // the test binary
			c.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "FOO="+childSays, "EXIT_CODE=0")
			c.Stdout, c.Stderr = os.Stdout, os.Stderr
			return c.Run()
		}
		o.RunHarnessDeploy, o.RunSetup = child, child
		return o
	}
	return repo, filepath.Join(state, "dotfiles"), &seen
}

func runTranscripts(t *testing.T, state string) []string {
	t.Helper()
	logs, err := filepath.Glob(filepath.Join(state, "converge", "runs", "*.log"))
	if err != nil {
		t.Fatal(err)
	}
	return logs
}

// The transcript holds what a child printed, with its credentials masked, and
// the report the run ended with: the two things a failed install on a box
// nobody can reach needs to be diagnosed after the terminal is gone.
func TestConverge_ApplyKeepsARedactedTranscript(t *testing.T) {
	const envSecret = "env-held-secret-value-42"
	t.Setenv("GITHUB_TOKEN", envSecret)
	gh := "ghp_" + strings.Repeat("Q7w8", 9)
	repo, state, _ := runLogFixture(t, "child: pushing with "+gh+" and "+envSecret+"\n")

	stdout, _, err := execute(t, "converge", "--repo", repo)
	if err != nil {
		t.Fatal(err)
	}
	logs := runTranscripts(t, state)
	if len(logs) != 1 {
		t.Fatalf("want one transcript, got %v", logs)
	}
	raw, err := os.ReadFile(logs[0])
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	if !strings.Contains(got, "child: pushing with [REDACTED]") {
		t.Errorf("the child's output is missing or unmasked:\n%s", got)
	}
	if !strings.Contains(got, "[REDACTED:GITHUB_TOKEN]") {
		t.Errorf("a secret-named variable's value should be masked by name:\n%s", got)
	}
	for _, leak := range []string{gh, envSecret} {
		if strings.Contains(got, leak) {
			t.Errorf("the transcript holds %q", leak)
		}
	}
	if !strings.Contains(got, "converge on ") || !strings.Contains(got, " changed, ") {
		t.Errorf("the transcript should end with the run's report:\n%s", got)
	}
	if _, err := os.Stat(strings.TrimSuffix(logs[0], ".log") + ".json"); err != nil {
		t.Errorf("the run's report is not beside its transcript: %v", err)
	}
	if !strings.Contains(stdout, "log: "+logs[0]) {
		t.Errorf("the run should say where its transcript is:\n%s", stdout)
	}
}

// While the run is captured, a terminal check still sees the real console: a
// sudo prompt or the identity walk must not decide "no terminal" because dotf
// itself is now writing to a pipe. And the swap is undone when the run ends.
func TestConverge_TerminalChecksSeeTheRealConsole(t *testing.T) {
	before := os.Stdout
	repo, _, seen := runLogFixture(t, "hello\n")
	if _, _, err := execute(t, "converge", "--repo", repo); err != nil {
		t.Fatal(err)
	}
	if len(*seen) == 0 || (*seen)[0] != before {
		t.Errorf("during the run consoleOut() was %v, want the original stdout %v", *seen, before)
	}
	if os.Stdout != before {
		t.Errorf("stdout was not restored after the run")
	}
}

// A plan changes nothing and keeps nothing.
func TestConvergePlan_KeepsNoTranscript(t *testing.T) {
	repo, state, _ := runLogFixture(t, "hello\n")
	if _, _, err := execute(t, "converge", "--plan", "--repo", repo); err != nil {
		t.Fatal(err)
	}
	if logs := runTranscripts(t, state); len(logs) != 0 {
		t.Errorf("a plan wrote a transcript: %v", logs)
	}
}

// An --only run is a run: it gets a transcript, and leaves last.json, the
// latest full run, alone.
func TestConvergeOnly_KeepsATranscriptButNotLastJSON(t *testing.T) {
	repo, state, _ := runLogFixture(t, "hello\n")
	if _, _, err := execute(t, "converge", "--only", "records-mirror", "--repo", repo); err != nil {
		t.Fatal(err)
	}
	if logs := runTranscripts(t, state); len(logs) != 1 {
		t.Errorf("an --only run should keep one transcript, got %v", logs)
	}
	if _, err := os.Stat(filepath.Join(state, "converge", "last.json")); !os.IsNotExist(err) {
		t.Errorf("an --only run wrote last.json: %v", err)
	}
}

// Only names that end like a credential are masked by name. A name that merely
// contains SESSION or CREDENTIAL holds ordinary values ("wayland", a socket
// path) that would otherwise be shredded throughout the transcript.
func TestRunLogSecrets_MasksCredentialNamesOnly(t *testing.T) {
	t.Setenv("XDG_SESSION_TYPE", "wayland")
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/run/user/1000/bus")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/home/u/.aws/credentials")
	t.Setenv("BW_SESSION", "bw-session-value-123")
	t.Setenv("OPENROUTER_API_KEY", "or-key-value-123")
	t.Setenv("GH_TOKEN", "gh-token-value-123")
	t.Setenv("BITACORA_PAT", "bitacora-pat-value-123")
	got := runLogSecrets(t.TempDir())
	for _, name := range []string{"XDG_SESSION_TYPE", "DBUS_SESSION_BUS_ADDRESS", "AWS_SHARED_CREDENTIALS_FILE"} {
		if _, ok := got[name]; ok {
			t.Errorf("%s is not a credential and must not be masked", name)
		}
	}
	for _, name := range []string{"BW_SESSION", "OPENROUTER_API_KEY", "GH_TOKEN", "BITACORA_PAT"} {
		if _, ok := got[name]; !ok {
			t.Errorf("%s holds a credential and must be masked", name)
		}
	}
}

// A variable the secrets registry exposes is masked whatever its name.
func TestRunLogSecrets_MasksRegistryExposedVars(t *testing.T) {
	repo := t.TempDir()
	reg := "version: 1\nsecrets:\n  - id: VAULT_DSN\n    plane: app\n    backend: bw\n    bw:\n      item: vault-dsn\n      field: VAULT_DSN\n      folder: Dotfiles/apps\n    expose: { env: VAULT_DSN }\n    consumers: [\"cli:test\"]\n"
	if err := os.MkdirAll(filepath.Join(repo, "secrets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "secrets", "registry.yaml"), []byte(reg), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VAULT_DSN", "postgres://db/vault-value")
	if _, ok := runLogSecrets(repo)["VAULT_DSN"]; !ok {
		t.Errorf("a registry-exposed variable must be masked")
	}
}

// The shipped registry parses, so the names it exposes reach the mask: a parse
// failure there would leave every registry secret unmasked, silently.
func TestRunLogSecrets_ReadsTheShippedRegistry(t *testing.T) {
	// A name no credential-shaped rule matches: only the registry knows it.
	const name = "KUBELAB_RESTIC_PASSWORD_BEELINK"
	t.Setenv(name, "restic-password-value-123")
	if secretNameShape.MatchString(name) {
		t.Fatalf("%s matches the name rule; pick a name only the registry knows", name)
	}
	if _, ok := runLogSecrets(repoRootForTest(t))[name]; !ok {
		t.Errorf("%s, which secrets/registry.yaml exposes, was not collected", name)
	}
}

// A failed run's error reaches both reports, the run's own and last.json, and
// both are redacted like the transcript: a step can fail with a token in its
// message (a remote URL it echoes).
func TestConverge_FailedRunReportsAreRedacted(t *testing.T) {
	gh := "ghp_" + strings.Repeat("F4il", 9)
	repo, home := convergeFixture(t)
	state := filepath.Join(home, "state")
	t.Setenv("XDG_STATE_HOME", state)
	base := convergeOptions
	convergeOptions = func() converge.Options {
		o := base()
		fail := func(converge.Env) error {
			return errors.New("push to https://x:" + gh + "@github.com/o/r refused")
		}
		o.RunHarnessDeploy, o.RunSetup = fail, fail // one runs on every OS
		return o
	}
	if _, _, err := execute(t, "converge", "--repo", repo); err == nil {
		t.Fatal("the run should fail")
	}
	dir := filepath.Join(state, "dotfiles", "converge")
	reports, _ := filepath.Glob(filepath.Join(dir, "runs", "*.json"))
	reports = append(reports, filepath.Join(dir, "last.json"))
	for _, p := range reports {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), gh) || !strings.Contains(string(raw), "[REDACTED]") {
			t.Errorf("%s leaks the token or was not masked:\n%s", filepath.Base(p), raw)
		}
	}
	if len(reports) != 2 {
		t.Errorf("want the run's report and last.json, got %v", reports)
	}
}

// On a terminal the hints are set for the run and removed after it; a hint
// the person set is kept, and off a terminal nothing is touched.
func TestForceColorFor_SetsColorHintsOnlyOnATerminal(t *testing.T) {
	for _, k := range colorHints {
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
	t.Setenv("CLICOLOR_FORCE", "0")
	undo := forceColorFor(true)
	if os.Getenv("FORCE_COLOR") != "1" || os.Getenv("HOMEBREW_COLOR") != "1" {
		t.Errorf("unset hints should be set on a terminal")
	}
	if os.Getenv("CLICOLOR_FORCE") != "0" {
		t.Errorf("a hint the person set was overridden")
	}
	undo()
	for _, k := range []string{"FORCE_COLOR", "HOMEBREW_COLOR"} {
		if _, ok := os.LookupEnv(k); ok {
			t.Errorf("%s survived the run", k)
		}
	}
	forceColorFor(false)()
	if _, ok := os.LookupEnv("FORCE_COLOR"); ok {
		t.Errorf("off a terminal no hint should be set")
	}
}

// The identity walk asks whether a person is at a terminal while the run is
// captured, when os.Stdout is a pipe into the transcript. It must ask the
// console. The terminal test is a seam here, true only for the console and
// stdin, so the test sees which file the walk asked about; with no terminal
// at all every check would answer false and miss the regression.
func TestConverge_TheIdentityWalkAsksTheConsole(t *testing.T) {
	saved := isTerminal
	t.Cleanup(func() { isTerminal = saved })
	// The console is the stdout the run started with, which the capture keeps
	// in consoleStdout; execute() has already swapped os.Stdout for its own
	// buffer, so the test's own os.Stdout is not it.
	isTerminal = func(fd uintptr) bool {
		return (consoleStdout != nil && fd == consoleStdout.Fd()) || fd == os.Stdin.Fd()
	}
	walkTerminal := identityTerminal // the real one; the fixture stubs it out
	repo, _, _ := runLogFixture(t, "hello\n")
	var answers []bool
	base := convergeOptions
	convergeOptions = func() converge.Options {
		o := base()
		inner := o.RunSetup
		ask := func(e converge.Env) error {
			_, ok := walkTerminal()
			answers = append(answers, ok)
			return inner(e)
		}
		o.RunHarnessDeploy, o.RunSetup = ask, ask
		return o
	}
	if _, _, err := execute(t, "converge", "--repo", repo); err != nil {
		t.Fatal(err)
	}
	if len(answers) == 0 || !answers[0] {
		t.Errorf("during a captured run identityTerminal() said no terminal (%v); it asked the transcript's pipe, not the console", answers)
	}
}
