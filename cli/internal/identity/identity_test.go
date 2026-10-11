package identity

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

var testPaths = Paths{AgeKey: "/h/.config/age/key.txt", Vault: "/h/knowledge", VaultURL: "https://example.test/knowledge.git", Dotf: "dotf"}

// machine is a fake whose facts change as the guide runs commands: each
// command that restores something flips its fact, as the real one would.
type machine struct {
	f    Facts
	ran  []string
	fail string // a command that fails
	// noEffect is a command that exits 0 and restores nothing.
	noEffect string
}

func (m *machine) probe() Facts { return m.f }

func (m *machine) run(argv []string) error {
	cmd := strings.Join(argv, " ")
	m.ran = append(m.ran, cmd)
	if cmd == m.fail {
		return errors.New("exit status 1")
	}
	if cmd == m.noEffect {
		return nil
	}
	switch {
	case cmd == "bw login --quiet":
		m.f.BWLoggedIn = true
	case cmd == "dotf secrets unlock":
		m.f.BWUnlocked = true
	case cmd == "gh auth login":
		m.f.GHLoggedIn = true
	case strings.HasPrefix(cmd, "git clone "):
		m.f.Vault = true
	}
	return nil
}

func guide(m *machine, answers string) string {
	var out bytes.Buffer
	Guide(Terminal{In: strings.NewReader(answers), Out: &out}, Steps(testPaths), m.probe, m.run)
	return out.String()
}

// From zero, Enter at every prompt: the age key is the person's (the guide
// waits, re-reads, and goes on once it exists), every other step runs its
// commands in the runbook's order, and the chain ends restored.
func TestGuide_FromZeroRunsTheChainInOrder(t *testing.T) {
	m := &machine{}
	// The person restores the key while the guide waits on the first Enter.
	answers := &keyOnEnter{m: m, rest: "\n\n\n\n\n"}
	var out bytes.Buffer
	Guide(Terminal{In: answers, Out: &out}, Steps(testPaths), m.probe, m.run)

	want := []string{
		"bw login --quiet",
		"dotf secrets unlock", "dotf secrets verify",
		"gh auth login", "dotf converge --only git-config",
		"git clone https://example.test/knowledge.git /h/knowledge",
	}
	if strings.Join(m.ran, "|") != strings.Join(want, "|") {
		t.Errorf("ran %q\nwant %q", m.ran, want)
	}
	if !strings.Contains(out.String(), "identity: restored. Next: dotf converge, then dotf doctor") {
		t.Errorf("the chain did not end restored:\n%s", out.String())
	}
	if !strings.Contains(out.String(), testPaths.AgeKey) {
		t.Errorf("the age step must say where the key goes:\n%s", out.String())
	}
}

// keyOnEnter is the person restoring the age key by hand: their first Enter
// comes after the key file exists.
type keyOnEnter struct {
	m    *machine
	rest string
	done bool
}

func (k *keyOnEnter) Read(p []byte) (int, error) {
	if !k.done {
		k.done = true
		k.m.f.AgeKey = true
		return copy(p, "\n"), nil
	}
	if k.rest == "" {
		return 0, io.EOF
	}
	n := copy(p, k.rest)
	k.rest = k.rest[n:]
	return n, nil
}

// A restored machine asks nothing and runs nothing.
func TestGuide_RestoredAsksNothing(t *testing.T) {
	m := &machine{f: Facts{AgeKey: true, BWLoggedIn: true, BWUnlocked: true, GHLoggedIn: true, Vault: true}}
	out := guide(m, "")
	if len(m.ran) != 0 || strings.Contains(out, "Enter") {
		t.Errorf("ran %q, output:\n%s", m.ran, out)
	}
	if !strings.Contains(out, "identity: restored (") {
		t.Errorf("want the restored line:\n%s", out)
	}
}

