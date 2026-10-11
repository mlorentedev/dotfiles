package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/identity"
)

// stubIdentity replaces what the identity epilogue reads and runs: facts is
// the machine, and answers, when set, is a terminal the person types into
// (nil means no terminal). It returns the commands the guide ran.
func stubIdentity(t *testing.T, facts identity.Facts, answers *string) *[]string {
	t.Helper()
	savedFacts, savedTerm, savedExec := identityFacts, identityTerminal, identityExec
	t.Cleanup(func() { identityFacts, identityTerminal, identityExec = savedFacts, savedTerm, savedExec })
	identityFacts = func() identity.Facts { return facts }
	identityTerminal = func() (*os.File, bool) {
		if answers == nil {
			return nil, false
		}
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.WriteString(*answers); err != nil {
			t.Fatal(err)
		}
		_ = w.Close()
		return r, true
	}
	ran := &[]string{}
	identityExec = func(*os.File) identity.Exec {
		return func(argv []string) error {
			*ran = append(*ran, strings.Join(argv, " "))
			return nil
		}
	}
	return ran
}

// Without a terminal, a converge that succeeded ends with what is left of the
// identity, runs none of it, and exits as it would have: CI and a scheduled
// run see the plan, never a prompt (#2013 D12).
func TestConverge_WithoutATerminalEndsWithTheIdentityPlan(t *testing.T) {
	repo, _ := convergeFixture(t)
	ran := stubIdentity(t, identity.Facts{AgeKey: true}, nil)

	stdout, _, err := execute(t, "converge", "--repo", repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"identity: not restored yet", "[ OK ] age key (runbook step 1)", "[TODO] Bitwarden login (runbook step 2): bw login --quiet"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output lacks %q:\n%s", want, stdout)
		}
	}
	if len(*ran) != 0 {
		t.Errorf("ran %q without a terminal", *ran)
	}
	// The report comes before the identity, so the converge summary is still
	// the run's last word about the machine.
	if strings.Index(stdout, "report: ") > strings.Index(stdout, "identity: ") {
		t.Errorf("the identity plan must follow the converge report:\n%s", stdout)
	}
}

// On a terminal the converge walks the chain: here the person skips the first
// missing step, so nothing runs and the guide says how to resume.
func TestConverge_OnATerminalWalksTheRestore(t *testing.T) {
	repo, _ := convergeFixture(t)
	answers := "s\n"
	ran := stubIdentity(t, identity.Facts{AgeKey: true}, &answers)

	stdout, _, err := execute(t, "converge", "--repo", repo)
	if err != nil {
		t.Fatalf("a skipped identity step must not fail the converge: %v", err)
	}
	if !strings.Contains(stdout, "Bitwarden login: skipped") || !strings.Contains(stdout, "`dotf identity restore` resumes it") {
		t.Errorf("want the guide's stop:\n%s", stdout)
	}
	if len(*ran) != 0 {
		t.Errorf("ran %q after a skip", *ran)
	}
}

// A plan never walks the restore, even on a terminal; --only is not a full
// run and says nothing about identity.
func TestConverge_PlanAndOnlyNeverWalkTheRestore(t *testing.T) {
	repo, _ := convergeFixture(t)
	answers := "\n\n\n\n\n"
	ran := stubIdentity(t, identity.Facts{}, &answers)

	stdout, _, err := execute(t, "converge", "--plan", "--repo", repo)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "identity: not restored yet") || strings.Contains(stdout, "Enter") {
		t.Errorf("a plan prints the identity plan and asks nothing:\n%s", stdout)
	}
	stdout, _, err = execute(t, "converge", "--only", "records-mirror", "--repo", repo)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout, "identity") {
		t.Errorf("--only printed the identity:\n%s", stdout)
	}
	if len(*ran) != 0 {
		t.Errorf("ran %q", *ran)
	}
}

// `dotf identity restore` is the same walk on its own: on a terminal with
// every step to do, Enter at each runs the chain's commands in order.
func TestIdentityRestore_RunsTheChainOnATerminal(t *testing.T) {
	facts := identity.Facts{AgeKey: true}
	answers := "\n"
	ran := stubIdentity(t, facts, &answers)

	stdout, _, err := execute(t, "identity", "restore")
	if err != nil {
		t.Fatal(err)
	}
	// The stub's facts never change, so the first step that runs is still
	// missing afterwards: the guide must say so and stop.
	if got := strings.Join(*ran, "|"); got != "bw login --quiet" {
		t.Errorf("ran %q; want the Bitwarden login alone", got)
	}
	if !strings.Contains(stdout, "Bitwarden login: still not done") {
		t.Errorf("want the stop after a step that did not take:\n%s", stdout)
	}

	stdout, _, err = execute(t, "identity", "restore", "--plan")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "[TODO] Bitwarden unlock (runbook step 2)") || strings.Contains(stdout, "Enter") {
		t.Errorf("--plan prints and asks nothing:\n%s", stdout)
	}
}
