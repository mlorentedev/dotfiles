package harness

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The steps below mirror .github/workflows/pr-agent.yml as it stood before
// AI-054: the two Anthropic attempts declare ai_timeout and an output cap, the
// NaN attempt declares neither and lowers retry_same_model_on_timeout to false.
const (
	budgetTomlPreFix = `max_model_tokens = 200000
custom_model_max_tokens = 200000
fallback_models = ["openai/deepseek-v4-flash", "openai/glm5.3-flash"]
`

	budgetTomlFixed = `max_model_tokens = 120000
custom_model_max_tokens = 200000
fallback_models = ["openai/deepseek-v4-flash", "openai/glm5.3-flash"]
`

	budgetStepFirstPreFix = `      - name: PR-Agent on Anthropic, drawn first
        uses: docker://pragent/pr-agent@sha256:0000
        timeout-minutes: 8
        env:
          CONFIG__FALLBACK_MODELS: "[]"
          CONFIG__AI_TIMEOUT: "360"
          CONFIG__MAX_OUTPUT_TOKENS: "40000"
`

	budgetStepFirstFixed = `      - name: PR-Agent on Anthropic, drawn first
        uses: docker://pragent/pr-agent@sha256:0000
        timeout-minutes: 13
        env:
          CONFIG__FALLBACK_MODELS: "[]"
          CONFIG__AI_TIMEOUT: "360"
          CONFIG__MAX_OUTPUT_TOKENS: "40000"
`

	budgetStepNaNPreFix = `      - name: PR-Agent
        uses: docker://pragent/pr-agent@sha256:0000
        timeout-minutes: 12
        env:
          CONFIG__FALLBACK_MODELS: ${{ steps.route.outputs.nan_fallbacks }}
          CONFIG__CUSTOM_MODEL_MAX_TOKENS: "200000"
          CONFIG__RETRY_SAME_MODEL_ON_TIMEOUT: "false"
`

	budgetStepNaNFixed = `      - name: PR-Agent
        uses: docker://pragent/pr-agent@sha256:0000
        timeout-minutes: 12
        env:
          CONFIG__FALLBACK_MODELS: ${{ steps.route.outputs.nan_fallbacks }}
          CONFIG__CUSTOM_MODEL_MAX_TOKENS: "200000"
          CONFIG__AI_TIMEOUT: "120"
          CONFIG__RETRY_SAME_MODEL_ON_TIMEOUT: "false"
`

	budgetStepSecondPreFix = `      - name: PR-Agent on Anthropic, after NaN published nothing
        uses: docker://pragent/pr-agent@sha256:0000
        timeout-minutes: 8
        env:
          CONFIG__FALLBACK_MODELS: "[]"
          CONFIG__AI_TIMEOUT: "360"
          CONFIG__MAX_OUTPUT_TOKENS: "40000"
`

	budgetStepSecondFixed = `      - name: PR-Agent on Anthropic, after NaN published nothing
        uses: docker://pragent/pr-agent@sha256:0000
        timeout-minutes: 13
        env:
          CONFIG__FALLBACK_MODELS: "[]"
          CONFIG__AI_TIMEOUT: "360"
          CONFIG__MAX_OUTPUT_TOKENS: "40000"
`
)

// budgetRepo is a checkout holding only the two files the guard reads.
func budgetRepo(t *testing.T, toml, workflow string) string {
	t.Helper()
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, prAgentTomlFile), toml)
	writeFile(t, filepath.Join(repo, filepath.FromSlash(prAgentWorkflowFile)), budgetWorkflow(34, workflow))
	return repo
}

// budgetWorkflow wraps review steps in the job that runs them.
func budgetWorkflow(jobMinutes int, steps ...string) string {
	return "jobs:\n  review:\n    timeout-minutes: " + strconv.Itoa(jobMinutes) + "\n    steps:\n" + strings.Join(steps, "")
}

// budgetProblemText is the whole report as one string, for the assertions that
// ask what a line says rather than how many there are.
func budgetProblemText(t *testing.T, repoRoot string) (int, string) {
	t.Helper()
	found := problems(checkReviewBudget(repoRoot))
	return len(found), strings.Join(found, "\n")
}

