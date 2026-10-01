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

// canaryServer answers 200 for every model except refuse, which gets 401. It
// records which models were asked.
func canaryServer(t *testing.T, refuse string) (*httptest.Server, *[]string) {
	t.Helper()
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		asked = append(asked, req.Model)
		if req.Model == refuse {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &asked
}

// TestHarnessCanaryAgainstTheShippedRegistry drives the command end to end
// against the map and pin registry the repository actually ships, with a stub
// transport in place of NaN.
func TestHarnessCanaryAgainstTheShippedRegistry(t *testing.T) {
	root := repoRootForTest(t)
	t.Setenv("HOME", t.TempDir()) // no deployed pin site: a CI runner's view
	t.Setenv("NAN_API_KEY", "test-key")

	srv, asked := canaryServer(t, "")
	t.Setenv("NAN_API_BASE", srv.URL+"/v1")
	report := filepath.Join(t.TempDir(), "canary.md")

	out, _, err := executeStdin(t, "", "harness", "canary", "--repo-root", root, "--report", report)
	if err != nil {
		t.Fatalf("every model answered, want exit 0: %v\n%s", err, out)
	}
	if len(*asked) == 0 {
		t.Fatal("no model was probed")
	}
	written, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != out {
		t.Error("the report file and stdout differ")
	}
	if !strings.Contains(out, "Not read:") {
		t.Errorf("deployed sites absent on a runner must be named as not read:\n%s", out)
	}

	// The first model the shipped registry binds now refuses: exit 1, and its
	// row names the files that bind it.
	first := (*asked)[0]
	srv2, _ := canaryServer(t, first)
	t.Setenv("NAN_API_BASE", srv2.URL+"/v1")
	out, _, err = executeStdin(t, "", "harness", "canary", "--repo-root", root)
	if got := ExitCode(err); got != canaryExitModels {
		t.Fatalf("a refused model: exit %d, want %d\n%s", got, canaryExitModels, out)
	}
	if !strings.Contains(out, "`"+first+"` | refused (HTTP 401)") || !strings.Contains(out, "`harness/model-map.json`") {
		t.Errorf("the refused row names the model, its class and the files:\n%s", out)
	}
}

func TestHarnessCanaryWithoutAKeyCannotRun(t *testing.T) {
	t.Setenv("NAN_API_KEY", "")
	out, _, err := executeStdin(t, "", "harness", "canary", "--repo-root", repoRootForTest(t))
	if got := ExitCode(err); got != canaryExitSetup {
		t.Fatalf("no key: exit %d, want %d (a canary that could not look says nothing about the models)\n%s", got, canaryExitSetup, out)
	}
}
