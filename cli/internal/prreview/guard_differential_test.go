package prreview

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Until scripts/pr-agent-publish-guard.sh is deleted, it is the specification:
// the same answers from one stub gh, through the script and through Guard or
// ProbePublished, must give the same annotations, the same probe answer, the
// same API calls in the same order and the same exit class. The port reaches gh
// through execGH, the path the command takes, so the stub is on PATH for both.
// The script goes when the workflow switches to the command, and this test goes
// with it.

// guardStub is the stub gh of tests/pr-agent-publish-guard.bats, plus one line
// that logs each call's full arguments to $CALLS.args: the bats stub answers by
// path alone, so without it a port that changed a --jq filter would still match.
const guardStub = `#!/usr/bin/env bash
api_path="$2"
case "$api_path" in
    *contents/harness/review-attestation.json?ref=main) key=BASE ;;
    *contents/harness/review-attestation.json?ref=*) key=HEAD ;;
    */pulls/*) key=PULL ;;
    */comments) key=COMMENTS ;;
    *) echo "stub gh: unexpected call: $*" >&2; exit 99 ;;
esac
echo "$key" >> "$CALLS"
echo "$*" >> "$CALLS.args"
eval "answer=\${STUB_$key:-}"
fail() {
    echo '{"message":"stub failure body"}'
    case "$1" in
        404) echo "gh: Not Found (HTTP 404)" >&2 ;;
        *) echo "gh: Gateway Timeout (HTTP $1)" >&2 ;;
    esac
    exit 1
}
case "$answer" in
    fail:*) fail "${answer#fail:}" ;;
    flaky:*)
        if [ "$(grep -c "^$key\$" "$CALLS")" -eq 1 ]; then fail "${answer#flaky:}"; fi
        answer=$(eval "printf '%s' \"\${STUB_${key}_OK}\"")
        ;;
esac
printf '%s\n' "$answer"
`

type guardCase struct {
	name    string
	answers map[string]string
	edit    func(*GuardInputs)
	probe   bool
}

func guardCases() []guardCase {
	old := `[{"user":{"login":"github-actions[bot]"},"updated_at":"2026-10-07T08:00:00Z","body":"## PR Reviewer Guide"}]`
	pasted := `[{"user":{"login":"mallory"},"updated_at":"2026-10-07T10:00:00Z","body":"## PR Reviewer Guide"}]`
	suggestions := `[{"user":{"login":"github-actions[bot]"},"updated_at":"2026-10-07T10:00:00Z","body":"## PR Code Suggestions"}]`
	headSuggestions := b64(`{"reviewers":[{"login":"github-actions","review_markers":["PR Code Suggestions"]}]}`)
	atStart := `[{"user":{"login":"github-actions[bot]"},"updated_at":"2026-10-07T09:00:00Z","body":"## PR Reviewer Guide"}]`
	reviewed := map[string]string{"BASE": guardRegistry, "COMMENTS": guardReview}
	empty := map[string]string{"BASE": guardRegistry, "COMMENTS": "[]"}
	tried := func(a string) func(*GuardInputs) { return func(in *GuardInputs) { in.Attempts = a } }
	cases := []guardCase{
		{name: "registry keeps failing", answers: map[string]string{"BASE": "fail:504", "HEAD": "fail:504"}},
		{name: "registry fails once", answers: map[string]string{"BASE": "flaky:502", "BASE_OK": guardRegistry, "COMMENTS": guardReview}},
		{name: "registry without an entry", answers: map[string]string{"BASE": guardNoEntry, "HEAD": guardNoEntry}},
		{name: "base 404, head fallback", answers: map[string]string{"BASE": "fail:404", "HEAD": guardRegistry, "COMMENTS": guardReview}},
		{name: "head lookup fails", answers: map[string]string{"BASE": guardNoEntry, "PULL": "fail:503"},
			edit: func(in *GuardInputs) { in.HeadSHA = "" }},
		{name: "head lookup answers", answers: map[string]string{"BASE": guardNoEntry, "PULL": "abc123", "HEAD": guardRegistry, "COMMENTS": guardReview},
			edit: func(in *GuardInputs) { in.HeadSHA = "" }},
		{name: "listing keeps failing", answers: map[string]string{"BASE": guardRegistry, "COMMENTS": "fail:500"}},
		{name: "listing answers 404", answers: map[string]string{"BASE": guardRegistry, "COMMENTS": "fail:404"}},
		{name: "listing is not JSON", answers: map[string]string{"BASE": guardRegistry, "COMMENTS": "<html>bad gateway</html>"}},
		{name: "listing of two pages", answers: map[string]string{"BASE": guardRegistry, "COMMENTS": "[]\n" + guardReview}},
		{name: "no review comment", answers: empty},
		{name: "a review older than the run", answers: map[string]string{"BASE": guardRegistry, "COMMENTS": old}},
		// The start stamp itself counts: the comparison is >=.
		{name: "a review stamped at the run's start", answers: map[string]string{"BASE": guardRegistry, "COMMENTS": atStart}},
		{name: "a pasted heading", answers: map[string]string{"BASE": guardRegistry, "COMMENTS": pasted}},
		{name: "head fallback ignores the head's marker",
			answers: map[string]string{"BASE": "fail:404", "HEAD": headSuggestions, "COMMENTS": suggestions}},
		{name: "an Anthropic attempt published nothing", answers: empty, edit: tried("success anthropic/claude-haiku-5-5\nskipped m\n")},
		{name: "the review names its attempt", answers: reviewed, edit: tried(attempts("skipped", "failure", "success"))},
		{name: "a failed second attempt", answers: empty, edit: tried(attempts("skipped", "failure", "failure"))},
		{name: "Anthropic then NaN cancelled", answers: empty, edit: tried(attempts("failure", "cancelled", "skipped"))},
		{name: "Anthropic alone", answers: reviewed, edit: tried(attempts("success", "skipped", "skipped"))},
		{name: "a failure repeats the route note", edit: func(in *GuardInputs) {
			in.Attempts, in.RouteNote = attempts("skipped", "failure", "skipped"), "PR_AGENT_ANTHROPIC_API_KEY is not set"
		}},
		{name: "no attempt", edit: tried(attempts("skipped", "skipped", "skipped"))},
		{name: "no attempt, the draw failed", edit: func(in *GuardInputs) {
			in.Attempts, in.RouteOutcome = attempts("skipped", "skipped", "skipped"), "failure"
		}},
		{name: "nothing evaluated", edit: tried(" hk\n m\n hk\n")},
		{name: "no attempt and an empty model", edit: tried("success\n")},
	}
	// Every case that reaches the API also runs as the probe.
	for _, c := range cases {
		if c.answers != nil {
			p := c
			p.name, p.probe = "probe: "+c.name, true
			cases = append(cases, p)
		}
	}
	return cases
}