func TestCheckReviewBudget_TheCurrentConfigurationFails(t *testing.T) {
	repo := budgetRepo(t, budgetTomlPreFix,
		budgetStepFirstPreFix+budgetStepNaNPreFix+budgetStepSecondPreFix)

	// Five: the file's clip, the NaN step's clip (it declares the window, so
	// it is measured on its own), two step ceilings and the undeclared ai_timeout.
	n, text := budgetProblemText(t, repo)

	if n != 5 {
		t.Fatalf("want 5 problems (two clips, two step ceilings, undeclared ai_timeout), got %d:\n%s", n, text)
	}
	for _, want := range []string{
		"the prompt clip in",
		"cannot afford its own attempts",
		"does not declare CONFIG__AI_TIMEOUT",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("report does not mention %q:\n%s", want, text)
		}
	}
}

func TestCheckReviewBudget_TheFixedConfigurationPasses(t *testing.T) {
	repo := budgetRepo(t, budgetTomlFixed,
		budgetStepFirstFixed+budgetStepNaNFixed+budgetStepSecondFixed)

	if n, text := budgetProblemText(t, repo); n != 0 {
		t.Fatalf("want no problems, got %d:\n%s", n, text)
	}
}

func TestCheckReviewBudget_TheGuardLeavesTheJobCeilingToTheBatsTest(t *testing.T) {
	// The job's arithmetic needs the probe seconds and the route's pair, which
	// this guard does not model; a first attempt at it here passed a job ceiling
	// one minute short of the truth. tests/pr-agent-config.bats owns that number.
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, prAgentTomlFile), budgetTomlFixed)
	writeFile(t, filepath.Join(repo, filepath.FromSlash(prAgentWorkflowFile)),
		budgetWorkflow(1, budgetStepFirstFixed+budgetStepNaNFixed+budgetStepSecondFixed))

	if n, text := budgetProblemText(t, repo); n != 0 {
		t.Fatalf("the guard must say nothing about the job ceiling, got %d:\n%s", n, text)
	}
}

func TestCheckReviewBudget_TheTokenClipNeedsRoomForTheAnswer(t *testing.T) {
	// A clip equal to the window clips nothing, and one that leaves less room
	// than the answer needs is the same defect wearing a different number.
	for _, toml := range []string{
		"max_model_tokens = 200000\ncustom_model_max_tokens = 200000\n",
		"max_model_tokens = 170000\ncustom_model_max_tokens = 200000\n",
	} {
		repo := budgetRepo(t, toml, budgetStepFirstFixed)
		if n, text := budgetProblemText(t, repo); n == 0 || !strings.Contains(text, "prompt clip in") {
			t.Errorf("want a clip problem for %q, got %d:\n%s", toml, n, text)
		}
	}
}

func TestCheckReviewBudget_TheClipMustLeaveSlackNotJustFit(t *testing.T) {
	// 160000 + 40000 exactly fills a 200000 window, which is zero slack and the
	// guard refuses it; one token less is room for the answer to be a token
	// longer than declared. Both sides are asserted so the boundary is the
	// stated one rather than whatever the arithmetic happens to produce.
	if n, text := budgetProblemText(t, budgetRepo(t,
		"max_model_tokens = 160000\ncustom_model_max_tokens = 200000\n", budgetStepFirstFixed)); n == 0 {
		t.Errorf("160000 + 40000 = 200000 is not under 200000, so this must fail:\n%s", text)
	}
	if n, text := budgetProblemText(t, budgetRepo(t,
		"max_model_tokens = 159000\ncustom_model_max_tokens = 200000\n", budgetStepFirstFixed)); n != 0 {
		t.Errorf("159000 + 40000 is under 200000; got %d:\n%s", n, text)
	}
}

func TestCheckReviewBudget_RetryFalseHalvesTheAttempts(t *testing.T) {
	// Three models, 120 s each: one attempt fits a 6 minute ceiling, two do not.
	declared := `      - name: PR-Agent
        uses: docker://pragent/pr-agent@sha256:0000
        timeout-minutes: 6
        env:
          CONFIG__FALLBACK_MODELS: ["a", "b", "c"]
          CONFIG__AI_TIMEOUT: "120"
          CONFIG__MAX_OUTPUT_TOKENS: "40000"
          CONFIG__RETRY_SAME_MODEL_ON_TIMEOUT: "false"
`
	undeclared := `      - name: PR-Agent
        uses: docker://pragent/pr-agent@sha256:0000
        timeout-minutes: 6
        env:
          CONFIG__FALLBACK_MODELS: ["a", "b", "c"]
          CONFIG__AI_TIMEOUT: "120"
          CONFIG__MAX_OUTPUT_TOKENS: "40000"
`
	if n, text := budgetProblemText(t, budgetRepo(t, budgetTomlFixed, declared)); n != 0 {
		t.Errorf("one attempt of 360 s fits 360 s; got %d:\n%s", n, text)
	}
	if n, text := budgetProblemText(t, budgetRepo(t, budgetTomlFixed, undeclared)); n == 0 ||
		!strings.Contains(text, "2 attempts") {
		t.Errorf("two attempts of 360 s do not fit 360 s; got %d:\n%s", n, text)
	}
}

