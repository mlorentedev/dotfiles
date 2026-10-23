package cmd

import (
	"context"
	"encoding/base64"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

const guardReviewComment = `[{"user":{"login":"github-actions[bot]"},"updated_at":"2026-10-07T10:00:00Z","body":"## PR Reviewer Guide"}]`

// guardEnv sets the workflow's environment for `pr guard` and answers its gh
// calls: the registry declares PR-Agent's marker, and the comment listing is
// comments. It records each call's path.
func guardEnv(t *testing.T, comments string) *[]string {
	t.Helper()
	for k, v := range map[string]string{"GITHUB_REPOSITORY": "o/r", "PR_NUMBER": "7", "BASE_REF": "main",
		"STARTED": "2026-10-07T09:00:00Z", "HEAD_SHA": "abc123", "ATTEMPTS": "skipped hk\nsuccess m\nskipped hk\n",
		"ROUTE_NOTE": "", "ROUTE_OUTCOME": "success", "GUARD_RETRY_SECONDS": "0"} {
		t.Setenv(k, v)
	}
	registry := base64.StdEncoding.EncodeToString(
		[]byte(`{"reviewers":[{"login":"github-actions","review_markers":["PR Reviewer Guide"]}]}`))
	var calls []string
	old := prGuardGH
	prGuardGH = func(_ context.Context, args ...string) ([]byte, []byte, error) {
		calls = append(calls, args[1])
		switch {
		case strings.Contains(args[1], "review-attestation.json"):
			return []byte(registry + "\n"), nil, nil
		case strings.HasSuffix(args[1], "/comments"):
			if comments == "fail" {
				return nil, []byte("gh: Bad Gateway (HTTP 502)\n"), errors.New("exit status 1")
			}
			return []byte(comments + "\n"), nil, nil
		}
		return nil, []byte("unexpected call"), errors.New("exit status 99")
	}
	t.Cleanup(func() { prGuardGH = old })
	return &calls
}

func TestPrGuard_PassesWhenThisRunPublishedAReview(t *testing.T) {
	calls := guardEnv(t, guardReviewComment)
	stdout, stderr, err := execute(t, "pr", "guard")
	if err != nil {
		t.Fatalf("pr guard: %v\n%s", err, stderr)
	}
	if !strings.Contains(stdout, `review published (markers: ["PR Reviewer Guide"]) on m`) {
		t.Errorf("stdout = %q", stdout)
	}
	if want := "repos/o/r/contents/harness/review-attestation.json?ref=main"; len(*calls) == 0 || (*calls)[0] != want {
		t.Errorf("calls = %v, want the registry at main first", *calls)
	}
}

func TestPrGuard_ExitsOneWhenNoReviewWasPublished(t *testing.T) {
	guardEnv(t, "[]")
	stdout, _, err := execute(t, "pr", "guard")
	if ExitCode(err) != 1 {
		t.Fatalf("exit code = %d (%v), want 1", ExitCode(err), err)
	}
	if !strings.Contains(stdout, "::error::PR-Agent reported success but published no review") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestPrGuard_ProbeAppendsItsAnswerAndExitsZero(t *testing.T) {
	for comments, want := range map[string]string{
		guardReviewComment: "published=true\n",
		"[]":               "published=false\n",
		"fail":             "published=unknown\n",
	} {
		guardEnv(t, comments)
		out := filepath.Join(t.TempDir(), "github_output")
		stdout, stderr, err := execute(t, "pr", "guard", "--probe", "--output", out)
		if err != nil {
			t.Fatalf("pr guard --probe: %v\n%s", err, stderr)
		}
		if got := readFile(t, out); got != want {
			t.Errorf("comments %q: output = %q, want %q", comments, got, want)
		}
		if comments == "fail" && !strings.Contains(stdout, "::warning::gh api repos/o/r/issues/7/comments: gh: Bad Gateway (HTTP 502)") {
			t.Errorf("stdout = %q", stdout)
		}
	}
}

func TestPrGuard_AUsageErrorExitsTwo(t *testing.T) {
	guardEnv(t, "[]")
	unwritable := filepath.Join(t.TempDir(), "missing", "github_output")
	for name, args := range map[string][]string{
		"--probe without --output":  {"pr", "guard", "--probe"},
		"an output it cannot write": {"pr", "guard", "--probe", "--output", unwritable},
		"an argument":               {"pr", "guard", "extra"},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := execute(t, args...)
			if ExitCode(err) != 2 {
				t.Errorf("exit code = %d (%v), want 2", ExitCode(err), err)
			}
		})
	}
}
