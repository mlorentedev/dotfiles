package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const routeTestKey = "sk-ant-test-not-a-real-key"

const routeTestPool = `{"pool": [
 {"id": "nan/mimo-v2.6-flash", "pr_agent": {"model": "openai/mimo-v2.6-flash", "weight": 3}},
 {"id": "nan/deepseek-v4-flash", "pr_agent": {"model": "openai/deepseek-v4-flash", "weight": 2}},
 {"id": "anthropic-review/claude-haiku-5-5", "signs": "fallback",
  "pr_agent": {"model": "anthropic/claude-haiku-5-5", "weight": 5, "reasoning_effort": "high"}},
 {"id": "anthropic-review/claude-sonnet-5-5", "signs": "second",
  "pr_agent": {"model": "anthropic/claude-sonnet-5-5", "route": "risk"}}
], "pr_agent_risk": {"min_changed_lines": 900, "label": "deep-review"}}`

// routeEnv sets the workflow's inputs for one run of `pr route`, with every
// Anthropic probe answered 200 by a local server, and returns the pool's path.
func routeEnv(t *testing.T) (pool, dir string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("{}")) }))
	t.Cleanup(srv.Close)
	dir = t.TempDir()
	pool = filepath.Join(dir, "pool.json")
	if err := os.WriteFile(pool, []byte(routeTestPool), 0o600); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{
		"NAN_OUTCOME": "success", "NAN_MODEL": "openai/mimo-v2.6-flash", "NAN_FALLBACKS": `["openai/deepseek-v4-flash"]`,
		"PR_AGENT_ANTHROPIC_API_KEY": routeTestKey, "ANTHROPIC_API_BASE": srv.URL, "ROUTE_PROBE_TIMEOUT": "2",
		"GITHUB_REPOSITORY": "o/r", "PR_AGENT_DRAW": "0", "PR_AGENT_PROVIDER": "", "PR_CHANGED_LINES": "",
		"PR_LABELS": "", "NAN_PREFLIGHT_EXIT": "", "GITHUB_STEP_SUMMARY": "",
	} {
		t.Setenv(k, v)
	}
	return pool, dir
}

func outputLines(t *testing.T, path string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	kv := map[string]string{}
	for _, l := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if k, v, ok := strings.Cut(l, "="); ok {
			kv[k] = v
		}
	}
	return kv
}

func TestPrRoute_WritesTheOutputsTheWorkflowReadsAndNamesThePool(t *testing.T) {
	pool, dir := routeEnv(t)
	out := filepath.Join(dir, "github_output")
	summary := filepath.Join(dir, "summary")
	t.Setenv("GITHUB_STEP_SUMMARY", summary)
	t.Setenv("PR_AGENT_DRAW", "5")

	stdout, stderr, err := execute(t, "pr", "route", "--pool", pool, "--output", out)
	if err != nil {
		t.Fatalf("pr route: %v\n%s", err, stderr)
	}
	kv := outputLines(t, out)
	if kv["first"] != "anthropic" || kv["second"] != "nan" || kv["reviewer"] != "anthropic/claude-haiku-5-5" ||
		kv["route"] != "draw" || kv["anthropic_effort"] != "high" {
		t.Errorf("outputs = %v", kv)
	}
	if !strings.Contains(stdout, "::notice::PR-Agent reviews first on anthropic/claude-haiku-5-5") || !strings.Contains(stdout, "pool: "+pool) {
		t.Errorf("stdout = %q", stdout)
	}
	sum, _ := os.ReadFile(summary)
	if !strings.Contains(string(sum), "Pool: `"+pool+"`. dotf: `dev`.") || !strings.Contains(string(sum), "Drawn to review first") {
		t.Errorf("summary = %q", sum)
	}
	for name, text := range map[string]string{"stdout": stdout, "stderr": stderr, "output": readFile(t, out), "summary": string(sum)} {
		if strings.Contains(text, "sk-ant") {
			t.Errorf("%s carries the key", name)
		}
	}
}

