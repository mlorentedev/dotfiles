package prreview

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Until scripts/pr-agent-model-preflight.sh is deleted, it is the specification:
// the same inputs through the script (bash, jq and curl against a local server)
// and through Preflight must give the same outputs, the same warnings, the same
// summary, the same requests and the same exit class. The script goes when the
// workflow switches to the command, and this test goes with it.

type preflightCase struct {
	name, model, fallbacks, timeout string
	noKey                           bool
	modes                           map[string]string
}

// scriptRun is what the script did with one case.
type scriptRun struct {
	code                    int
	stdout, output, summary string
	requests                string
}

// requireScript is the path of scripts/<name>, or skips (fails on CI) when it
// cannot run here: it needs Linux and tools.
func requireScript(t *testing.T, name string, tools ...string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		// The script runs only on ubuntu runners, and jq on Windows ends each
		// line with CRLF, so the script's output differs there for reasons of
		// its own. The port is held to the script where the script runs.
		t.Skip("the script runs only on Linux runners")
	}
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			// On a Linux CI runner the script's tools are a given; their absence
			// there would skip every comparison silently, so it fails instead.
			if os.Getenv("CI") != "" {
				t.Fatalf("%s is not installed on this CI runner, so the port is not compared to the script", tool)
			}
			t.Skipf("%s is not installed", tool)
		}
	}
	script, err := filepath.Abs(repoFile("scripts", name))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(script); err != nil {
		t.Skipf("%s is not here", script)
	}
	return script
}

// runScript runs the script on c against api, with the output path given.
func runScript(t *testing.T, script string, api *nanAPI, c preflightCase, output string) scriptRun {
	t.Helper()
	summary := filepath.Join(t.TempDir(), "summary")
	env := []string{"PATH=" + os.Getenv("PATH"), "NAN_API_BASE=" + api.URL, "PREFLIGHT_TIMEOUT=" + c.timeout,
		"GITHUB_STEP_SUMMARY=" + summary}
	if !c.noKey {
		env = append(env, "NAN_API_KEY="+nanKey)
	}
	cmd := exec.CommandContext(t.Context(), "bash", script, "--model", c.model, "--fallbacks", c.fallbacks, "--output", output)
	cmd.Env = env
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var run scriptRun
	var ee *exec.ExitError
	if err := cmd.Run(); errors.As(err, &ee) {
		run.code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("running the script: %v", err)
	}
	run.stdout = stdout.String()
	if raw, err := os.ReadFile(output); err == nil {
		run.output = string(raw)
	}
	if raw, err := os.ReadFile(summary); err == nil {
		run.summary = string(raw)
	}
	run.requests = requestLog(api)
	return run
}

// requestLog is what the server saw, in order: the request line and body, and
// the headers the script sets itself.
func requestLog(api *nanAPI) string {
	var b strings.Builder
	for _, r := range api.requests() {
		fmt.Fprintf(&b, "%s|%s|%s|%s\n", r.Path, r.Auth, r.ContentType, r.Raw)
	}
	return b.String()
}

func TestPreflightMatchesTheScript(t *testing.T) {
	script := requireScript(t, "pr-agent-model-preflight.sh", "bash", "jq", "curl")
	for _, c := range preflightCases() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			elsewhere := newNanAPI(t, nil)
			api := newNanAPI(t, c.modes)
			api.redirect(elsewhere.URL + "/elsewhere")
			want := runScript(t, script, api, c, filepath.Join(t.TempDir(), "output"))
			if n := len(elsewhere.requests()); n != 0 {
				t.Fatalf("the script followed a redirect (%d requests)", n)
			}
			api.mu.Lock()
			api.reqs = nil
			api.mu.Unlock()

			in := PreflightInputs{Model: c.model, Fallbacks: c.fallbacks, Key: nanKey, APIBase: api.URL, Timeout: c.timeout}
			if c.noKey {
				in.Key = ""
			}
			var out bytes.Buffer
			res, err := Preflight(t.Context(), in, PreflightEnv{Out: &out})
			got := scriptRun{stdout: out.String(), requests: requestLog(api)}
			var setup PreflightSetupError
			switch {
			case err == nil:
				got.output = res.KeyValues()
				got.summary = res.Summary()
			case errors.Is(err, ErrNoModelAnswered):
				got.code, got.summary = 1, res.Summary()
			case errors.As(err, &setup):
				got.code = 2
			default:
				t.Fatalf("Preflight: %v", err)
			}
			if n := len(elsewhere.requests()); n != 0 {
				t.Fatalf("Preflight followed a redirect (%d requests)", n)
			}
			if got.code != want.code {
				t.Fatalf("exit class: Preflight %d (%v), script %d\n%s", got.code, err, want.code, want.stdout)
			}
			for what, pair := range map[string][2]string{"requests": {got.requests, want.requests},
				"annotations": {warningLines(got.stdout), warningLines(want.stdout)},
				"outputs":     {got.output, want.output}, "summary": {got.summary, want.summary}} {
				if pair[0] != pair[1] {
					t.Errorf("%s differ\nPreflight: %q\nscript:    %q", what, pair[0], pair[1])
				}
			}
			if got.stdout != want.stdout {
				t.Errorf("stdout differs\nPreflight: %q\nscript:    %q", got.stdout, want.stdout)
			}
		})
	}
}