func TestCheckReviewBudget_AnExpressionChainUsesTheTomlPoolAsItsBound(t *testing.T) {
	// The NaN step's list is a GitHub expression, so the guard falls back to
	// the pool the route draws from: 1 + 2 models, not 1.
	repo := budgetRepo(t, budgetTomlFixed, `      - name: PR-Agent
        uses: docker://pragent/pr-agent@sha256:0000
        timeout-minutes: 5
        env:
          CONFIG__FALLBACK_MODELS: ${{ steps.route.outputs.nan_fallbacks }}
          CONFIG__AI_TIMEOUT: "120"
          CONFIG__MAX_OUTPUT_TOKENS: "40000"
          CONFIG__RETRY_SAME_MODEL_ON_TIMEOUT: "false"
`)
	if n, text := budgetProblemText(t, repo); n == 0 || !strings.Contains(text, "3 models x 1 attempts") {
		t.Fatalf("want the toml pool as the bound, got %d:\n%s", n, text)
	}
}

func TestCheckReviewBudget_AStepThatDeclaresItsOwnClipIsCheckedToo(t *testing.T) {
	// The workflow's env overrides .pr_agent.toml, so a lowered file with a
	// step still exporting the old clip is clipped by a number nobody changed.
	step := `      - name: PR-Agent
        uses: docker://pragent/pr-agent@sha256:0000
        timeout-minutes: 13
        env:
          CONFIG__MAX_MODEL_TOKENS: "200000"
          CONFIG__CUSTOM_MODEL_MAX_TOKENS: "200000"
          CONFIG__MAX_OUTPUT_TOKENS: "40000"
          CONFIG__AI_TIMEOUT: "360"
`
	n, text := budgetProblemText(t, budgetRepo(t, budgetTomlFixed, step))
	if n == 0 || !strings.Contains(text, "the prompt clip in PR-Agent never fires") {
		t.Fatalf("want the step's own clip checked, got %d:\n%s", n, text)
	}
}

func TestCheckReviewBudget_AStepThatLowersItsOwnWindowIsMeasuredAgainstIt(t *testing.T) {
	// Reviewed on #3289: CONFIG__CUSTOM_MODEL_MAX_TOKENS overrides the toml
	// like the clip does, so a step window of 160000 under a 150000 clip and
	// a 40000 answer leaves no room, while the file's 200000 would say it does.
	step := `      - name: PR-Agent
        uses: docker://pragent/pr-agent@sha256:0000
        timeout-minutes: 13
        env:
          CONFIG__CUSTOM_MODEL_MAX_TOKENS: "160000"
          CONFIG__MAX_OUTPUT_TOKENS: "40000"
          CONFIG__AI_TIMEOUT: "360"
`
	toml := "max_model_tokens = 150000\ncustom_model_max_tokens = 200000\n"
	n, text := budgetProblemText(t, budgetRepo(t, toml, step))
	if n == 0 || !strings.Contains(text, "the prompt clip in PR-Agent never fires: max_model_tokens 150000 + answer cap 40000 = 190000, not under custom_model_max_tokens 160000") {
		t.Fatalf("want the step's own window used, got %d:\n%s", n, text)
	}
	if n, text := budgetProblemText(t, budgetRepo(t, toml, strings.Replace(step, `"160000"`, `"200000"`, 1))); n != 0 {
		t.Fatalf("a step window equal to the file's changes nothing, got %d:\n%s", n, text)
	}
}

