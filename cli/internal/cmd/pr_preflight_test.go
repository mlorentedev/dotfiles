package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

const preflightTestKey = "sk-nan-test-not-a-real-key"

// preflightEnv sets the workflow's inputs for one run of `pr preflight` against
// a local server that answers each model by the code it is given (404 for any
// other), and returns a directory for the output and the summary.
func preflightEnv(t *testing.T, codes map[string]int) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 512)
		n, _ := r.Body.Read(buf)
		code := http.StatusNotFound
		for model, c := range codes {
			if strings.Contains(string(buf[:n]), `"model":"`+model+`"`) {
				code = c
			}
		}
		w.WriteHeader(code)
		_, _ = w.Write([]byte("{}"))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("NAN_API_KEY", preflightTestKey)
	t.Setenv("NAN_API_BASE", srv.URL)
	t.Setenv("PREFLIGHT_TIMEOUT", "2")
	t.Setenv("GITHUB_STEP_SUMMARY", "")
	return t.TempDir()
}

var chain = []string{"--model", "openai/mimo-v2.6-flash", "--fallbacks", `["openai/deepseek-v4-flash"]`}

func preflightArgs(extra ...string) []string {
	return append(append([]string{"pr", "preflight"}, chain...), extra...)
}

func TestPrPreflight_WritesTheOutputsTheWorkflowReadsAndTheSummary(t *testing.T) {
	dir := preflightEnv(t, map[string]int{"mimo-v2.6-flash": 401, "deepseek-v4-flash": 200})
	out, summary := filepath.Join(dir, "github_output"), filepath.Join(dir, "summary")
	t.Setenv("GITHUB_STEP_SUMMARY", summary)

	stdout, stderr, err := execute(t, preflightArgs("--output", out)...)
	if err != nil {
		t.Fatalf("pr preflight: %v\n%s", err, stderr)
	}
	if got := readFile(t, out); got != "model=openai/deepseek-v4-flash\nfallbacks=[]\n" {
		t.Errorf("output = %q", got)
	}
	if !strings.Contains(stdout, "::warning::PR-Agent model openai/mimo-v2.6-flash answered HTTP 401") || strings.Contains(stdout, "model=") {
		t.Errorf("stdout = %q", stdout)
	}
	sum := readFile(t, summary)
	if !strings.Contains(sum, "| `openai/mimo-v2.6-flash` | HTTP 401 |") || !strings.Contains(sum, "Reviewing with `openai/deepseek-v4-flash`.") {
		t.Errorf("summary = %q", sum)
	}
	for name, text := range map[string]string{"stdout": stdout, "stderr": stderr, "output": readFile(t, out), "summary": sum} {
		if strings.Contains(text, preflightTestKey) {
			t.Errorf("%s carries the key", name)
		}
	}
}

