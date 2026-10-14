package herdr

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// statusOut is `herdr integration status` as 0.9.3 prints it, trimmed.
const statusOut = `pi: current (v9) (/h/.pi/agent/extensions/herdr-agent-state.ts)
claude: not installed (/h/.claude/hooks/herdr-agent-state.sh)
copilot: not installed (/h/.copilot/hooks/herdr-agent-state.sh)
opencode: outdated (v12, current v13) (/h (1)/.config/opencode/plugins/herdr-agent-state.js)
antigravity-cli: current (v3) (/h/.gemini/config/hooks/herdr-agent-state.sh)
letta (experimental): not installed (/h/.letta/hooks/herdr-agent-session.sh)
`

func TestParseStatus_ReadsEveryTargetAndItsState(t *testing.T) {
	got := ParseStatus([]byte(statusOut))
	want := map[string]string{
		"pi":              "current (v9)",
		"claude":          "not installed",
		"copilot":         "not installed",
		"opencode":        "outdated (v12, current v13)",
		"antigravity-cli": "current (v3)",
		"letta":           "not installed",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: got %q, want %q", k, got[k], v)
		}
	}
}

// Only a state that says current is current: an outdated integration, or one
// whose wording a later herdr changes, is reinstalled rather than trusted.
func TestCurrent_IsReadOnThePositive(t *testing.T) {
	for state, want := range map[string]bool{
		"current (v9)":                true,
		"not installed":               false,
		"outdated (v12, current v13)": false,
		"stale":                       false,
		"":                            false,
	} {
		if Current(state) != want {
			t.Errorf("Current(%q) = %v, want %v", state, !want, want)
		}
	}
}

// fakeHerdr answers status from states and records every install; an install
// of a target in broken fails.
type fakeHerdr struct {
	states   map[string]string
	broken   map[string]bool
	installs []string
}

func (f *fakeHerdr) run(name string, args ...string) ([]byte, error) {
	if name != "herdr" || len(args) < 2 || args[0] != "integration" {
		return nil, errors.New("unexpected command: " + name + " " + strings.Join(args, " "))
	}
	switch args[1] {
	case "status":
		var b strings.Builder
		for t, s := range f.states {
			b.WriteString(t + ": " + s + " (/h/" + t + ")\n")
		}
		return []byte(b.String()), nil
	case "install":
		t := args[2]
		f.installs = append(f.installs, t)
		if f.broken[t] {
			return []byte("error: cannot write"), errors.New("exit status 1")
		}
		f.states[t] = "current (v1)"
		return nil, nil
	}
	return nil, errors.New("unexpected subcommand " + args[1])
}

func installed(names ...string) func(string) bool {
	return func(n string) bool { return slices.Contains(names, n) }
}

// Every installed harness herdr has an integration for ends current; agy is
// herdr's antigravity-cli; a harness that is not installed is never touched,
// because herdr would create its config dir (the #1312 class); and a second
// run installs nothing.
func TestApply_InstallsForInstalledHarnessesOnly(t *testing.T) {
	f := &fakeHerdr{states: map[string]string{
		"claude": "not installed", "copilot": "not installed", "pi": "current (v9)",
		"opencode": "outdated (v12, current v13)", "antigravity-cli": "not installed",
	}}
	rep, err := Apply(f.run, []string{"agy", "claude", "copilot", "opencode", "pi"}, installed("agy", "claude", "opencode", "pi"), false)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(f.installs)
	if want := []string{"antigravity-cli", "claude", "opencode"}; !slices.Equal(f.installs, want) {
		t.Errorf("installed %v, want %v", f.installs, want)
	}
	if !slices.Equal(rep.Current, []string{"pi"}) || !slices.Equal(rep.Absent, []string{"copilot"}) {
		t.Errorf("report %+v", rep)
	}
	f.installs = nil
	again, err := Apply(f.run, []string{"agy", "claude", "copilot", "opencode", "pi"}, installed("agy", "claude", "opencode", "pi"), false)
	if err != nil || len(f.installs) != 0 || len(again.Installed) != 0 {
		t.Errorf("second run installed %v (%+v, %v); want nothing", f.installs, again, err)
	}
}

// A plan names what an apply would install and runs no install.
func TestApply_PlanInstallsNothing(t *testing.T) {
	f := &fakeHerdr{states: map[string]string{"claude": "not installed"}}
	rep, err := Apply(f.run, []string{"claude"}, installed("claude"), true)
	if err != nil || len(f.installs) != 0 || !slices.Equal(rep.Installed, []string{"claude"}) {
		t.Errorf("plan: %+v, %v, installs %v", rep, err, f.installs)
	}
}

// A harness herdr has no integration for is reported, not an error: the
// harness set is dotfiles', the integration set is herdr's.
func TestApply_AHarnessHerdrDoesNotKnowIsReported(t *testing.T) {
	f := &fakeHerdr{states: map[string]string{"claude": "current (v10)"}}
	rep, err := Apply(f.run, []string{"claude", "newagent"}, installed("claude", "newagent"), false)
	if err != nil || !slices.Equal(rep.Unknown, []string{"newagent"}) {
		t.Errorf("got %+v, %v; want newagent unknown", rep, err)
	}
}

// One failed install does not stop the others, and the error names it with
// herdr's own words.
func TestApply_AFailedInstallIsNamedAndTheRestProceed(t *testing.T) {
	f := &fakeHerdr{
		states: map[string]string{"claude": "not installed", "pi": "not installed"},
		broken: map[string]bool{"claude": true},
	}
	_, err := Apply(f.run, []string{"claude", "pi"}, installed("claude", "pi"), false)
	if err == nil || !strings.Contains(err.Error(), "claude") || !strings.Contains(err.Error(), "cannot write") {
		t.Errorf("err = %v; want the claude install named with herdr's message", err)
	}
	if f.states["pi"] != "current (v1)" {
		t.Errorf("pi was not installed after claude failed")
	}
}

func TestApply_AFailedStatusIsAnError(t *testing.T) {
	run := func(string, ...string) ([]byte, error) { return []byte("boom"), errors.New("exit status 2") }
	if _, err := Apply(run, []string{"claude"}, installed("claude"), true); err == nil {
		t.Error("a status that fails must not read as nothing to do")
	}
}

// herdr lists every integration it ships, so a status that exits 0 and
// parses to none is a format this reader does not know, not "herdr has no
// integration for any harness": reading it that way would report success
// and install nothing.
func TestApply_AStatusWithNoReadableLineIsAnError(t *testing.T) {
	for name, out := range map[string]string{
		"empty":    "",
		"reworded": "pi -> current (v9)\nclaude -> outdated\n",
	} {
		run := func(string, ...string) ([]byte, error) { return []byte(out), nil }
		if _, err := Apply(run, []string{"claude"}, installed("claude"), true); err == nil || !strings.Contains(err.Error(), "no integration") {
			t.Errorf("%s status: want an error naming the unreadable output, got %v", name, err)
		}
	}
}