// Done steps are passed without a prompt; the first missing one is where the
// guide starts asking.
func TestGuide_StartsAtTheFirstMissingStep(t *testing.T) {
	m := &machine{f: Facts{AgeKey: true, BWLoggedIn: true, BWUnlocked: true}}
	out := guide(m, "\n\n")
	if got := strings.Join(m.ran, "|"); got != "gh auth login|dotf converge --only git-config|git clone https://example.test/knowledge.git /h/knowledge" {
		t.Errorf("ran %q", got)
	}
	if strings.Count(out, "Enter to run it") != 2 {
		t.Errorf("want a prompt for each missing step only:\n%s", out)
	}
}

// A skip, a failed command, a command that restores nothing, and the end of
// input each stop the chain: no later step runs, and the guide says how to
// resume.
func TestGuide_AStepNotDoneStopsTheChain(t *testing.T) {
	cases := []struct {
		name    string
		m       *machine
		answers string
		why     string
	}{
		{"skipped", &machine{f: Facts{AgeKey: true}}, "s\n", "skipped"},
		{"failed", &machine{f: Facts{AgeKey: true}, fail: "bw login --quiet"}, "\n", "`bw login --quiet` failed"},
		{"no effect", &machine{f: Facts{AgeKey: true}, noEffect: "bw login --quiet"}, "\n", "still not done"},
		{"end of input", &machine{f: Facts{AgeKey: true}}, "", "skipped"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := guide(tc.m, tc.answers)
			for _, cmd := range tc.m.ran {
				if cmd != "bw login --quiet" {
					t.Errorf("ran %q after Bitwarden login was not done", cmd)
				}
			}
			if !strings.Contains(out, "Bitwarden login: "+tc.why) || !strings.Contains(out, "`dotf identity restore` resumes it") {
				t.Errorf("want the stop named %q and the resume command:\n%s", tc.why, out)
			}
			// Stopped means stopped: no later step is offered, let alone run.
			if strings.Contains(out, "Bitwarden unlock") || strings.Count(out, "Enter") != 1 {
				t.Errorf("a later step was offered after the stop:\n%s", out)
			}
		})
	}
}

// The plan runs nothing, marks what is done, and names each missing step's
// command, so a run without a terminal still tells the person what is left.
func TestPlan_NamesWhatIsLeftAndRunsNothing(t *testing.T) {
	var out bytes.Buffer
	Plan(&out, Steps(testPaths), Facts{AgeKey: true, BWLoggedIn: true})
	got := out.String()
	for _, want := range []string{
		"1. [ OK ] age key",
		"2. [ OK ] Bitwarden login",
		"3. [TODO] Bitwarden unlock: dotf secrets unlock, then dotf secrets verify",
		"4. [TODO] GitHub login: gh auth login, then dotf converge --only git-config",
		"5. [TODO] knowledge vault: git clone https://example.test/knowledge.git /h/knowledge",
		"`dotf identity restore`",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("plan lacks %q:\n%s", want, got)
		}
	}
}

// The doctor's SKIP texts and the guide are one chain: each Restore text names
// the runbook step the guide's order gives it.
func TestRestoreTextsNameTheRunbookStepsInOrder(t *testing.T) {
	for i, r := range []string{RestoreAge, RestoreBW, RestoreGH, RestoreVault} {
		if want := "guide-new-machine.md, step " + string(rune('1'+i)) + ")"; !strings.HasSuffix(r, want) {
			t.Errorf("restore text %d = %q; want it to end %q", i, r, want)
		}
	}
}

// An unresolved VAULT_PATH turns the clone into a manual step: the guide never
// clones into whatever directory it was started from.
func TestSteps_AnUnresolvedVaultPathIsNotCloned(t *testing.T) {
	p := testPaths
	p.Vault = ""
	steps := Steps(p)
	v := steps[len(steps)-1]
	if v.Run != nil || !strings.Contains(v.Manual, "VAULT_PATH resolves to nothing") {
		t.Errorf("vault step = %+v; want a manual step naming VAULT_PATH", v)
	}
}
