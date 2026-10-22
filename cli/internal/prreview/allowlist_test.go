package prreview

import (
	"maps"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Every model CI may spend on carries its prices and the most one review of it
// may cost, so a model cannot be admitted without the arithmetic that bounds
// its spend (ADR-046 decision 5).
func TestEveryAllowedModelHasAPriceRowAndACeiling(t *testing.T) {
	if len(Allowlist()) == 0 {
		t.Fatal("the allowlist is empty")
	}
	for _, m := range Allowlist() {
		if !strings.HasPrefix(m.ID, "anthropic/") {
			t.Errorf("%q is not an anthropic/ model id", m.ID)
		}
		if m.InputUSDPerMTok <= 0 || m.OutputUSDPerMTok <= 0 || m.CeilingUSD <= 0 {
			t.Errorf("%s has no price row or ceiling: %+v", m.ID, m)
		}
	}
}

func TestAllowedIsExact(t *testing.T) {
	for id, want := range map[string]bool{
		"anthropic/claude-haiku-5-5":      true,
		"anthropic/claude-sonnet-5-5":     true,
		"anthropic/claude-opus-5-5":       false,
		"anthropic/claude-haiku-5-5-x":    false,
		"claude-haiku-5-5":                false,
		"":                                false,
		"anthropic/claude-sonnet-5-5 ":    false,
		"openai/claude-sonnet-5-5":        false,
		" anthropic/claude-haiku-5-5":     false,
		"anthropic/claude-haiku-5-5 more": false,
	} {
		if got := Allowed(id); got != want {
			t.Errorf("Allowed(%q) = %v, want %v", id, got, want)
		}
	}
}

// Until the script is deleted, the two allowlists must be one list: a model
// admitted in one and not the other would route differently depending on which
// reads the pool.
func TestAllowlistEqualsTheScripts(t *testing.T) {
	raw, err := os.ReadFile(repoFile("scripts", "pr-agent-route.sh"))
	if err != nil {
		t.Skipf("scripts/pr-agent-route.sh is not here: %v", err)
	}
	m := regexp.MustCompile(`(?m)^ALLOWED_ANTHROPIC="([^"]*)"$`).FindSubmatch(raw)
	if m == nil {
		t.Fatal("scripts/pr-agent-route.sh no longer declares ALLOWED_ANTHROPIC")
	}
	want := strings.Fields(string(m[1]))
	var got []string
	for _, a := range Allowlist() {
		got = append(got, a.ID)
	}
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("Go allowlist %v differs from the script's %v", got, want)
	}
}

// Until the workflow's worst-review test reads this list, the bats suite holds a
// second copy of the price rows. The two must be one table: a price changed in
// one would bound the spend by a number the router does not use.
func TestAllowlistEqualsTheBatsPrices(t *testing.T) {
	raw, err := os.ReadFile(repoFile("tests", "pr-agent-config.bats"))
	if err != nil {
		t.Skipf("tests/pr-agent-config.bats is not here: %v", err)
	}
	_, dict, ok := strings.Cut(string(raw), "prices = {")
	if !ok {
		t.Fatal("tests/pr-agent-config.bats no longer declares the `prices` table")
	}
	dict, _, _ = strings.Cut(dict, "}")
	rows := regexp.MustCompile(`'(anthropic/[^']+)':\s*\(([\d.]+),\s*([\d.]+),\s*([\d.]+)\)`).FindAllStringSubmatch(dict, -1)
	got := map[string]Model{}
	for _, m := range rows {
		in, _ := strconv.ParseFloat(m[2], 64)
		out, _ := strconv.ParseFloat(m[3], 64)
		ceiling, _ := strconv.ParseFloat(m[4], 64)
		got[m[1]] = Model{ID: m[1], InputUSDPerMTok: in, OutputUSDPerMTok: out, CeilingUSD: ceiling}
	}
	want := map[string]Model{}
	for _, m := range Allowlist() {
		want[m.ID] = m
	}
	if len(got) == 0 || !maps.Equal(got, want) {
		t.Errorf("bats prices %+v\nGo allowlist %+v", got, want)
	}
}