// guardEnviron is the case's environment, as the workflow gives it to the script.
func guardEnviron(in GuardInputs, c guardCase, stubDir, calls string) []string {
	env := []string{"PATH=" + stubDir + string(os.PathListSeparator) + os.Getenv("PATH"), "CALLS=" + calls,
		"GUARD_RETRY_SECONDS=0", "GITHUB_REPOSITORY=" + in.Repo, "PR_NUMBER=" + in.PR, "BASE_REF=" + in.BaseRef,
		"STARTED=" + in.Started, "HEAD_SHA=" + in.HeadSHA, "ATTEMPTS=" + in.Attempts, "ROUTE_NOTE=" + in.RouteNote,
		"ROUTE_OUTCOME=" + in.RouteOutcome}
	for k, v := range c.answers {
		env = append(env, "STUB_"+k+"="+v)
	}
	return env
}

type guardResult struct {
	code          int
	stdout, probe string
	calls         string
}

func TestGuardMatchesTheScript(t *testing.T) {
	script := requireScript(t, "pr-agent-publish-guard.sh", "bash", "jq", "base64")
	stubDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(stubDir, "gh"), []byte(guardStub), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, c := range guardCases() {
		t.Run(c.name, func(t *testing.T) {
			in := guardInputs()
			in.RetrySeconds = "0"
			if c.edit != nil {
				c.edit(&in)
			}
			dir := t.TempDir()
			calls := filepath.Join(dir, "calls")
			env := guardEnviron(in, c, stubDir, calls)

			want := runGuardScript(t, script, env, c.probe, calls, filepath.Join(dir, "probe.out"))

			// The port, through the same stub on PATH.
			resetCalls(t, calls)
			for _, kv := range env {
				k, v, _ := strings.Cut(kv, "=")
				t.Setenv(k, v)
			}
			var out bytes.Buffer
			got := guardResult{}
			if c.probe {
				got.probe = "published=" + ProbePublished(t.Context(), in, GuardEnv{Out: &out}) + "\n"
			} else if err := Guard(t.Context(), in, GuardEnv{Out: &out}); errors.Is(err, ErrNotPublished) {
				got.code = 1
			} else if err != nil {
				t.Fatalf("Guard: %v", err)
			}
			got.stdout = out.String()
			got.calls = readCalls(calls)

			if got != want {
				t.Errorf("the port differs from the script\n--- script ---\n%+v\n--- port ---\n%+v", want, got)
			}
		})
	}
}

func runGuardScript(t *testing.T, script string, env []string, probe bool, calls, probeOut string) guardResult {
	t.Helper()
	resetCalls(t, calls)
	args := []string{script}
	if probe {
		args = append(args, "--probe", "--output", probeOut)
	}
	cmd := exec.CommandContext(t.Context(), "bash", args...)
	cmd.Env = env
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var res guardResult
	var ee *exec.ExitError
	if err := cmd.Run(); errors.As(err, &ee) {
		res.code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("running the script: %v", err)
	}
	res.stdout = stdout.String()
	if raw, err := os.ReadFile(probeOut); err == nil {
		res.probe = string(raw)
	}
	res.calls = readCalls(calls)
	return res
}

func resetCalls(t *testing.T, calls string) {
	t.Helper()
	for _, f := range []string{calls, calls + ".args"} {
		if err := os.WriteFile(f, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// readCalls is the subject of every call, then every call's full arguments.
func readCalls(calls string) string {
	keys, _ := os.ReadFile(calls)
	args, _ := os.ReadFile(calls + ".args")
	return string(keys) + "--\n" + string(args)
}
