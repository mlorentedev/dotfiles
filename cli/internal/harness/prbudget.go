package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"go.yaml.in/yaml/v3"
)

// The reviewer's budget lives in two files: the toml PR-Agent reads from the
// default branch, and the workflow that decides how long each attempt may run.
// Both are parsed with the libraries already in go.mod rather than matched by
// pattern, because a check that silently stops matching is the defect it
// exists to catch. What can still rot is prAgentImage, the marker that
// identifies a step running the reviewer: finding none is an error, not a pass.
const (
	prAgentTomlFile     = ".pr_agent.toml"
	prAgentWorkflowFile = ".github/workflows/pr-agent.yml"
	prAgentImage        = "pragent/pr-agent"

	// prAttemptsWithoutDeclaration is the bound for a step that leaves
	// `retry_same_model_on_timeout` at upstream's default: a model that times
	// out may be tried twice, so the step must afford two passes.
	prAttemptsWithoutDeclaration = 2
)

// Env keys the budget is computed from.
const (
	envAITimeout      = "CONFIG__AI_TIMEOUT"
	envFallbackModels = "CONFIG__FALLBACK_MODELS"
	envRetrySameModel = "CONFIG__RETRY_SAME_MODEL_ON_TIMEOUT"
	envMaxOutput      = "CONFIG__MAX_OUTPUT_TOKENS"
	envMaxModelTokens = "CONFIG__MAX_MODEL_TOKENS"
	// envModelWindow is the step's own model window. Lowering it on the step
	// narrows the room its clip has, so a step that declares it is checked
	// against its own window rather than the toml's.
	envModelWindow = "CONFIG__CUSTOM_MODEL_MAX_TOKENS"
	// envForceStreaming is the prefix of LiteLLM's forced-streaming settings.
	// A streamed call's ai_timeout bounds the gap between chunks rather than
	// the answer, which changes what the ceiling arithmetic proves.
	envForceStreaming = "LITELLM__FORCE_STREAMING"
)

// reviewConfig is everything the invariants are computed from.
type reviewConfig struct {
	maxModelTokens       int
	customModelMaxTokens int
	tomlFallbacks        int
	steps                []reviewStep
}

// reviewStep is one step that runs the reviewer, with the numbers its own
// ceiling must cover.
type reviewStep struct {
	name      string
	minutes   int
	aiTimeout int
	attempts  int
	chain     int
	outputCap int
	// maxTokens is the step's own prompt clip. The workflow's env overrides
	// .pr_agent.toml, so a step that declares it is checked against its own
	// number rather than the file's.
	maxTokens int
	// window is the step's own model window, 0 when the step inherits the
	// toml's. A step that narrows it has less room for the answer than the
	// file suggests, so the clip is checked against this one.
	window int
	// outputCapRaw is CONFIG__MAX_OUTPUT_TOKENS as written, empty when the step
	// does not declare it. It is kept because a zero reserve is the pass
	// direction of the clip invariant: an expression or a typo parses to 0, and
	// the check would then report room for an answer nobody sized.
	outputCapRaw string
	// streams marks a step whose call is streamed, where ai_timeout bounds the
	// gap between chunks rather than the answer. The ceiling arithmetic is
	// still worth reporting, but as a lower bound rather than as coverage.
	streams bool
}

// workflowStep is the part of a workflow step the budget reads.
type workflowStep struct {
	Name           string         `yaml:"name"`
	Uses           string         `yaml:"uses"`
	TimeoutMinutes int            `yaml:"timeout-minutes"`
	Env            map[string]any `yaml:"env"`
}

// workflowJob is the part of a job the budget reads. The job's own ceiling is
// checked by tests/pr-agent-config.bats, not here (see checkReviewBudget).
type workflowJob struct {
	Steps []workflowStep `yaml:"steps"`
}

// checkReviewBudget asks that the reviewer's budget protects something: the
// prompt clip must leave room for the answer, and every ceiling must afford
// the attempts and fallbacks the configuration declares.
//
// The job's own ceiling is deliberately not checked here. Covering it needs the
// probes' seconds and the publication calls' count as well as the pair of steps
// that can run, and modelling the route to find that pair would be fragile: an
// upper bound taken from the two largest step ceilings passes at 34 minutes on a
// job that needs 37, which is a guard that protects nothing. tests/pr-agent-config.bats
// computes it exactly, from the numbers where they are set, and it is what
// caught the 34-minute ceiling when a step ceiling grew (AI-054).
func checkReviewBudget(repoRoot string) []CheckLine {
	cfg, problem := loadReviewConfig(repoRoot)
	if problem != nil || cfg == nil {
		return problem
	}
	return append(checkTokenClip(cfg), checkStepCeilings(cfg)...)
}

