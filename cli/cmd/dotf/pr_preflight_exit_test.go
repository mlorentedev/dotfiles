package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/cmd"
)

// `dotf pr preflight` reports a setup error as 2 and "no model answered" as 1
// through the process's real exit mapping, which is how the workflow tells a
// caller mistake from a NaN outage.
func TestRunExitsWithTheCodePrPreflightCarries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) }))
	t.Cleanup(srv.Close)
	t.Setenv("NAN_API_BASE", srv.URL)
	t.Setenv("PREFLIGHT_TIMEOUT", "2")
	t.Setenv("GITHUB_STEP_SUMMARY", "")
	for name, tc := range map[string]struct {
		args []string
		key  string
		code int
		err  string
	}{
		"no key":          {[]string{"pr", "preflight", "--model", "openai/a", "--fallbacks", "[]"}, "", 2, "NAN_API_KEY is not set"},
		"no model":        {[]string{"pr", "preflight", "--fallbacks", "[]"}, "k", 2, "usage:"},
		"none answers":    {[]string{"pr", "preflight", "--model", "openai/a", "--fallbacks", "[]"}, "k", 1, ""},
		"an unknown flag": {[]string{"pr", "preflight", "--no-such-flag"}, "k", 1, "unknown flag"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("NAN_API_KEY", tc.key)
			rootCmd := cmd.New("dev", "")
			rootCmd.SetArgs(tc.args)
			var stdout bytes.Buffer
			rootCmd.SetOut(&stdout)
			var stderr bytes.Buffer
			if code := run(rootCmd, &stderr); code != tc.code {
				t.Errorf("exit code = %d, want %d; stderr: %s", code, tc.code, stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.err) {
				t.Errorf("stderr = %q, want it to mention %q", stderr.String(), tc.err)
			}
			if tc.err == "" && stderr.Len() != 0 {
				t.Errorf("stderr = %q, want none", stderr.String())
			}
		})
	}
}