func TestCheckReviewBudget_AStepWithoutItsOwnTimeoutSaysSo(t *testing.T) {
	// The step inherits the job's ceiling, which this guard does not model. The
	// line must say that rather than report the zero as a ceiling of its own.
	step := `      - name: PR-Agent
        uses: docker://pragent/pr-agent@sha256:0000
        env:
          CONFIG__AI_TIMEOUT: "360"
          CONFIG__MAX_OUTPUT_TOKENS: "40000"
`
	n, text := budgetProblemText(t, budgetRepo(t, budgetTomlFixed, step))
	if n == 0 || !strings.Contains(text, "declares no timeout-minutes") {
		t.Fatalf("want the missing step ceiling named, got %d:\n%s", n, text)
	}
	if strings.Contains(text, "in a 0 s ceiling") {
		t.Errorf("the guard must not report a 0 s ceiling:\n%s", text)
	}
}

func TestCheckReviewBudget_AnAnswerCapItCannotReadIsAGapNotAPass(t *testing.T) {
	// Zero is the pass direction of `clip + reserve < window`, so an expression
	// or a typo'd number would report room for an answer nobody sized. Both
	// forms are checked, because the expression is the one this workflow can
	// actually acquire.
	for _, raw := range []string{"${{ vars.MAX_OUTPUT }}", "40.000"} {
		step := `      - name: PR-Agent
        uses: docker://pragent/pr-agent@sha256:0000
        timeout-minutes: 13
        env:
          CONFIG__AI_TIMEOUT: "360"
          CONFIG__MAX_OUTPUT_TOKENS: "` + raw + `"
`
		repo := budgetRepo(t, budgetTomlFixed, step)
		n, text := budgetProblemText(t, repo)
		if n == 0 || !strings.Contains(text, "cannot read as a number") {
			t.Errorf("an unreadable answer cap %q must be a gap, got %d:\n%s", raw, n, text)
		}
		// All the lines, not just the problems: the clip line this asserts the
		// absence of is an OK line, and budgetProblemText drops exactly those,
		// so the assertion could never fail (#3301 review).
		if all := budgetAllText(t, repo); strings.Contains(all, "OK -> prompt clip") {
			t.Errorf("no clip may be reported OK while the reserve is unreadable (%q):\n%s", raw, all)
		}
	}
}

func TestCheckReviewBudget_AWorkflowThatNeverDeclaresTheAnswerCapSaysSo(t *testing.T) {
	// The image supplies its own default, which this guard cannot see. Saying
	// nothing would claim the clip leaves room for an answer nobody declared.
	step := `      - name: PR-Agent
        uses: docker://pragent/pr-agent@sha256:0000
        timeout-minutes: 13
        env:
          CONFIG__AI_TIMEOUT: "360"
`
	n, text := budgetProblemText(t, budgetRepo(t, budgetTomlFixed, step))
	if n == 0 || !strings.Contains(text, "no review step declares CONFIG__MAX_OUTPUT_TOKENS") {
		t.Fatalf("want the undeclared answer cap named, got %d:\n%s", n, text)
	}
}

func TestCheckReviewBudget_AStreamedStepIsReportedAsALowerBound(t *testing.T) {
	// A streamed call is bounded in chunks, not in total, so the same numbers
	// prove less. The line says so rather than reading as coverage.
	streamed := `      - name: PR-Agent
        uses: docker://pragent/pr-agent@sha256:0000
        timeout-minutes: 12
        env:
          CONFIG__AI_TIMEOUT: "120"
          CONFIG__MAX_OUTPUT_TOKENS: "40000"
          LITELLM__FORCE_STREAMING_API_BASE_SUBSTRINGS: '["api.nan.builders"]'
`
	plain := strings.Replace(streamed, "          LITELLM__FORCE_STREAMING_API_BASE_SUBSTRINGS: '[\"api.nan.builders\"]'\n", "", 1)

	if text := budgetAllText(t, budgetRepo(t, budgetTomlFixed, streamed)); !strings.Contains(text, "lower bound on the step's wall clock") {
		t.Errorf("a streamed step's ceiling is a lower bound and must say so:\n%s", text)
	}
	if text := budgetAllText(t, budgetRepo(t, budgetTomlFixed, plain)); strings.Contains(text, "lower bound") {
		t.Errorf("an unstreamed step must not carry the caveat:\n%s", text)
	}
}

// budgetAllText is every line the guard emits, OK ones included.
func budgetAllText(t *testing.T, repoRoot string) string {
	t.Helper()
	var texts []string
	for _, l := range checkReviewBudget(repoRoot) {
		texts = append(texts, l.Text)
	}
	return strings.Join(texts, "\n")
}

