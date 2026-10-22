package prreview

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The embedded pool is a build-time copy: go:embed cannot reach the repo's
// harness/ directory from here. The copy is the pool a consumer with no file of
// its own draws from, so a drift is a silent change of the shares CI runs.
func TestEmbeddedPoolEqualsTheRepoPool(t *testing.T) {
	want, err := os.ReadFile(repoFile("harness", "reviewer-pool.json"))
	if err != nil {
		// Fail, not skip: a test that skips when its source is gone checks nothing.
		t.Fatalf("harness/reviewer-pool.json is not here: %v", err)
	}
	if !bytes.Equal(embeddedPool, want) {
		t.Fatal("cli/internal/prreview/reviewer-pool.json differs from harness/reviewer-pool.json;" +
			" re-sync it: cp harness/reviewer-pool.json cli/internal/prreview/reviewer-pool.json")
	}
}

func TestLoadPool(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "pool.json")
	if err := os.WriteFile(good, []byte(`{"pool":[{"id":"nan/a","pr_agent":{"model":"openai/a","weight":1}}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{"pool":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if p, err := LoadPool("embedded"); err != nil || len(p.Pool) == 0 {
		t.Fatalf("embedded: %v, %d members", err, len(p.Pool))
	}
	if p, err := LoadPool(good); err != nil || p.Pool[0].PRAgent.Model != "openai/a" {
		t.Fatalf("file: %v, %+v", err, p)
	}
	for _, src := range []string{filepath.Join(dir, "missing.json"), bad, ""} {
		_, err := LoadPool(src)
		if err == nil || !strings.Contains(err.Error(), "cannot read a pool from") {
			t.Errorf("LoadPool(%q) = %v, want a refusal", src, err)
		}
	}
}

// The shipped pool is the one the route draws from, so it must parse through
// the shared parser with the blocks the route needs.
func TestEmbeddedPoolCarriesTheRouteBlocks(t *testing.T) {
	p, err := LoadPool("embedded")
	if err != nil {
		t.Fatal(err)
	}
	var risk, weighted int
	for _, e := range p.Pool {
		if e.PRAgent == nil {
			continue
		}
		if e.PRAgent.Route == "risk" {
			risk++
		}
		if e.PRAgent.Weight > 0 {
			weighted++
		}
	}
	if risk != 1 || weighted == 0 || p.PRAgentRisk == nil || p.PRAgentRisk.MinChangedLines == nil {
		t.Errorf("risk members %d, weighted %d, risk line %+v", risk, weighted, p.PRAgentRisk)
	}
}
