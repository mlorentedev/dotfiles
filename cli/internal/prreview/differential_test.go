package prreview

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Until scripts/pr-agent-route.sh is deleted, it is the specification: the same
// inputs through the script (bash, jq and curl against a local server) and
// through Route must give the same outputs, the same warnings, the same probes
// and the same exit class. The script goes when the workflow switches to the
// command, and this test goes with it.

const (
	noWeightsPool = `{"pool": [
 {"id": "nan/a", "pr_agent": {"model": "openai/a"}},
 {"id": "anthropic-review/claude-haiku-5-5", "signs": "fallback", "pr_agent": {"model": "anthropic/claude-haiku-5-5"}},
 {"id": "anthropic-review/claude-sonnet-5-5", "signs": "second", "pr_agent": {"model": "anthropic/claude-sonnet-5-5", "route": "risk"}}
], "pr_agent_risk": {"min_changed_lines": 900, "label": "deep-review"}}`
	noRiskPool = `{"pool": [
 {"id": "nan/mimo-v2.6-flash", "pr_agent": {"model": "openai/mimo-v2.6-flash", "weight": 3}},
 {"id": "anthropic-review/claude-haiku-5-5", "signs": "fallback", "pr_agent": {"model": "anthropic/claude-haiku-5-5", "weight": 2}}
]}`
	nanOnlyPool = `{"pool": [
 {"id": "nan/mimo-v2.6-flash", "pr_agent": {"model": "openai/mimo-v2.6-flash", "weight": 3}},
 {"id": "nan/deepseek-v4-flash", "pr_agent": {"model": "openai/deepseek-v4-flash", "weight": 2}}
]}`
	badModelPool = `{"pool": [
 {"id": "nan/mimo-v2.6-flash", "pr_agent": {"model": "openai/mimo-v2.6-flash", "weight": 3}},
 {"id": "anthropic-review/claude-opus-5-5", "signs": "second", "pr_agent": {"model": "anthropic/claude-opus-5-5", "route": "risk"}}
]}`
)

type diffCase struct {
	name  string
	pool  string
	in    Inputs
	modes map[string]string // by model, as fakeAPI
	def   string
}

func TestRouteMatchesTheScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		// The script runs only on ubuntu runners, and jq on Windows ends each
		// line with CRLF, so its allowlist match fails on ids it accepts on
		// Linux. The port is held to the script where the script runs.
		t.Skip("the script runs only on Linux runners")
	}
	for _, tool := range []string{"bash", "jq", "curl"} {
		if _, err := exec.LookPath(tool); err != nil {
			// On a Linux CI runner the script's tools are a given; their absence
			// there would skip every comparison silently, so it fails instead.
			if os.Getenv("CI") != "" {
				t.Fatalf("%s is not installed on this CI runner, so the port is not compared to the script", tool)
			}
			t.Skipf("%s is not installed", tool)
		}
	}
	script, err := filepath.Abs(repoFile("scripts", "pr-agent-route.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(script); err != nil {
		t.Skipf("%s is not here", script)
	}

	dir := t.TempDir()
	pools := map[string]string{"fixture": fixturePool, "noweights": noWeightsPool, "norisk": noRiskPool,
		"nanonly": nanOnlyPool, "badmodel": badModelPool}
	paths := map[string]string{}
	for name, raw := range pools {
		paths[name] = filepath.Join(dir, name+".json")
		if err := os.WriteFile(paths[name], []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	cases := differentialCases()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			compareWithScript(t, script, paths[c.pool], c)
		})
	}
}