// loadReviewConfig reads both files, reporting nil for a checkout that has no
// reviewer: a repo without pr-agent has no budget to check, which is not a
// pass over a file it failed to understand.
func loadReviewConfig(repoRoot string) (*reviewConfig, []CheckLine) {
	tomlRaw, tomlErr := os.ReadFile(filepath.Join(repoRoot, prAgentTomlFile))                     //nolint:gosec // repo-relative, fixed name
	wfRaw, wfErr := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(prAgentWorkflowFile))) //nolint:gosec // repo-relative, fixed name
	if tomlErr != nil && wfErr != nil {
		return nil, nil
	}
	// Half a reviewer is worse than none: one file present means the budget is
	// meant to be guarded, and an unreadable half must not read as a pass.
	if tomlErr != nil {
		return nil, []CheckLine{prBudgetProblem(prAgentTomlFile, tomlErr)}
	}
	if wfErr != nil {
		return nil, []CheckLine{prBudgetProblem(prAgentWorkflowFile, wfErr)}
	}
	cfg := &reviewConfig{}
	if err := parseReviewToml(tomlRaw, cfg); err != nil {
		return nil, []CheckLine{prBudgetProblem(prAgentTomlFile, err)}
	}
	if err := parseReviewWorkflow(wfRaw, cfg); err != nil {
		return nil, []CheckLine{prBudgetProblem(prAgentWorkflowFile, err)}
	}
	return cfg, nil
}

// parseReviewToml reads the token budget and the fallback pool. The keys live
// under [config], which is where PR-Agent reads them; a top-level key is
// accepted too, because that is where the fixtures and a shorter file put them.
// The pool is the upper bound for a step whose own list is a GitHub expression:
// the route can only draw from it.
func parseReviewToml(raw []byte, cfg *reviewConfig) error {
	type budget struct {
		MaxModelTokens       int      `toml:"max_model_tokens"`
		CustomModelMaxTokens int      `toml:"custom_model_max_tokens"`
		FallbackModels       []string `toml:"fallback_models"`
	}
	var doc struct {
		Config budget `toml:"config"`
	}
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("not valid TOML: %w", err)
	}
	t := doc.Config
	if t.CustomModelMaxTokens == 0 {
		if err := toml.Unmarshal(raw, &t); err != nil {
			return fmt.Errorf("not valid TOML: %w", err)
		}
	}
	if t.CustomModelMaxTokens <= 0 {
		return errors.New("custom_model_max_tokens is absent or not positive, so the prompt clip has no ceiling to compare against")
	}
	cfg.maxModelTokens = t.MaxModelTokens
	cfg.customModelMaxTokens = t.CustomModelMaxTokens
	cfg.tomlFallbacks = len(t.FallbackModels)
	return nil
}