func warningLines(text string) string {
	var keep []string
	for _, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(l, "::warning::") {
			keep = append(keep, l)
		}
	}
	return strings.Join(keep, "\n")
}

// preflightCases is the inputs the two implementations are held equal on: every
// status class in the primary's place and in the fallback's, the chains, the
// setup errors and the timeouts.
func preflightCases() []preflightCase {
	const model, fallbacks = "openai/mimo-v2.6-flash", `["openai/deepseek-v4-flash"]`
	var cases []preflightCase
	add := func(name, m, f string, modes map[string]string) {
		cases = append(cases, preflightCase{name: name, model: m, fallbacks: f, timeout: "0.5", modes: modes})
	}

	// Every status class, as the primary and as the last resort.
	codes := []string{"200", "201", "204", "299", "301", "307", "400", "401", "402", "403", "404", "408", "429",
		"500", "502", "503", "529", "redirect", "hang", "stall"}
	for _, code := range codes {
		for _, other := range []string{"200", "401"} {
			add("primary-"+code+"/fallback-"+other, model, fallbacks,
				map[string]string{"mimo-v2.6-flash": code, "deepseek-v4-flash": other})
			add("primary-"+other+"/fallback-"+code, model, fallbacks,
				map[string]string{"mimo-v2.6-flash": other, "deepseek-v4-flash": code})
		}
		add("both-"+code, model, fallbacks, map[string]string{"mimo-v2.6-flash": code, "deepseek-v4-flash": code})
	}

	// The fallback order, and which entries survive.
	chain := `["openai/b","openai/c","openai/d"]`
	for _, ms := range [][4]string{{"200", "200", "200", "200"}, {"503", "200", "200", "200"}, {"200", "429", "200", "200"},
		{"401", "401", "200", "200"}, {"401", "401", "401", "200"}, {"200", "200", "hang", "200"}, {"401", "401", "401", "401"},
		{"hang", "stall", "500", "404"}} {
		add("order-"+strings.Join(ms[:], "-"), "openai/a", chain,
			map[string]string{"a": ms[0], "b": ms[1], "c": ms[2], "d": ms[3]})
	}
	ok := map[string]string{"a": "200", "b": "200", "c": "503"}
	add("blank entries", "openai/a", `["","openai/b","","openai/c"]`, ok)
	add("padded entries", "openai/a", `[" openai/b ","\topenai/c"]`, ok)
	add("no prefix", "a", `["b","openai/c"]`, ok)
	add("a bare id with the prefix elsewhere", "a", `["openai/openai/b"]`, map[string]string{"openai/b": "200"})
	add("duplicates", "openai/a", `["openai/a","openai/b","openai/b"]`, ok)
	add("the primary again", "openai/a", `["openai/a"]`, map[string]string{"a": "401"})
	add("no fallbacks", "openai/a", `[]`, ok)
	add("null fallbacks", "openai/a", `null`, ok)
	add("no fallbacks, primary down", "openai/a", `[]`, map[string]string{"a": "500"})
	add("an id that needs escaping", "openai/a\"b", `["openai/c\\d"]`, map[string]string{`a"b`: "200", `c\d`: "200"})
	add("a non-ASCII id", "openai/né", `["openai/<b>&"]`, map[string]string{"né": "200", "<b>&": "200"})
	add("a model that is only a prefix", "openai/", `["openai/a"]`, ok)

	// Setup errors: nothing is probed.
	for _, f := range []string{`not json`, `{}`, `{"a":1}`, `"x"`, `5`, `true`, `[`, ``} {
		add("fallbacks-"+f, model, f, ok)
	}
	add("no model", "", fallbacks, ok)
	cases = append(cases, preflightCase{name: "no key", model: model, fallbacks: fallbacks, timeout: "0.5", noKey: true, modes: ok})

	// The timeout: unset, whole, fractional; a fast server never reaches it.
	for _, timeout := range []string{"", "1", "2.5", "0", "0.5"} {
		cases = append(cases, preflightCase{name: "timeout-" + timeout, model: model, fallbacks: fallbacks, timeout: timeout,
			modes: map[string]string{"mimo-v2.6-flash": "503", "deepseek-v4-flash": "200"}})
	}
	return cases
}