func compareWithScript(t *testing.T, script, poolPath string, c diffCase) {
	t.Helper()
	api := newFakeAPI(t)
	for m, mode := range c.modes {
		api.answer(m, mode)
	}
	if c.def != "" {
		api.answer("", c.def)
	}
	in := c.in
	in.APIBase, in.ProbeTimeout, in.Repository, in.AnthropicKey = api.URL, "1", "o/r", c.in.AnthropicKey

	out := filepath.Join(t.TempDir(), "output")
	cmd := exec.CommandContext(t.Context(), "bash", script, "--output", out)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"), "POOL_FILE=" + poolPath, "ANTHROPIC_API_BASE=" + in.APIBase,
		"ROUTE_PROBE_TIMEOUT=" + in.ProbeTimeout, "GITHUB_REPOSITORY=" + in.Repository,
		"NAN_OUTCOME=" + in.NanOutcome, "NAN_MODEL=" + in.NanModel, "NAN_FALLBACKS=" + in.NanFallbacks,
		"NAN_PREFLIGHT_EXIT=" + in.PreflightExit, "PR_AGENT_ANTHROPIC_API_KEY=" + in.AnthropicKey,
		"PR_AGENT_PROVIDER=" + in.Provider, "PR_CHANGED_LINES=" + in.ChangedLines, "PR_LABELS=" + in.Labels,
		"PR_AGENT_DRAW=" + in.Draw,
	}
	var scriptOut bytes.Buffer
	cmd.Stdout = &scriptOut
	runErr := cmd.Run()
	wantCode := 0
	var ee *exec.ExitError
	if errors.As(runErr, &ee) {
		wantCode = ee.ExitCode()
	} else if runErr != nil {
		t.Fatalf("running the script: %v", runErr)
	}
	scriptProbes := api.probedModels()
	api.mu.Lock()
	api.reqs = nil
	api.mu.Unlock()

	pool := mustPool(t, poolText(t, poolPath))
	var goOut bytes.Buffer
	res, err := Route(t.Context(), pool, poolPath, in, Env{Out: &goOut})

	gotCode := 0
	switch {
	case err == nil:
	case errors.Is(err, ErrRefused):
		gotCode = 1
	case isUsage(err):
		gotCode = 2
	default:
		t.Fatalf("Route: %v", err)
	}
	if gotCode != wantCode {
		t.Fatalf("exit class: Route %d (%v), script %d\n%s", gotCode, err, wantCode, scriptOut.String())
	}
	if got := api.probedModels(); got != scriptProbes {
		t.Errorf("probes: Route %q, script %q", got, scriptProbes)
	}
	if w, g := annotations(scriptOut.String(), "::warning::"), annotations(goOut.String(), "::warning::"); w != g {
		t.Errorf("warnings differ\nscript: %s\nRoute:  %s", w, g)
	}
	raw, _ := os.ReadFile(out)
	if wantCode != 0 {
		if len(raw) != 0 {
			t.Errorf("the script wrote outputs on exit %d: %q", wantCode, raw)
		}
		return
	}
	if got := res.KeyValues(); got != string(raw) {
		t.Errorf("outputs differ\nscript: %q\nRoute:  %q", raw, got)
	}
}

func poolText(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func annotations(text, prefix string) string {
	var keep []string
	for _, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(l, prefix) {
			keep = append(keep, l)
		}
	}
	return strings.Join(keep, "\n")
}