// --output appends, as `>> $GITHUB_OUTPUT` does: the step's other outputs stay.
func TestPrRoute_AppendsToTheOutputFile(t *testing.T) {
	pool, dir := routeEnv(t)
	out := filepath.Join(dir, "github_output")
	if err := os.WriteFile(out, []byte("earlier=kept\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := execute(t, "pr", "route", "--pool", pool, "--output", out); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, out); !strings.HasPrefix(got, "earlier=kept\nfirst=") {
		t.Errorf("output = %q", got)
	}
}

func TestPrRoute_WithNoOutputFileTheLinesGoToStdoutAfterTheAnnotations(t *testing.T) {
	pool, _ := routeEnv(t)
	stdout, _, err := execute(t, "pr", "route", "--pool", pool)
	if err != nil {
		t.Fatal(err)
	}
	if n, f := strings.Index(stdout, "::notice::"), strings.Index(stdout, "first=nan\n"); n < 0 || f < n {
		t.Errorf("stdout = %q", stdout)
	}
}

// A caller with no GitHub context passes the PR's size and labels as flags, and
// they win over the environment.
func TestPrRoute_FlagsOverrideTheEnvironment(t *testing.T) {
	pool, dir := routeEnv(t)
	out := filepath.Join(dir, "o")
	t.Setenv("PR_CHANGED_LINES", "3")
	t.Setenv("PR_LABELS", `[]`)

	if _, _, err := execute(t, "pr", "route", "--pool", pool, "--output", out, "--changed-lines", "1000"); err != nil {
		t.Fatal(err)
	}
	if kv := outputLines(t, out); kv["route"] != "risk" || kv["reviewer"] != "anthropic/claude-sonnet-5-5" {
		t.Errorf("--changed-lines 1000: %v", kv)
	}
	if err := os.Remove(out); err != nil {
		t.Fatal(err)
	}
	if _, _, err := execute(t, "pr", "route", "--pool", pool, "--output", out, "--labels", "docs", "--labels", "deep-review"); err != nil {
		t.Fatal(err)
	}
	if kv := outputLines(t, out); kv["route"] != "risk" {
		t.Errorf("--labels: %v", kv)
	}
}

func TestPrRoute_JSONReplacesTheLines(t *testing.T) {
	pool, _ := routeEnv(t)
	stdout, _, err := execute(t, "pr", "route", "--pool", pool, "--json")
	if err != nil {
		t.Fatal(err)
	}
	last := strings.TrimSpace(stdout[strings.LastIndex(strings.TrimSpace(stdout), "\n")+1:])
	var got map[string]any
	if err := json.Unmarshal([]byte(last), &got); err != nil {
		t.Fatalf("last line %q: %v", last, err)
	}
	if got["first"] != "nan" || got["pool"] != pool || got["dotf_version"] != "dev" {
		t.Errorf("json = %v", got)
	}
	if strings.Contains(stdout, "first=nan") {
		t.Errorf("--json still printed the lines: %q", stdout)
	}
}

func TestPrRoute_EmbeddedNamesItsPool(t *testing.T) {
	routeEnv(t)
	stdout, stderr, err := execute(t, "pr", "route", "--pool", "embedded")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if !strings.Contains(stdout, "pool: embedded") {
		t.Errorf("stdout = %q", stdout)
	}
}

// The exit contract: 0 routed, 1 refused, 2 usage.
func TestPrRoute_ExitCodes(t *testing.T) {
	pool, dir := routeEnv(t)
	missing := filepath.Join(dir, "missing.json")
	cases := []struct {
		name string
		args []string
		env  map[string]string
		code int
		out  string // in stdout
		err  string // in stderr
	}{
		{"no pool source", []string{"pr", "route"}, nil, 2, "", "--pool"},
		{"an unreadable pool", []string{"pr", "route", "--pool", missing}, nil, 2, "", "cannot read a pool"},
		{"a bad draw point", []string{"pr", "route", "--pool", pool}, map[string]string{"PR_AGENT_DRAW": "10"}, 2, "", "outside the total weight"},
		{"a size that is no number", []string{"pr", "route", "--pool", pool, "--changed-lines", "-4"}, nil, 2, "", "--changed-lines"},
		{"an unknown override", []string{"pr", "route", "--pool", pool}, map[string]string{"PR_AGENT_PROVIDER": "nan_only"}, 1, "::error::PR_AGENT_PROVIDER is nan_only", ""},
		{"a preflight that broke", []string{"pr", "route", "--pool", pool}, map[string]string{"NAN_PREFLIGHT_EXIT": "2"}, 1, "::error::The NaN preflight failed", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for k, v := range c.env {
				t.Setenv(k, v)
			}
			out := filepath.Join(t.TempDir(), "o")
			stdout, stderr, err := execute(t, append(c.args, "--output", out)...)
			if got := ExitCode(err); got != c.code {
				t.Fatalf("exit %d (%v), want %d", got, err, c.code)
			}
			if !strings.Contains(stdout, c.out) || !strings.Contains(stderr, c.err) {
				t.Errorf("stdout %q stderr %q", stdout, stderr)
			}
			if raw, _ := os.ReadFile(out); len(raw) != 0 {
				t.Errorf("a failed route wrote outputs: %q", raw)
			}
		})
	}
}

func TestPrRoute_PrintsTheAllowlistWithoutAPool(t *testing.T) {
	stdout, _, err := execute(t, "pr", "route", "--print-allowlist")
	if err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatalf("%q: %v", stdout, err)
	}
	if len(rows) < 2 || rows[0]["model"] != "anthropic/claude-haiku-5-5" || rows[0]["ceiling_usd"] != 0.25 ||
		rows[1]["input_usd_per_mtok"] != 2.0 {
		t.Errorf("rows = %v", rows)
	}
}
