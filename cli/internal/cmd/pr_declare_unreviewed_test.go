package cmd

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func declareEnv(t *testing.T, labels string) *[]string {
	t.Helper()
	for k, v := range map[string]string{"GITHUB_REPOSITORY": "o/r", "PR_NUMBER": "7", "DEFAULT_BRANCH": "main"} {
		t.Setenv(k, v)
	}
	registry := base64.StdEncoding.EncodeToString([]byte(`{"escape":{"label":"l","section":"## s"}}`))
	var calls []string
	old := prDeclareGH
	prDeclareGH = func(_ context.Context, args ...string) ([]byte, []byte, error) {
		joined := strings.Join(args, " ")
		calls = append(calls, joined)
		switch {
		case strings.Contains(joined, "-X "):
			return nil, nil, nil
		case strings.Contains(joined, "/contents/"):
			return []byte(registry), nil, nil
		case strings.HasSuffix(args[1], "/pulls/7"):
			return []byte(`{"body":"x"}`), nil, nil
		case strings.HasSuffix(args[1], "/labels"):
			return []byte(labels), nil, nil
		}
		return nil, []byte("gh: Bad Gateway (HTTP 502)\n"), errors.New("exit status 1")
	}
	t.Cleanup(func() { prDeclareGH = old })
	return &calls
}

func TestPrDeclareUnreviewed_EmptyDiffDeclares(t *testing.T) {
	calls := declareEnv(t, `[{"name":"l"}]`)
	stdout, stderr, err := execute(t, "pr", "declare-unreviewed", "--diff", "empty")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if !strings.Contains(stdout, "PR #7") {
		t.Errorf("stdout: %q", stdout)
	}
	if !strings.Contains(strings.Join(*calls, "\n"), "-X POST repos/o/r/issues/7/labels") {
		t.Errorf("calls: %q", *calls)
	}
}

func TestPrDeclareUnreviewed_ReviewableClears(t *testing.T) {
	calls := declareEnv(t, `[{"name":"l"}]`)
	if _, stderr, err := execute(t, "pr", "declare-unreviewed", "--diff", "reviewable"); err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if !strings.Contains(strings.Join(*calls, "\n"), "-X DELETE repos/o/r/issues/7/labels/l") {
		t.Errorf("calls: %q", *calls)
	}
}

func TestPrDeclareUnreviewed_UnreadIsANoOp(t *testing.T) {
	calls := declareEnv(t, `[]`)
	if _, stderr, err := execute(t, "pr", "declare-unreviewed", "--diff", "unread"); err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if len(*calls) != 0 {
		t.Errorf("calls: %q", *calls)
	}
}

func TestPrDeclareUnreviewed_UsageErrorsExit2(t *testing.T) {
	for name, tc := range map[string]struct {
		args []string
		env  string
	}{
		"no diff":                        {args: nil},
		"a skipped step leaves it empty": {args: []string{"--diff", ""}},
		"unknown":                        {args: []string{"--diff", "true"}},
		"an argument":                    {args: []string{"--diff", "empty", "7"}},
		"a bad flag":                     {args: []string{"--nope"}},
		"no env":                         {args: []string{"--diff", "empty"}, env: "GITHUB_REPOSITORY"},
	} {
		t.Run(name, func(t *testing.T) {
			calls := declareEnv(t, `[]`)
			if tc.env != "" {
				t.Setenv(tc.env, "")
			}
			_, _, err := execute(t, append([]string{"pr", "declare-unreviewed"}, tc.args...)...)
			if code := ExitCode(err); code != 2 {
				t.Errorf("exit %d (%v)", code, err)
			}
			if len(*calls) != 0 {
				t.Errorf("calls: %q", *calls)
			}
		})
	}
}
