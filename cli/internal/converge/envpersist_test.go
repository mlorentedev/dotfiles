package converge

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
)

// fakeLaunchctl is launchctl over one user's gui domain: the session
// environment, the loaded labels, and every call made. With background set the
// caller sits in tmux or herdr (#2507): getenv sees an empty domain of its own,
// and the session is reachable only through `print gui/501`.
type fakeLaunchctl struct {
	vars       map[string]string
	loaded     bool
	calls      []string
	failBoot   error
	background bool
}

func newFakeLaunchctl() *fakeLaunchctl { return &fakeLaunchctl{vars: map[string]string{}} }

func (f *fakeLaunchctl) run(args ...string) (string, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	if f.background && (args[0] == "getenv" || args[0] == "setenv" || args[0] == "unsetenv") {
		return "", nil // the caller's own Background domain, which nothing reads
	}
	switch args[0] {
	case "managername":
		if f.background {
			return "Background\n", nil
		}
		return "Aqua\n", nil
	case "getenv":
		if v, ok := f.vars[args[1]]; ok {
			return v + "\n", nil
		}
		return "", nil
	case "setenv":
		f.vars[args[1]] = args[2]
	case "unsetenv":
		delete(f.vars, args[1])
	case "print":
		if args[1] == "gui/501" {
			var b strings.Builder
			b.WriteString("gui/501 = {\n\tenvironment = {\n")
			for k, v := range f.vars {
				b.WriteString("\t\t" + k + " => " + v + "\n")
			}
			b.WriteString("\t}\n}\n")
			return b.String(), nil
		}
		if !f.loaded {
			return "", errors.New("exit status 113")
		}
	case "bootstrap":
		if f.failBoot != nil {
			return "", f.failBoot
		}
		f.loaded = true
	case "bootout":
		f.loaded = false
	}
	return "", nil
}

func (f *fakeLaunchctl) called(verb string) int {
	n := 0
	for _, c := range f.calls {
		if strings.HasPrefix(c, verb+" ") {
			n++
		}
	}
	return n
}

func envPersistEnv(t *testing.T) Env {
	t.Helper()
	repo := t.TempDir()
	contract := `{"env_vars":[
		{"name":"VAULT_PATH","default":{"linux":"$HOME/Projects/knowledge"}},
		{"name":"DOTFILES_DIR","default":{"linux":"$HOME/.dotfiles"}}]}`
	if err := os.WriteFile(filepath.Join(repo, "env-contract.json"), []byte(contract), 0o600); err != nil {
		t.Fatal(err)
	}
	return Env{RepoRoot: repo, Home: t.TempDir(), GOOS: "darwin"}
}

// The criterion: a plan writes nothing; an apply sets every variable in the
// launchd session, writes and loads the login agent, and passes its probe; a
// second run changes nothing.
func TestEnvPersist_ApplyThenSecondRunIsZero(t *testing.T) {
	e := envPersistEnv(t)
	lc := newFakeLaunchctl()
	r := envPersist{launchctl: lc.run, uid: 501}

	plan, err := r.Reconcile(e, true)
	if err != nil {
		t.Fatal(err)
	}
	// 2 variables + the marker + write and load the agent.
	if plan.Changes != 5 {
		t.Errorf("plan changes = %d, want 5 (%s)", plan.Changes, plan.Detail)
	}
	if lc.called("setenv")+lc.called("bootstrap") > 0 {
		t.Fatalf("a plan must write nothing, calls: %v", lc.calls)
	}
	if _, err := os.Stat(env.LaunchAgentPath(e.Home)); err == nil {
		t.Fatal("a plan must not write the plist")
	}

	if _, err := r.Reconcile(e, false); err != nil {
		t.Fatal(err)
	}
	if err := r.Probe(e); err != nil {
		t.Fatalf("probe after apply: %v", err)
	}
	if got := lc.vars["VAULT_PATH"]; got != e.Home+"/Projects/knowledge" {
		t.Errorf("VAULT_PATH in the session = %q", got)
	}
	if !strings.Contains(strings.Join(lc.calls, "\n"), "bootstrap gui/501 "+env.LaunchAgentPath(e.Home)) {
		t.Errorf("the agent must be bootstrapped into the user's gui domain, calls: %v", lc.calls)
	}

	again, err := r.Reconcile(e, false)
	if err != nil {
		t.Fatal(err)
	}
	if again.Changes != 0 {
		t.Errorf("second run changes = %d (%s)", again.Changes, again.Detail)
	}
	if lc.called("bootstrap") != 1 {
		t.Errorf("a loaded, current agent must not be bootstrapped again (bootstrap refuses a loaded label), calls: %v", lc.calls)
	}
}

