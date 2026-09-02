package harness

import (
	"path/filepath"
	"strings"
	"testing"
)

// skillCapability is the neutral verb that grants a persona the ability to
// invoke a skill. Named once here so the guard and its message cannot drift.
const skillCapability = "skill"

// TestEveryPersonaDeclaringSkillsCanInvokeThem is the guard for #1420, and it is
// the one that would have caught the defect the whole spec is about.
//
// The gate requires a persona to CONSUME its forced skills. Consuming one means
// invoking the harness's skill primitive. On claude that primitive is a tool, and
// `tools:` is an ALLOW-LIST — a tool not named is unavailable — so a record that
// declares `skills:` while declaring no `skill` capability deploys an agent that
// is required to do something it was never given the means to do.
//
// Measured 2026-09-01, before the fix: all seven personas declared skills and
// none declared the capability. Nothing reported it. `dotf doctor` reported the
// roster healthy, the deployed files were exactly what the SSOT asked for, and
// every check that existed passed — because each layer was individually correct
// and no check spanned the two frontmatter keys whose relationship is the
// requirement.
//
// Under `enforce: warn` that is invisible noise. Under `enforce: block` it is a
// hard deadlock: every tool call refused, and the one action that would clear it
// unavailable. `dotf harness gate` guards against blocking a skill invocation
// ("forbidding it would deadlock the session") one layer ABOVE where the deadlock
// actually is.
func TestEveryPersonaDeclaringSkillsCanInvokeThem(t *testing.T) {
	dir := filepath.Join(repoRootForTest(t), "harness", "agents")
	personas, err := LoadPersonas(dir)
	if err != nil {
		t.Fatalf("load personas from %s: %v", dir, err)
	}

	// The guard's own fixture check. A roster that loaded but declares no forced
	// skills anywhere would pass this test while asserting nothing — the vacuous
	// pass this repository keeps re-learning.
	declaring := 0
	var offenders []string
	for _, p := range personas {
		if len(p.Skills) == 0 {
			continue
		}
		declaring++
		if !contains(p.Capabilities, skillCapability) {
			offenders = append(offenders, p.Name+" ("+filepath.Base(filepath.Dir(p.Path))+")")
		}
	}
	if declaring == 0 {
		t.Fatalf("no persona under %s declares any forced skills; either the roster changed or "+
			"this guard is no longer measuring what it claims", dir)
	}

	if len(offenders) > 0 {
		t.Fatalf("these personas declare forced skills but no %q capability, so the deployed agent\n"+
			"cannot invoke the skills its own gate demands (#1420):\n  %s\n\n"+
			"Fix in the vault SSOT — 00_meta/agents/definitions/<role>/AGENT.md — and redeploy.\n"+
			"Editing harness/agents/ or ~/.claude/agents/ directly is overwritten by the next\n"+
			"compile, and leaves the SSOT still wrong.",
			skillCapability, strings.Join(offenders, "\n  "))
	}
}

// TestSkillCapabilityResolvesForEveryTargetedHarness ties the record-level
// declaration to the thing it has to produce. Declaring the verb is worthless if
// the map cannot turn it into a native grant, and the two live in different files
// under different review — exactly the seam #1420 fell through.
//
// A harness that declares the verb unsupported is skipped rather than failed: it
// answered the question, and that answer is the design decision recorded in
// HARNESS-106. What must never happen is a verb that is neither mapped nor
// declared, and the loader already refuses such a map outright.
func TestSkillCapabilityResolvesForEveryTargetedHarness(t *testing.T) {
	root := repoRootForTest(t)
	m, err := LoadCapabilityMap(root)
	if err != nil {
		t.Fatalf("load capability map: %v", err)
	}
	harnesses, _ := m["harnesses"].(map[string]any)
	if len(harnesses) == 0 {
		t.Fatal("shipped map declares no harnesses")
	}

	checked := 0
	for _, name := range sortedKeys(harnesses) {
		unsup, err := UnsupportedFor(m, []string{skillCapability}, name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if len(unsup) > 0 {
			continue
		}
		checked++
		line, err := ResolveCapabilities(m, []string{skillCapability}, name)
		if err != nil {
			t.Errorf("%s maps %q but cannot resolve it: %v", name, skillCapability, err)
			continue
		}
		if strings.TrimSpace(line) == "" {
			t.Errorf("%s resolves %q to an empty line — an empty grant is not a grant", name, skillCapability)
		}
	}
	if checked == 0 {
		t.Fatalf("every declared harness reports %q unsupported; no persona can invoke a skill "+
			"anywhere, which cannot be the intended state", skillCapability)
	}
}