// --output appends, as `>> $GITHUB_OUTPUT` does: the step's other outputs stay.
func TestPrPreflight_AppendsToTheOutputFile(t *testing.T) {
	dir := preflightEnv(t, map[string]int{"mimo-v2.6-flash": 200, "deepseek-v4-flash": 200})
	out := filepath.Join(dir, "github_output")
	if err := os.WriteFile(out, []byte("earlier=kept\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := execute(t, preflightArgs("--output", out)...); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, out); got != "earlier=kept\nmodel=openai/mimo-v2.6-flash\nfallbacks=[\"openai/deepseek-v4-flash\"]\n" {
		t.Errorf("output = %q", got)
	}
}

func TestPrPreflight_WithNoOutputFileTheLinesGoToStdoutAfterTheAnnotations(t *testing.T) {
	preflightEnv(t, map[string]int{"mimo-v2.6-flash": 503, "deepseek-v4-flash": 200})
	stdout, _, err := execute(t, preflightArgs()...)
	if err != nil {
		t.Fatal(err)
	}
	w, m := strings.Index(stdout, "::warning::"), strings.Index(stdout, "model=openai/deepseek-v4-flash\n")
	if w < 0 || m < w {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestPrPreflight_JSONWritesOneObjectInsteadOfTheLines(t *testing.T) {
	dir := preflightEnv(t, map[string]int{"mimo-v2.6-flash": 200, "deepseek-v4-flash": 429})
	out := filepath.Join(dir, "result.json")
	if _, _, err := execute(t, preflightArgs("--output", out, "--json")...); err != nil {
		t.Fatal(err)
	}
	want := `{"model":"openai/mimo-v2.6-flash","fallbacks":[],"probes":[{"model":"openai/mimo-v2.6-flash","status":"200"},{"model":"openai/deepseek-v4-flash","status":"429"}]}` + "\n"
	if got := readFile(t, out); got != want {
		t.Errorf("output = %q\nwant     %q", got, want)
	}
}

// Exit 1 says NaN left the pool: the summary is written, the outputs are not,
// and the reason was already printed as a warning.
func TestPrPreflight_WhenNoModelAnswersItExitsOneWithTheSummaryAndNoOutputs(t *testing.T) {
	dir := preflightEnv(t, map[string]int{"mimo-v2.6-flash": 401, "deepseek-v4-flash": 429})
	out, summary := filepath.Join(dir, "github_output"), filepath.Join(dir, "summary")
	t.Setenv("GITHUB_STEP_SUMMARY", summary)

	stdout, stderr, err := execute(t, preflightArgs("--output", out)...)
	if ExitCode(err) != 1 {
		t.Fatalf("exit code = %d (%v), want 1", ExitCode(err), err)
	}
	if !strings.Contains(stdout, "::warning::No model in PR-Agent's declared NaN chain answered") || stderr != "" {
		t.Errorf("stdout = %q, stderr = %q", stdout, stderr)
	}
	if _, err := os.Stat(out); err == nil {
		t.Errorf("outputs were written: %q", readFile(t, out))
	}
	if !strings.Contains(readFile(t, summary), "No declared model answered") {
		t.Errorf("summary = %q", readFile(t, summary))
	}
}

// A usage error is 2, not 1: the workflow reads 1 as "NaN is down" and would
// move every review onto the paid provider for a typo.
func TestPrPreflight_ASetupErrorExitsTwo(t *testing.T) {
	dir := preflightEnv(t, map[string]int{"mimo-v2.6-flash": 200})
	for name, tc := range map[string]struct {
		args []string
		key  string
		want string
	}{
		"no key":            {preflightArgs(), "", "NAN_API_KEY is not set"},
		"no model":          {[]string{"pr", "preflight", "--fallbacks", "[]"}, preflightTestKey, "usage:"},
		"no fallbacks":      {[]string{"pr", "preflight", "--model", "openai/x"}, preflightTestKey, "usage:"},
		"bad fallbacks":     {[]string{"pr", "preflight", "--model", "openai/x", "--fallbacks", "not json"}, preflightTestKey, "must be a JSON array"},
		"unwritable output": {preflightArgs("--output", filepath.Join(dir, "missing", "out")), preflightTestKey, "missing"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("NAN_API_KEY", tc.key)
			_, stderr, err := execute(t, tc.args...)
			if ExitCode(err) != 2 {
				t.Errorf("exit code = %d (%v), want 2", ExitCode(err), err)
			}
			if !strings.Contains(stderr, tc.want) {
				t.Errorf("stderr = %q, want it to mention %q", stderr, tc.want)
			}
			if strings.Contains(stderr, preflightTestKey) {
				t.Errorf("stderr carries the key: %q", stderr)
			}
		})
	}
}

// The key is read from NAN_API_KEY and from nowhere else: a flag would put it on
// argv, where any process on the runner can read it.
func TestPrPreflight_HasNoFlagThatTakesTheKey(t *testing.T) {
	c := newPrPreflightCmd()
	c.Flags().VisitAll(func(f *pflag.Flag) {
		if strings.Contains(strings.ToLower(f.Name), "key") || strings.Contains(strings.ToLower(f.Name), "token") {
			t.Errorf("flag --%s could carry the key on argv", f.Name)
		}
	})
}
