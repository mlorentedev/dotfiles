package cmd

import (
	"encoding/json"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/agent"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// What these pin is the WIRING between the command and the dispatcher: the
// dispatcher's own tests (internal/agent) hand it a semaphore and a capacity,
// and the capacity function has its own table (TestDeclaredCapacity). Neither
// would notice if `agent run` stopped passing them on, which is the property a
// shell caller of the real binary depends on: a pool the map bounds is bounded.
//
// They replace a bats case that held real flocks from a python helper and ran
// the compiled binary to prove the same thing, one build and one interpreter
// start per case.
func TestAgentRun_ASaturatedPoolAdvancesTheChainThroughTheCommand(t *testing.T) {
	root := repoRootForTest(t)
	declareIdentity(t)

	m, err := harness.LoadModelMap(root)
	if err != nil {
		t.Fatalf("load the shipped map: %v", err)
	}
	// The dispatchable share of nan's pool: what the map declares minus the slots
	// it reserves for an interactive session. Derived rather than written as 5, so
	// the case follows the map; TestDeclaredCapacity is what pins the subtraction.
	capacity, declared := declaredCapacity(m)("nan")
	if !declared || capacity < 2 {
		t.Fatalf("nan capacity = %d (declared %v); the case needs a pool that can be both partly and fully held", capacity, declared)
	}

	run := func(t *testing.T, held int) (pool, firstAttempt, firstStatus string) {
		t.Helper()
		dir := t.TempDir()
		sem := agent.NewSemaphore(dir)
		for i := 0; i < held; i++ {
			slot, err := sem.Acquire("nan", capacity)
			if err != nil {
				t.Fatalf("hold slot %d: %v", i, err)
			}
			t.Cleanup(slot.Release)
		}
		stdout, _, err := captureRealStreams(t,
			"agent", "run", "--role", "r", "--task", "t", "--tier", "low",
			"--backend", "dry-run", "--timeout", "1m", "--repo-root", root,
			"--semaphore-dir", dir,
		)
		if err != nil {
			t.Fatalf("agent run: %v", err)
		}
		var rec struct {
			Pool     string `json:"pool"`
			Attempts []struct {
				Pool   string `json:"pool"`
				Status string `json:"status"`
			} `json:"attempts"`
		}
		if jsonErr := json.Unmarshal([]byte(stdout), &rec); jsonErr != nil {
			t.Fatalf("stdout is not a record: %v (%q)", jsonErr, stdout)
		}
		if len(rec.Attempts) == 0 {
			t.Fatalf("the record carries no attempts: %q", stdout)
		}
		return rec.Pool, rec.Attempts[0].Pool, rec.Attempts[0].Status
	}

	t.Run("one slot left is served by nan", func(t *testing.T) {
		pool, first, status := run(t, capacity-1)
		if pool != "nan" || first != "nan" || status != "dry_run" {
			t.Errorf("served by %s (first attempt %s: %s), want nan: the pool still had a free slot", pool, first, status)
		}
	})

	t.Run("every dispatchable slot held advances to the next entry", func(t *testing.T) {
		pool, first, status := run(t, capacity)
		if first != "nan" || status != "pool_unavailable" {
			t.Errorf("first attempt = %s: %s, want nan: pool_unavailable", first, status)
		}
		if pool != "claude" {
			t.Errorf("served by %q, want claude: the low chain is nan then claude, and nan was full", pool)
		}
	})
}

// The top chain is one entry on purpose (ADR-032 §4), and the entry the shipped
// map declares is claude. harness.ResolveChain's own test pins the length; this
// pins that the command, given no fixture, walks exactly that one entry.
func TestAgentRun_TheTopTierWalksItsSingleDeclaredEntry(t *testing.T) {
	root := repoRootForTest(t)
	declareIdentity(t)

	stdout, _, err := captureRealStreams(t,
		"agent", "run", "--role", "architect", "--task", "decide", "--tier", "top",
		"--backend", "dry-run", "--timeout", "1m", "--repo-root", root,
		"--semaphore-dir", t.TempDir(),
	)
	if err != nil {
		t.Fatalf("agent run: %v", err)
	}
	var rec struct {
		Pool     string        `json:"pool"`
		Attempts []interface{} `json:"attempts"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &rec); jsonErr != nil {
		t.Fatalf("stdout is not a record: %v (%q)", jsonErr, stdout)
	}
	if len(rec.Attempts) != 1 {
		t.Errorf("attempts = %d, want 1: the top tier has no fallback to walk", len(rec.Attempts))
	}
	if rec.Pool != "claude" {
		t.Errorf("pool = %q, want claude", rec.Pool)
	}
}