// parseReviewWorkflow finds the job that runs the reviewer and records its
// ceiling and each review step's. No review step is an error.
func parseReviewWorkflow(raw []byte, cfg *reviewConfig) error {
	var doc struct {
		Jobs map[string]workflowJob `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("not valid YAML: %w", err)
	}
	for _, job := range doc.Jobs {
		steps := reviewStepsOf(job.Steps, cfg)
		if len(steps) == 0 {
			continue
		}
		cfg.steps = append(cfg.steps, steps...)
	}
	if len(cfg.steps) == 0 {
		return errors.New("no step runs the reviewer (" + prAgentImage + "), so the budget guards nothing")
	}
	return nil
}

// reviewStepsOf keeps the steps that run the reviewer.
func reviewStepsOf(steps []workflowStep, cfg *reviewConfig) []reviewStep {
	var out []reviewStep
	for _, s := range steps {
		if strings.Contains(s.Uses, prAgentImage) {
			out = append(out, newReviewStep(s, cfg))
		}
	}
	return out
}

// newReviewStep reads the four numbers from a step's env, defaulting the
// fallback chain to the toml's pool and the attempts to the conservative bound.
func newReviewStep(s workflowStep, cfg *reviewConfig) reviewStep {
	st := reviewStep{
		name:         orDefault(s.Name, "unnamed review step"),
		minutes:      s.TimeoutMinutes,
		attempts:     prAttemptsWithoutDeclaration,
		chain:        cfg.tomlFallbacks + 1,
		aiTimeout:    envInt(s.Env[envAITimeout]),
		outputCap:    envInt(s.Env[envMaxOutput]),
		outputCapRaw: strings.TrimSpace(envText(s.Env[envMaxOutput])),
		maxTokens:    envInt(s.Env[envMaxModelTokens]),
		window:       envInt(s.Env[envModelWindow]),
		streams:      declaresStreaming(s.Env),
	}
	if list, ok := literalList(s.Env[envFallbackModels]); ok {
		st.chain = len(list) + 1
	}
	if strings.EqualFold(strings.TrimSpace(envText(s.Env[envRetrySameModel])), "false") {
		st.attempts = 1
	}
	return st
}

// declaresStreaming reports whether the step turns on LiteLLM's forced
// streaming, which is what makes ai_timeout a chunk-gap bound rather than an
// answer bound. Any of the FORCE_STREAMING_* settings counts: the provider one
// is enough for PR-Agent to stream a matching request.
func declaresStreaming(env map[string]any) bool {
	for k := range env {
		if strings.HasPrefix(k, envForceStreaming) {
			return true
		}
	}
	return false
}

// checkTokenClip asserts that the prompt cap plus the largest declared answer
// stays under the model window. Equal numbers are the failure this exists for:
// the cap then has no reachable subject. A step that declares its own clip or
// its own window is checked too, because the workflow's env overrides the toml:
// lowering only the file would leave the run clipped by a number nobody
// changed, and raising only the step's clip would fill a window nobody sized.
//
// A reserve this check cannot read is a gap, not a pass. Zero is the pass
// direction of `clip + reserve < window`, so an expression, a typo'd number or
// no declaration at all would otherwise report room for an answer nobody sized.
func checkTokenClip(cfg *reviewConfig) []CheckLine {
	if cfg.maxModelTokens <= 0 {
		return []CheckLine{prBudgetProblem(prAgentTomlFile,
			errors.New("max_model_tokens is absent or not positive, so no prompt is ever clipped"))}
	}
	reserve, gaps := answerReserve(cfg)
	if len(gaps) > 0 {
		return gaps
	}
	out := clipLine(prAgentTomlFile, cfg.maxModelTokens, reserve, cfg.customModelMaxTokens)
	for _, s := range cfg.steps {
		// A step that exports neither number runs on the file's, already
		// checked above; one that exports either is clipped and windowed by
		// its own pair.
		if s.maxTokens <= 0 && s.window <= 0 {
			continue
		}
		out = append(out, clipLine(s.name, orInt(s.maxTokens, cfg.maxModelTokens), reserve, orInt(s.window, cfg.customModelMaxTokens))...)
	}
	return out
}

// answerReserve is the room the prompt must leave for the answer, and the gaps
// that make it unverifiable: a step whose declared cap is not a number, or no
// step declaring one at all. The largest declaration is the bound, because the
// window has to cover the worst case rather than the smallest.
func answerReserve(cfg *reviewConfig) (int, []CheckLine) {
	reserve := 0
	var gaps []CheckLine
	declared := false
	for _, s := range cfg.steps {
		if s.outputCapRaw == "" {
			continue
		}
		declared = true
		if s.outputCap <= 0 {
			gaps = append(gaps, prBudgetProblem(prAgentWorkflowFile,
				fmt.Errorf("the answer cap in %s is %q, which this check cannot read as a number, so nothing says how much room the prompt must leave",
					s.name, s.outputCapRaw)))
			continue
		}
		if s.outputCap > reserve {
			reserve = s.outputCap
		}
	}
	if !declared {
		gaps = append(gaps, prBudgetProblem(prAgentWorkflowFile,
			fmt.Errorf("no review step declares %s, so the room the prompt must leave for the answer is unknown", envMaxOutput)))
	}
	return reserve, gaps
}

// orInt is the effective value of a per-step override: the step's own number
// when it declares one, the file's otherwise.
func orInt(step, file int) int {
	if step > 0 {
		return step
	}
	return file
}

// clipLine is the invariant for one clip: prompt plus answer, under the window.
func clipLine(where string, clip, reserve, window int) []CheckLine {
	if clip+reserve < window {
		return []CheckLine{{Text: fmt.Sprintf("[check] OK -> prompt clip in %s: %d + %d < %d", where, clip, reserve, window)}}
	}
	return []CheckLine{{Problem: true,
		Text: fmt.Sprintf("[GAP] the prompt clip in %s never fires: max_model_tokens %d + answer cap %d = %d, not under custom_model_max_tokens %d",
			where, clip, reserve, clip+reserve, window),
		Detail: []string{fmt.Sprintf("      -> set the clip below %d so the prompt leaves room for the answer", window-reserve)}}}
}

// checkStepCeilings asserts each step's own bound.
func checkStepCeilings(cfg *reviewConfig) []CheckLine {
	out := make([]CheckLine, 0, len(cfg.steps))
	for _, s := range cfg.steps {
		out = append(out, stepCeiling(s)...)
	}
	return out
}

// stepCeiling asserts chain × attempts × ai_timeout fits the step's own
// timeout-minutes. A fallback the step cannot afford is not a fallback, and an
// undeclared ai_timeout is the defect this exists for: the step then runs
// whatever the image defaults to, which is a number this repository never
// chose and cannot check.
func stepCeiling(s reviewStep) []CheckLine {
	if s.aiTimeout <= 0 {
		return []CheckLine{{Problem: true,
			Text:   fmt.Sprintf("[GAP] %s does not declare %s, so its attempt budget is the image's default", s.name, envAITimeout),
			Detail: []string{"      -> declare it on the step; an inherited timeout is the setting that killed svqtriana run 42562380120"}}}
	}
	if s.minutes <= 0 {
		return []CheckLine{{Problem: true,
			Text:   fmt.Sprintf("[GAP] %s declares no timeout-minutes, so its ceiling is the job's and this check cannot see it", s.name),
			Detail: []string{fmt.Sprintf("      -> declare it on the step; %d s of attempts need %d minutes", s.chain*s.attempts*s.aiTimeout, prCeilDiv(s.chain*s.attempts*s.aiTimeout, 60))}}}
	}
	need, have := s.chain*s.attempts*s.aiTimeout, s.minutes*60
	if need <= have {
		line := fmt.Sprintf("[check] OK -> %s: %d s of attempts in a %d s ceiling", s.name, need, have)
		if s.streams {
			line += " (streamed: ai_timeout bounds the gap between chunks, not the answer, so this is a lower bound on the step's wall clock)"
		}
		return []CheckLine{{Text: line}}
	}
	return []CheckLine{{Problem: true,
		Text: fmt.Sprintf("[GAP] %s cannot afford its own attempts: %d models x %d attempts x %d s = %d s in a %d s ceiling",
			s.name, s.chain, s.attempts, s.aiTimeout, need, have),
		Detail: []string{fmt.Sprintf("      -> raise timeout-minutes to at least %d, or declare the settings the attempt count rests on",
			prCeilDiv(need, 60))}}}
}

func prCeilDiv(a, b int) int { return (a + b - 1) / b }

func prBudgetProblem(file string, err error) CheckLine {
	return CheckLine{Problem: true, Text: fmt.Sprintf("[ERROR] %s: %v", file, err)}
}

// envText renders an env value as written, whether YAML read it as a string or
// as a number.
func envText(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

// envInt is an env value as a number, and 0 for anything else, which the
// invariants then treat as a ceiling of nothing rather than a pass.
func envInt(v any) int {
	n, err := strconv.Atoi(strings.TrimSpace(envText(v)))
	if err != nil {
		return 0
	}
	return n
}

// literalList reads a fallback list written in the workflow itself. A GitHub
// expression reports false, and the caller's toml-derived bound is the safe
// answer there.
func literalList(v any) ([]string, bool) {
	s := strings.TrimSpace(envText(v))
	if !strings.HasPrefix(s, "[") || !strings.HasSuffix(s, "]") {
		return nil, false
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, false
	}
	return out, true
}
