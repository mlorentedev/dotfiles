package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedMachine writes an arbitrary machine.json into a fixture config dir, for
// the cases that need a shape declareIdentity does not produce.
func seedMachine(t *testing.T, body string) {
	t.Helper()
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	dir := filepath.Join(cfg, "dotfiles")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "machine.json"), []byte(body), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

// AC4's second half. ADR-032 §8: a machine whose identity cannot be established
// denies every non-local pool, so the unknown case degrades to "no cross-pool
// dispatch" rather than to "all pools allowed".
func TestAgentRun_AnUnidentifiedMachineIsRefused(t *testing.T) {
	root := repoRootForTest(t)

	tests := []struct {
		name string
		// seed is the machine.json body; empty means no file at all.
		seed string
	}{
		{name: "no machine.json at all"},
		{name: "only paths declared", seed: `{"paths": {"VAULT_PATH": "/v"}}`},
		{name: "a machine block with no id", seed: `{"machine": {}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.seed == "" {
				t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // exists, holds no machine.json
			} else {
				seedMachine(t, tc.seed)
			}

			stdout, _, err := captureRealStreams(t,
				"agent", "run", "--role", "r", "--task", "t", "--tier", "mid",
				"--backend", "dry-run", "--timeout", "1m", "--repo-root", root,
			)
			if err == nil {
				t.Fatal("an unidentified machine dispatched; the default must be denial, not permission")
			}
			if strings.TrimSpace(stdout) != "" {
				t.Errorf("stdout is not empty on a refusal: %q", stdout)
			}
			if got := ExitCode(err); got != 1 {
				t.Errorf("exit = %d, want 1", got)
			}

			// The remedy has to be IN the message. A fail-closed refusal whose
			// fix the operator must go and look up is one people route around.
			msg := err.Error()
			for _, want := range []string{"machine.json", `"machine"`, `"id"`, "deny"} {
				if !strings.Contains(msg, want) {
					t.Errorf("refusal does not name %q, so it is not actionable:\n%s", want, msg)
				}
			}
			if !strings.Contains(msg, "ADR-032") {
				t.Errorf("refusal cites no decision record, so it reads as an arbitrary rule:\n%s", msg)
			}
		})
	}
}

// A deny entry naming no declared pool is refused rather than ignored: `claud`
// in the list leaves `claude` allowed on a machine whose intent was to forbid
// it, and nothing would ever say so.
func TestAgentRun_ADenyTypoIsRefused(t *testing.T) {
	root := repoRootForTest(t)
	seedMachine(t, `{"machine": {"id": "corp"}, "pools": {"deny": ["claud"]}}`)

	stdout, _, err := captureRealStreams(t,
		"agent", "run", "--role", "r", "--task", "t", "--tier", "mid",
		"--backend", "dry-run", "--timeout", "1m", "--repo-root", root,
	)
	if err == nil {
		t.Fatal("a deny entry naming no declared pool was accepted")
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("stdout is not empty on a refusal: %q", stdout)
	}
	if !strings.Contains(err.Error(), "claud") {
		t.Errorf("refusal does not name the offending entry:\n%s", err.Error())
	}
}

// An identified machine that denies the chain's first pool routes around it.
func TestAgentRun_DenialSkipsThePoolAndRecordsIt(t *testing.T) {
	root := repoRootForTest(t)
	declareIdentity(t, "claude")

	stdout, _, err := captureRealStreams(t,
		"agent", "run", "--role", "reviewer", "--task", "t", "--tier", "mid",
		"--backend", "dry-run", "--timeout", "1m", "--repo-root", root,
	)
	if err != nil {
		t.Fatalf("agent run: %v", err)
	}

	var rec struct {
		Status   string `json:"status"`
		Pool     string `json:"pool"`
		Attempts []struct {
			Pool   string `json:"pool"`
			Status string `json:"status"`
		} `json:"attempts"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &rec); jsonErr != nil {
		t.Fatalf("stdout is not a record: %v (%q)", jsonErr, stdout)
	}
	if rec.Pool == "claude" {
		t.Error("the dispatch went to a denied pool")
	}
	if len(rec.Attempts) == 0 || rec.Attempts[0].Pool != "claude" || rec.Attempts[0].Status != "denied" {
		t.Errorf("attempts = %+v; the denial must leave a trace naming the pool", rec.Attempts)
	}
}

// Every pool denied is its own record, distinguishable from "no pool could
// serve this" — the two send an operator to different places.
func TestAgentRun_EveryPoolDeniedIsItsOwnOutcome(t *testing.T) {
	root := repoRootForTest(t)
	declareIdentity(t, "claude", "nan")

	stdout, _, err := captureRealStreams(t,
		"agent", "run", "--role", "r", "--task", "t", "--tier", "mid",
		"--backend", "dry-run", "--timeout", "1m", "--repo-root", root,
	)
	if err == nil {
		t.Fatal("a dispatch with every pool denied succeeded")
	}

	var rec struct {
		Status string `json:"status"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &rec); jsonErr != nil {
		t.Fatalf("no record for a walk that happened: %v (%q)", jsonErr, stdout)
	}
	if rec.Status != "denied" {
		t.Errorf("status = %q, want denied — not chain_exhausted, which would send an operator "+
			"looking at quota rather than at machine.json", rec.Status)
	}
	if got := ExitCode(err); got != 3 {
		t.Errorf("exit = %d, want 3: nothing ran, and another machine may be permitted to run it", got)
	}
}

// AC4's wording clause, asserted rather than trusted to review. The guarantee
// this tool can keep is narrow, and overstating it is the failure: a reader who
// believes exhaustion cannot happen will not build for the case where it does.
func TestAgentRun_StatesTheNarrowGuaranteeAndNotAWiderOne(t *testing.T) {
	stdout, _, err := captureRealStreams(t, "agent", "run", "--help")
	if err != nil {
		t.Fatalf("help: %v", err)
	}
	help := stdout

	if !strings.Contains(help, "dotf alone will never be the cause of exhaustion") {
		t.Error("help text does not state the narrow guarantee verbatim")
	}
	// The concurrency the map declares is shared with consumers dotf cannot
	// see, and the help has to say so or the narrow claim reads as a wide one.
	for _, want := range []string{"shared", "reserve"} {
		if !strings.Contains(strings.ToLower(help), want) {
			t.Errorf("help text does not mention %q, so the bound reads as absolute", want)
		}
	}
	// Claims the tool cannot keep.
	for _, forbidden := range []string{
		"exhaustion cannot happen",
		"guarantees that the pool",
		"will never be exhausted",
		"prevents exhaustion",
	} {
		if strings.Contains(strings.ToLower(help), strings.ToLower(forbidden)) {
			t.Errorf("help text overstates the guarantee with %q", forbidden)
		}
	}
}