// Where the script is wrong the port is not, and each divergence is pinned on
// both sides so that neither moves unnoticed.
func TestPreflightDivergesFromTheScriptWhereTheScriptIsWrong(t *testing.T) {
	script := requireScript(t, "pr-agent-model-preflight.sh", "bash", "jq", "curl")
	up := map[string]string{"mimo-v2.6-flash": "200", "1": "200", "null": "200"}

	// A fallback that is not a model id: the script probes the JSON text of a
	// number and treats it as a model; the port refuses the chain.
	c := preflightCase{model: "openai/mimo-v2.6-flash", fallbacks: `[1]`, timeout: "1"}
	api := newNanAPI(t, up)
	if got := runScript(t, script, api, c, filepath.Join(t.TempDir(), "o")); got.code != 0 || !strings.Contains(got.requests, `"model":"1"`) {
		t.Errorf("the script no longer probes a numeric fallback: exit %d, %q", got.code, got.requests)
	}
	_, err := Preflight(t.Context(), PreflightInputs{Model: c.model, Fallbacks: c.fallbacks, Key: nanKey, APIBase: api.URL},
		PreflightEnv{})
	if !errors.As(err, new(PreflightSetupError)) {
		t.Errorf("Preflight on a numeric fallback: %v, want a setup error", err)
	}

	// A timeout curl cannot read: curl refuses the option before it connects, so
	// the script sends nothing, reads every model as unanswering ("NaN is down")
	// and exits 1. A typo in a variable moves every review off NaN. The port
	// falls back to its default.
	c = preflightCase{model: "openai/mimo-v2.6-flash", fallbacks: `[]`, timeout: "abc"}
	api = newNanAPI(t, up)
	if got := runScript(t, script, api, c, filepath.Join(t.TempDir(), "o")); got.code != 1 || !strings.Contains(got.stdout, "no answer within abcs") || got.requests != "" {
		t.Errorf("the script no longer reads an unreadable timeout as no answer: exit %d, %q", got.code, got.stdout)
	}
	res, err := Preflight(t.Context(), PreflightInputs{Model: c.model, Fallbacks: c.fallbacks, Key: nanKey, APIBase: api.URL, Timeout: "abc"},
		PreflightEnv{})
	if err != nil || res.Model != c.model {
		t.Errorf("Preflight on an unreadable timeout: %v, %q", err, res.Model)
	}

	// A timeout of 0: curl takes it as no limit at all, so against a server that
	// holds the connection the script waits for as long as the server does. The
	// port falls back to its default of 90 seconds.
	hung := newNanAPI(t, map[string]string{"mimo-v2.6-flash": "hang"})
	for timeout, ends := range map[string]bool{"0.5": true, "0": false} {
		ctx, cancel := context.WithTimeout(t.Context(), 1500*time.Millisecond)
		cmd := exec.CommandContext(ctx, "bash", script, "--model", "openai/mimo-v2.6-flash", "--fallbacks", "[]", "--output", os.DevNull)
		cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "NAN_API_KEY=" + nanKey, "NAN_API_BASE=" + hung.URL, "PREFLIGHT_TIMEOUT=" + timeout}
		cmd.Stdout = io.Discard
		_ = cmd.Run()
		if finished := ctx.Err() == nil; finished != ends {
			t.Errorf("the script with PREFLIGHT_TIMEOUT=%s finished within 1.5s: %v, want %v", timeout, finished, ends)
		}
		cancel()
	}

	// An output file that cannot be written: the script exits 1, the status that
	// means no model answered. The command exits 2 (see pr_preflight_test.go).
	c = preflightCase{model: "openai/mimo-v2.6-flash", fallbacks: `[]`, timeout: "1"}
	if got := runScript(t, script, newNanAPI(t, up), c, filepath.Join(t.TempDir(), "missing", "o")); got.code != 1 {
		t.Errorf("the script no longer exits 1 on an unwritable output: %d", got.code)
	}
}
