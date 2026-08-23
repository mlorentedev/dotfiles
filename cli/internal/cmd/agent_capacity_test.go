package cmd

import (
	"encoding/json"
	"testing"
)

// declaredCapacity turns the map's declaration into the number the semaphore
// enforces, and the interesting part is what it does with silence.
func TestDeclaredCapacity(t *testing.T) {
	var m map[string]any
	raw := `{
      "pools": {
        "nan":      {"auth": "subscription", "concurrency": 5, "reserve_interactive": 2, "probe": "env:X"},
        "claude":   {"auth": "subscription", "concurrency": 10, "probe": "bin:claude"},
        "copilot":  {"auth": "seat", "probe": "bin:copilot"},
        "greedy":   {"auth": "subscription", "concurrency": 2, "reserve_interactive": 5, "probe": "env:Y"},
        "zeroed":   {"auth": "subscription", "concurrency": 0, "probe": "env:Z"}
      }
    }`
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	capacity := declaredCapacity(m)

	tests := []struct {
		name         string
		pool         string
		want         int
		wantDeclared bool
		why          string
	}{
		{
			name: "the reserve is subtracted", pool: "nan", want: 3, wantDeclared: true,
			why: "5 declared minus a reserve of 2 leaves 3 for dispatched work; enforcing 5 would spend the reserve",
		},
		{
			name: "no reserve declared means none is held back", pool: "claude", want: 10, wantDeclared: true,
			why: "an absent reserve is zero reserve, not a default one this code invented",
		},
		{
			name: "a seat-based pool declares nothing", pool: "copilot", want: 0, wantDeclared: false,
			why: "concurrency is a fleet property there; reading silence as zero would refuse every dispatch",
		},
		{
			name: "a reserve larger than the pool floors at one", pool: "greedy", want: 1, wantDeclared: true,
			why: "2 minus 5 is negative; a declaration error must not silently disable dispatch entirely",
		},
		{
			name: "an explicit zero is a declaration and floors at one", pool: "zeroed", want: 1, wantDeclared: true,
			why: "distinct from copilot: the map DID state a number, so the pool is bounded rather than unbounded",
		},
		{
			name: "an undeclared pool is not capacity zero", pool: "nowhere", want: 0, wantDeclared: false,
			why: "a pool absent from the map cannot be bounded by it; the chain walk rejects the route elsewhere",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, declared := capacity(tc.pool)
			if declared != tc.wantDeclared {
				t.Errorf("declared = %v, want %v — %s", declared, tc.wantDeclared, tc.why)
			}
			if got != tc.want {
				t.Errorf("capacity = %d, want %d — %s", got, tc.want, tc.why)
			}
		})
	}
}