// A changed plist on a loaded agent is booted out before it is bootstrapped:
// launchd keeps the definition it read at load time.
func TestEnvPersist_ChangedPlistReloadsTheAgent(t *testing.T) {
	e := envPersistEnv(t)
	lc := newFakeLaunchctl()
	r := envPersist{launchctl: lc.run, uid: 501}
	if _, err := r.Reconcile(e, false); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env.LaunchAgentPath(e.Home), []byte("<plist/>"), 0o644); err != nil { //nolint:gosec // test fixture
		t.Fatal(err)
	}

	if _, err := r.Reconcile(e, false); err != nil {
		t.Fatal(err)
	}
	if lc.called("bootout") != 1 || lc.called("bootstrap") != 2 {
		t.Errorf("want one bootout then a second bootstrap, calls: %v", lc.calls)
	}
	if err := r.Probe(e); err != nil {
		t.Fatalf("probe: %v", err)
	}
}

// After a logout the session is empty again while the agent stays loaded: the
// run re-sets the variables and leaves the agent alone.
func TestEnvPersist_ClearedSessionIsReapplied(t *testing.T) {
	e := envPersistEnv(t)
	lc := newFakeLaunchctl()
	r := envPersist{launchctl: lc.run, uid: 501}
	if _, err := r.Reconcile(e, false); err != nil {
		t.Fatal(err)
	}
	lc.vars = map[string]string{}

	res, err := r.Reconcile(e, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changes != 3 || lc.called("bootstrap") != 1 {
		t.Errorf("want 2 variables + the marker and no reload, got %d (%s), calls: %v", res.Changes, res.Detail, lc.calls)
	}
}

// A bootstrap that fails (no GUI session, for one) fails the run; the probe
// would otherwise be the only thing to notice.
func TestEnvPersist_FailedBootstrapFails(t *testing.T) {
	e := envPersistEnv(t)
	lc := newFakeLaunchctl()
	lc.failBoot = errors.New("Bootstrap failed: 125: Domain does not support specified action")
	r := envPersist{launchctl: lc.run, uid: 501}

	if _, err := r.Reconcile(e, false); err == nil || !strings.Contains(err.Error(), "bootstrap gui/501") {
		t.Fatalf("want the bootstrap failure, got %v", err)
	}
	if err := r.Probe(e); err == nil {
		t.Fatal("the probe must fail while the agent is not loaded")
	}
}

func TestEnvPersist_DarwinOnlyAndSkippedWithoutARunner(t *testing.T) {
	r := envPersist{}
	if got := r.Platforms(); len(got) != 1 || got[0] != "darwin" {
		t.Errorf("Platforms = %v", got)
	}
	res, err := r.Reconcile(envPersistEnv(t), false)
	if err != nil || res.Skip == "" {
		t.Fatalf("no runner must skip, got %+v, %v", res, err)
	}
}

// From tmux or herdr (#2507) the run reads the login session through `print`:
// a session already in line with the contract is 0 changes, where getenv on
// the Background domain used to report every variable as missing.
func TestEnvPersist_FromBackgroundReadsTheLoginSession(t *testing.T) {
	e := envPersistEnv(t)
	lc := newFakeLaunchctl()
	r := envPersist{launchctl: lc.run, uid: 501}
	if _, err := r.Reconcile(e, false); err != nil {
		t.Fatal(err)
	}
	lc.background = true

	res, err := r.Reconcile(e, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changes != 0 || res.Skip != "" {
		t.Errorf("want 0 changes from Background over a current session, got %+v", res)
	}
	if err := r.Probe(e); err != nil {
		t.Errorf("probe from Background: %v", err)
	}
}

// Real drift seen from Background is reported and left alone: setenv there
// would land where no Dock app reads it, and report a success that is not one.
func TestEnvPersist_FromBackgroundSkipsADriftItCannotFix(t *testing.T) {
	e := envPersistEnv(t)
	lc := newFakeLaunchctl()
	lc.background = true
	r := envPersist{launchctl: lc.run, uid: 501}

	res, err := r.Reconcile(e, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Skip == "" || res.Changes != 0 {
		t.Fatalf("want a skip, got %+v", res)
	}
	for _, want := range []string{"VAULT_PATH", "login terminal"} {
		if !strings.Contains(res.Skip, want) {
			t.Errorf("skip %q must name %q", res.Skip, want)
		}
	}
	if n := lc.called("setenv") + lc.called("bootstrap") + lc.called("bootout"); n != 0 {
		t.Errorf("nothing may be written from Background, calls: %v", lc.calls)
	}
	if _, err := os.Stat(env.LaunchAgentPath(e.Home)); err == nil {
		t.Error("the plist must not be written from Background either")
	}
}