func TestCheckReviewBudget_AWorkflowWithoutAReviewStepFailsClosed(t *testing.T) {
	repo := budgetRepo(t, budgetTomlFixed, `      - uses: actions/checkout@0000
        timeout-minutes: 1
`)
	n, text := budgetProblemText(t, repo)
	if n == 0 || !strings.Contains(text, prAgentImage) {
		t.Fatalf("want an error naming the marker, got %d:\n%s", n, text)
	}
}

func TestCheckReviewBudget_HalfAReviewerFailsClosed(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, prAgentTomlFile), budgetTomlFixed)
	if n, text := budgetProblemText(t, repo); n == 0 || !strings.Contains(text, prAgentWorkflowFile) {
		t.Fatalf("want an error naming the missing workflow, got %d:\n%s", n, text)
	}
	other := t.TempDir()
	writeFile(t, filepath.Join(other, filepath.FromSlash(prAgentWorkflowFile)), budgetWorkflow(34, budgetStepFirstFixed))
	if n, text := budgetProblemText(t, other); n == 0 || !strings.Contains(text, prAgentTomlFile) {
		t.Fatalf("want an error naming the missing toml, got %d:\n%s", n, text)
	}
}

func TestCheckReviewBudget_ACheckoutWithoutAReviewerSaysNothing(t *testing.T) {
	if n, text := budgetProblemText(t, t.TempDir()); n != 0 {
		t.Fatalf("a repo without pr-agent has no budget to guard, got %d:\n%s", n, text)
	}
}

func TestCheckReviewBudget_TheRepositoryPassesItsOwnGuard(t *testing.T) {
	// The gate runs against this checkout, so the two files here must satisfy
	// the guard the same way CI requires them to.
	root := filepath.Join("..", "..", "..")
	if n, text := budgetProblemText(t, root); n != 0 {
		t.Fatalf("this repository's own reviewer budget does not hold, got %d:\n%s", n, text)
	}
}

func TestCheckReviewBudget_AFileWithoutAClipFailsClosed(t *testing.T) {
	toml := "custom_model_max_tokens = 200000\n"
	if n, text := budgetProblemText(t, budgetRepo(t, toml, budgetStepFirstFixed)); n == 0 || !strings.Contains(text, "max_model_tokens is absent") {
		t.Fatalf("want a missing clip reported, got %d:\n%s", n, text)
	}
}

func TestCheckReviewBudget_TheReserveIsTheLargestCapNotTheFirst(t *testing.T) {
	// Two steps can declare different caps, and the window must be reserved
	// for the roomiest answer, not the one that happens to be published first.
	// 150000 + 40000 fits under 200000; 150000 + 60000 does not, so a reserve
	// that stopped at the first cap would pass on step order alone.
	small := budgetStepFirstFixed
	large := strings.Replace(budgetStepSecondFixed, `"40000"`, `"60000"`, 1)
	toml := "max_model_tokens = 150000\ncustom_model_max_tokens = 200000\n"
	if n, text := budgetProblemText(t, budgetRepo(t, toml, small+large)); n == 0 || !strings.Contains(text, "answer cap 60000") {
		t.Fatalf("want the 60000 cap to be the reserve, got %d:\n%s", n, text)
	}
}

func TestCheckReviewBudget_AnUnparseableFileFailsClosed(t *testing.T) {
	// A reviewer whose configuration cannot be read has a budget nobody can
	// vouch for, so garbage in either file is a finding naming that file, not
	// a silent pass over the half that still parses.
	brokenToml := budgetRepo(t, "[config\nmax_model_tokens = ", budgetStepFirstFixed)
	if n, text := budgetProblemText(t, brokenToml); n == 0 || !strings.Contains(text, prAgentTomlFile) {
		t.Fatalf("want a problem naming the unparseable toml, got %d:\n%s", n, text)
	}
	brokenWorkflow := t.TempDir()
	writeFile(t, filepath.Join(brokenWorkflow, prAgentTomlFile), budgetTomlFixed)
	writeFile(t, filepath.Join(brokenWorkflow, filepath.FromSlash(prAgentWorkflowFile)), "jobs:\n  review: [\n")
	if n, text := budgetProblemText(t, brokenWorkflow); n == 0 || !strings.Contains(text, prAgentWorkflowFile) {
		t.Fatalf("want a problem naming the unparseable workflow, got %d:\n%s", n, text)
	}
}
