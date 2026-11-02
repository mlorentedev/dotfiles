package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const reviewableRef = "981344be2755e303df501e18e3f60b848a188a80"

func reviewableEnv(t *testing.T, pull string) {
	t.Helper()
	for k, v := range map[string]string{"GITHUB_REPOSITORY": "o/r", "PR_NUMBER": "7",
		"GITHUB_EVENT_NAME": "pull_request", "DEFAULT_BRANCH": "main"} {
		t.Setenv(k, v)
	}
	old := prReviewableGH
	prReviewableGH = func(_ context.Context, args ...string) ([]byte, []byte, error) {
		switch {
		case strings.HasSuffix(args[1], "/pulls/7"):
			return []byte(pull), nil, nil
		case strings.HasSuffix(args[1], "/pulls/7/files"):
			return []byte(`[{"filename":"cli/main.go"}]`), nil, nil
		}
		return nil, []byte("gh: Not Found (HTTP 404)\n"), errors.New("exit status 1")
	}
	t.Cleanup(func() { prReviewableGH = old })
}

const reviewableOpenPR = `{"state":"open","head":{"ref":"feat/x"}}`

func TestPrReviewable_PrintsItsAnswer(t *testing.T) {
	reviewableEnv(t, reviewableOpenPR)
	stdout, stderr, err := execute(t, "pr", "reviewable", "--audited-ref", reviewableRef)
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if !strings.HasSuffix(stdout, "reviewable=true\n") {
		t.Errorf("stdout: %q", stdout)
	}
}

func TestPrReviewable_AppendsToOutputAndPassesThePrefixes(t *testing.T) {
	reviewableEnv(t, reviewableOpenPR)
	out := filepath.Join(t.TempDir(), "github_output")
	if err := os.WriteFile(out, []byte("earlier=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, stderr, err := execute(t, "pr", "reviewable", "--audited-ref", reviewableRef,
		"--skip-branch-prefixes", "feat/", "--output", out); err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "earlier=1\nreviewable=false\n" {
		t.Errorf("output: %q", got)
	}
}

func TestPrReviewable_ReadsTheAuditedCommitFromTheContract(t *testing.T) {
	reviewableEnv(t, reviewableOpenPR)
	contract := filepath.Join(t.TempDir(), "contract.json")
	if err := os.WriteFile(contract, []byte(`{"audited_ref":"`+reviewableRef+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, stderr, err := execute(t, "pr", "reviewable", "--contract", contract); err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
}

func TestPrReviewable_AnUnreadableContractExitsOneAndWritesNoOutput(t *testing.T) {
	reviewableEnv(t, reviewableOpenPR)
	out := filepath.Join(t.TempDir(), "github_output")
	_, _, err := execute(t, "pr", "reviewable", "--contract", filepath.Join(t.TempDir(), "absent"), "--output", out)
	if code := ExitCode(err); code != 1 {
		t.Errorf("exit = %d (%v), want 1", code, err)
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Errorf("output written: %v", statErr)
	}
}

func TestPrReviewable_UsageErrorsExitTwo(t *testing.T) {
	cases := map[string]struct {
		args  []string
		unset string
	}{
		"neither source":    {args: nil},
		"both sources":      {args: []string{"--contract", "c", "--audited-ref", reviewableRef}},
		"an argument":       {args: []string{"--audited-ref", reviewableRef, "extra"}},
		"an unknown flag":   {args: []string{"--nope"}},
		"no repository":     {args: []string{"--audited-ref", reviewableRef}, unset: "GITHUB_REPOSITORY"},
		"no PR":             {args: []string{"--audited-ref", reviewableRef}, unset: "PR_NUMBER"},
		"no event":          {args: []string{"--audited-ref", reviewableRef}, unset: "GITHUB_EVENT_NAME"},
		"no default branch": {args: []string{"--audited-ref", reviewableRef}, unset: "DEFAULT_BRANCH"},
		"unwritable output": {args: []string{"--audited-ref", reviewableRef, "--output", "/nonexistent/dir/out"}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			reviewableEnv(t, reviewableOpenPR)
			if c.unset != "" {
				t.Setenv(c.unset, "")
			}
			_, _, err := execute(t, append([]string{"pr", "reviewable"}, c.args...)...)
			if code := ExitCode(err); code != 2 {
				t.Errorf("exit = %d (%v), want 2", code, err)
			}
		})
	}
}