// differentialCases is the inputs the two implementations are held equal on:
// every point of the fixture pool's total weight, then the scenarios that move
// a member out of the draw, then the pools and the failures.
func differentialCases() []diffCase {
	healthy := Inputs{NanOutcome: "success", NanModel: "openai/mimo-v2.6-flash",
		NanFallbacks: `["openai/deepseek-v4-flash"]`, AnthropicKey: testKey, ChangedLines: "10"}
	with := func(f func(*Inputs)) Inputs { in := healthy; f(&in); return in }

	var cases []diffCase
	add := func(name, pool string, in Inputs, modes map[string]string, def string) {
		cases = append(cases, diffCase{name: name, pool: pool, in: in, modes: modes, def: def})
	}

	// Every point of the total weight, by provider and by size.
	for _, provider := range []string{"", "draw", "nan", "anthropic"} {
		for _, lines := range []string{"10", "900"} {
			for p := range 10 {
				add(fmt.Sprintf("point%d/%s/lines%s", p, orDash(provider), lines), "fixture",
					with(func(i *Inputs) { i.Provider, i.ChangedLines, i.Draw = provider, lines, fmt.Sprint(p) }), nil, "")
			}
		}
	}

	// The scenarios that move a member out of the draw.
	nanStates := map[string]func(*Inputs){
		"nanup":      func(*Inputs) {},
		"nandown":    func(i *Inputs) { i.NanOutcome, i.NanModel, i.NanFallbacks = "failure", "", "" },
		"preflight2": func(i *Inputs) { i.NanOutcome, i.NanModel, i.NanFallbacks, i.PreflightExit = "failure", "", "", "2" },
		"preflight1": func(i *Inputs) { i.NanOutcome, i.NanModel, i.NanFallbacks, i.PreflightExit = "failure", "", "", "1" },
	}
	apis := map[string]struct {
		modes map[string]string
		def   string
	}{
		"ok":        {nil, ""},
		"all401":    {nil, "401"},
		"sonnet529": {map[string]string{"claude-sonnet-5-5": "529"}, ""},
		"haiku404":  {map[string]string{"claude-haiku-5-5": "404"}, ""},
		"all429":    {nil, "429"},
	}
	for _, provider := range []string{"", "nan", "anthropic", "nan-only", "nan_only"} {
		for nanName, nanState := range nanStates {
			for apiName, api := range apis {
				for _, keyed := range []bool{true, false} {
					for _, lines := range []string{"", "900"} {
						name := fmt.Sprintf("%s/%s/%s/key%v/lines%s", orDash(provider), nanName, apiName, keyed, orDash(lines))
						add(name, "fixture", with(func(i *Inputs) {
							nanState(i)
							i.Provider, i.ChangedLines, i.Draw = provider, lines, "0"
							if !keyed {
								i.AnthropicKey = ""
							}
						}), api.modes, api.def)
					}
				}
			}
		}
	}

	// The label, and what is not the label.
	for _, labels := range []string{`["deep-review"]`, `["docs","deep-review"]`, `["deep-review-later"]`, `[]`, `not json`, ``} {
		add("labels/"+labels, "fixture", with(func(i *Inputs) { i.ChangedLines, i.Labels, i.Draw = "3", labels, "0" }), nil, "")
	}

	// Other pools.
	for _, pool := range []string{"noweights", "norisk", "nanonly", "badmodel"} {
		for _, provider := range []string{"", "nan", "anthropic"} {
			for _, lines := range []string{"10", "5000"} {
				add(fmt.Sprintf("pool-%s/%s/lines%s", pool, orDash(provider), lines), pool,
					with(func(i *Inputs) { i.Provider, i.ChangedLines, i.Draw = provider, lines, "0" }), nil, "")
			}
		}
	}

	// A fallback chain with a model the pool does not weigh, an empty one, and a
	// malformed one.
	for _, chain := range []string{`["openai/deepseek-v4-flash","openai/glm5.3-flash"]`, `[]`, ``, `nope`, `["","openai/deepseek-v4-flash"]`} {
		add("chain/"+chain, "fixture", with(func(i *Inputs) { i.NanFallbacks, i.Draw = chain, "0" }), nil, "")
	}

	// Draw points that are not points.
	for _, draw := range []string{"10", "x", "-1", "3x", "08", "007"} {
		add("draw/"+draw, "fixture", with(func(i *Inputs) { i.Draw = draw }), nil, "")
	}

	// Servers that never answer: the probe ends at its timeout.
	for _, mode := range []string{"hang", "stall"} {
		add("silent/"+mode, "fixture", with(func(i *Inputs) { i.Draw = "0" }), nil, mode)
	}
	return cases
}

func orDash(s string) string {
	if s == "" {
		return "default"
	}
	return s
}
