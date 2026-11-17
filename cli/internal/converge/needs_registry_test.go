package converge

import (
	"slices"
	"testing"
)

// Every production step says what it needs. legacy-setup is the one that must
// not: it needs every step before it, and a list would go stale with the next
// step added.
func TestRegistry_EveryStepDeclaresItsNeeds(t *testing.T) {
	for _, r := range Registry(Options{}) {
		_, declares := r.(Needer)
		if want := r.Name() != "legacy-setup"; declares != want {
			t.Errorf("%s: declares its needs = %v, want %v", r.Name(), declares, want)
		}
	}
}

// A need names a step of the registry that comes earlier: a typo, or a need
// on a later step, would hold nothing back and say nothing.
func TestRegistry_EachNeedIsAnEarlierStep(t *testing.T) {
	reg := Registry(Options{})
	var seen []string
	for i, r := range reg {
		for _, n := range needsOf(reg, i) {
			if !slices.Contains(seen, n) {
				t.Errorf("%s needs %q, which is not a step before it", r.Name(), n)
			}
		}
		seen = append(seen, r.Name())
	}
}

// The graph, read as the run reads it. BUG-173 is the first group: a step
// nothing native needs no longer keeps the rest from being reached.
func TestRegistry_WhatAFailureHoldsBack(t *testing.T) {
	reg := Registry(Options{})
	var all []string
	for _, r := range reg {
		all = append(all, r.Name())
	}
	legacyOnly := []string{"legacy-setup"}
	for _, tc := range []struct {
		failed string
		want   []string
	}{
		{"system-upgrade", legacyOnly},
		{"cache-gc", legacyOnly},
		{"pwsh-modules", legacyOnly},
		{"orca-hooks", legacyOnly},
		{"shell-completions", legacyOnly},
		{"ts-bridge-client", legacyOnly},
		{"git-lfs", legacyOnly},
		{"env-persist", legacyOnly},
		{"self-deploy", legacyOnly},
		{"hive-upgrade-retire", legacyOnly},
		{"workstation-target", legacyOnly},
		{"agent-extensions", legacyOnly},
		{"git-config", legacyOnly},
		{"lanes", legacyOnly},
		{"git-hooks", []string{"lanes", "legacy-setup"}},
		{"records-bind", []string{"agent-extensions", "legacy-setup"}},
		{"pi-settings", []string{"configs-deploy", "records-bind", "git-config", "agent-extensions", "legacy-setup"}},
		{"configs-deploy", []string{"records-bind", "git-config", "agent-extensions", "legacy-setup"}},
		// ADR-041 decision 4: nothing runs on a checkout or on records that failed.
		{"checkout", all[1:]},
		{"records-mirror", []string{
			"env-generate", "records-harness", "tools", "system-upgrade", "cache-gc", "pwsh-modules",
			"pi-settings", "configs-deploy", "shell-completions", "records-bind", "git-config", "git-lfs",
			"git-hooks", "lanes", "env-persist", "agent-extensions", "legacy-setup",
		}},
		// No agent is installed before its instructions exist (#2016).
		{"records-harness", []string{
			"tools", "system-upgrade", "cache-gc", "pwsh-modules", "pi-settings", "configs-deploy",
			"shell-completions", "records-bind", "git-config", "git-lfs", "git-hooks", "lanes",
			"env-persist", "agent-extensions", "legacy-setup",
		}},
	} {
		if got := Unreached(reg, tc.failed); !slices.Equal(got, tc.want) {
			t.Errorf("%s fails: not reached %v, want %v", tc.failed, got, tc.want)
		}
	}
}

// The quiet-day update runs system-upgrade and cache-gc alone, and reads a
// skipped system-upgrade as the upgrade's own skip (cmd/update.go). Neither
// may come to need the other.
func TestRegistry_TheQuietDayStepsDoNotNeedEachOther(t *testing.T) {
	reg := Registry(Options{})
	for i, r := range reg {
		for _, pair := range [][2]string{{"system-upgrade", "cache-gc"}, {"cache-gc", "system-upgrade"}} {
			if r.Name() == pair[0] && slices.Contains(needsOf(reg, i), pair[1]) {
				t.Errorf("%s needs %s: a quiet day would report one's failure as the other's skip", pair[0], pair[1])
			}
		}
	}
}
