package prreview

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// ErrRefused is returned when the route refuses to draw: an unknown provider
// override, a NaN preflight that failed on its setup, or a pool naming an
// Anthropic model outside the allowlist. The reason has already been written to
// Env.Out as an ::error:: line.
var ErrRefused = errors.New("pr route: refused")

// UsageError is a caller mistake: a draw point outside the total weight, say.
type UsageError string

func (e UsageError) Error() string { return string(e) }

// Inputs are the route's inputs. Each field is the raw text of the environment
// variable of the same name in the script this replaces, so a caller passes the
// workflow's values through unchanged.
type Inputs struct {
	NanOutcome    string // NAN_OUTCOME: `success` when a NaN model answered
	NanModel      string // NAN_MODEL: the first NaN model that answered
	NanFallbacks  string // NAN_FALLBACKS: JSON array of the others, in declared order
	PreflightExit string // NAN_PREFLIGHT_EXIT: 2 is a broken setup
	AnthropicKey  string // PR_AGENT_ANTHROPIC_API_KEY: empty means no Anthropic member
	Provider      string // PR_AGENT_PROVIDER: draw (default), nan, anthropic or nan-only
	ChangedLines  string // PR_CHANGED_LINES: additions plus deletions; empty when unread
	Labels        string // PR_LABELS: JSON array of label names
	Draw          string // PR_AGENT_DRAW: a point in [0, total weight), for tests
	APIBase       string // ANTHROPIC_API_BASE, default https://api.anthropic.com
	ProbeTimeout  string // ROUTE_PROBE_TIMEOUT: seconds per probe, default 30
	Repository    string // GITHUB_REPOSITORY: names the repository in the remedy
}

// Env is what the route reaches outside itself through.
type Env struct {
	Out    io.Writer       // workflow annotations (::warning:: and friends)
	Client *http.Client    // nil: a client that follows no redirect
	Intn   func(n int) int // nil: math/rand/v2
}

const (
	providerDraw      = "draw"
	providerNaN       = "nan"
	providerAnthropic = "anthropic"
	providerNaNOnly   = "nan-only"
)

// router is one Route call's state.
type router struct {
	pool      spec.ReviewerPool
	source    string
	in        Inputs
	env       Env
	provider  string
	nan       []string
	anthropic []string
	notes     []string
}

// Route draws which pool member reviews a PR first, and on which provider.
//
// The pool's `pr_agent` members that answered are drawn by weight: NaN's chain
// as the preflight reported it, and the Anthropic member after a probe. A member
// that did not answer is out of the draw and its weight is shared out among the
// rest. A `route: risk` member is never drawn: it reviews first when the PR
// crosses `pr_agent_risk`, and when it does not answer the PR goes to the draw
// with a note, because the quality tier being over its spend limit never means
// no review.
//
// One PR-Agent attempt talks to ONE provider, so the draw picks the provider of
// the first attempt and, within NaN, the model; the other provider, when one of
// its members answered, is the second attempt.
//
// Provider overrides the draw: nan puts NaN first with no risk route, anthropic
// puts Anthropic first, nan-only never probes Anthropic at all (the switch that
// stops all spending on the key). A forced provider with no member that
// answered falls back to the draw with a warning. An unknown value is refused:
// it may be a misspelt nan-only, and spending against a switch someone meant to
// turn off is the one outcome this must not produce.
func Route(ctx context.Context, pool spec.ReviewerPool, source string, in Inputs, env Env) (Result, error) {
	r := &router{pool: pool, source: source, in: in, env: env, provider: in.Provider}
	if r.provider == "" {
		r.provider = providerDraw
	}
	if err := r.refuse(); err != nil {
		return Result{}, err
	}
	for _, e := range pool.Pool {
		if e.PRAgent == nil {
			continue
		}
		if m := e.PRAgent.Model; strings.HasPrefix(m, "anthropic/") && !Allowed(m) {
			r.annotate("error", "%s names %s for PR-Agent, which is not in the allowlist in dotf pr route (%s)."+
				" Nothing is reviewed until it is fixed: an allowlist edit is a cost decision, made in a reviewed PR.",
				source, m, allowedIDs())
			return Result{}, ErrRefused
		}
	}
	return r.route(ctx)
}

// refuse stops the draw before any probe when the override or the preflight
// says a draw would spend wrongly.
func (r *router) refuse() error {
	switch r.provider {
	case providerDraw, providerNaN, providerAnthropic, providerNaNOnly:
	default:
		r.annotate("error", "PR_AGENT_PROVIDER is %s, which is none of draw, nan, anthropic or nan-only."+
			" Nothing is reviewed until it is fixed: a misspelt nan-only must not spend on the Anthropic key.", r.provider)
		return ErrRefused
	}
	// The preflight's exit 2 is a broken setup (no NAN_API_KEY, a missing tool, a
	// bad argument), not a NaN outage. Read as "no NaN member answered", it would
	// send every review to the paid member until someone noticed.
	if r.in.PreflightExit == "2" {
		r.annotate("error", "The NaN preflight failed on its setup (exit 2), not on NaN: its step names the cause"+
			" (a missing NAN_API_KEY, tool or argument). Nothing is drawn, so a broken setup cannot move every"+
			" review onto the paid Anthropic key.")
		return ErrRefused
	}
	return nil
}

func (r *router) route(ctx context.Context) (Result, error) {
	res := Result{First: none, Second: none, Route: providerDraw, NanFallbacks: []string{},
		Override: r.provider, Pool: r.source}
	r.nan = r.nanChain()

	drawn, riskModel := r.drawnAnthropic(), r.riskModel()
	keyOK := true
	if r.provider != providerNaNOnly && r.in.AnthropicKey == "" {
		keyOK = false
		repo := cmpOr(r.in.Repository, "<repo>")
		r.note("PR_AGENT_ANTHROPIC_API_KEY is not set, so no Anthropic model was in the review pool." +
			" Remedy: add ci:" + repo + " to its consumers in dotfiles/secrets/registry.yaml, then run:" +
			" dotf secrets sync ci --repo " + repo + " PR_AGENT_ANTHROPIC_API_KEY")
	}
	if r.provider != providerNaNOnly && keyOK && drawn != "" && r.answers(ctx, drawn) {
		r.anthropic = append(r.anthropic, drawn)
	}

	risky := r.riskReason(riskModel)
	if risky != "" && keyOK {
		if r.answers(ctx, riskModel) {
			res.Reviewer, res.Route, res.First, res.AnthropicModel = riskModel, "risk", providerAnthropic, riskModel
			if len(r.nan) > 0 {
				res.Second = providerNaN
			}
		} else {
			r.note("The PR crossed the risk line (" + risky + "), but " + riskModel + " did not answer, so it was drawn by weight instead.")
		}
	}
	res.risk = risky

	candidates := r.candidates()
	if res.Route == providerDraw {
		if err := r.draw(&res, candidates); err != nil {
			return Result{}, err
		}
	}
	r.attempts(&res)
	res.Members = slices.Concat(r.nan, r.anthropic)
	res.Note = strings.Join(r.notes, " ")
	return res, nil
}

// nanChain is the NaN members that answered, in declared order.
func (r *router) nanChain() []string {
	if r.in.NanOutcome != "success" || r.in.NanModel == "" {
		return nil
	}
	chain := []string{r.in.NanModel}
	var rest []string
	if json.Unmarshal([]byte(r.in.NanFallbacks), &rest) == nil {
		for _, m := range rest {
			if m != "" {
				chain = append(chain, m)
			}
		}
	}
	return chain
}

// riskReason says why the PR is risky: past the line count, or carrying the
// label. Empty when it is not, or when the override rules the risk route out.
func (r *router) riskReason(riskModel string) string {
	if riskModel == "" || r.provider == providerNaN || r.provider == providerNaNOnly {
		return ""
	}
	var limit *int
	var label string
	if risk := r.pool.PRAgentRisk; risk != nil {
		limit, label = risk.MinChangedLines, risk.Label
	}
	if n, isNum := digits(r.in.ChangedLines); !isNum {
		if limit != nil {
			r.note("The PR's size could not be read, so only the " + cmpOr(label, "risk") + " label could route it to " + riskModel + ".")
		}
	} else if limit != nil && n >= *limit {
		return fmt.Sprintf("%d changed lines (at least %d)", n, *limit)
	}
	var labels []string
	if label != "" && json.Unmarshal([]byte(r.in.Labels), &labels) == nil && slices.Contains(labels, label) {
		return "the " + label + " label"
	}
	return ""
}

// candidates are the members the draw picks from, by the override. A forced
// provider with no member that answered draws from the rest, with a warning,
// even when a risk route has already chosen the reviewer.
func (r *router) candidates() []string {
	var candidates []string
	switch r.provider {
	case providerDraw:
		candidates = slices.Concat(r.nan, r.anthropic)
	case providerNaN, providerNaNOnly:
		candidates = r.nan
	case providerAnthropic:
		candidates = r.anthropic
	}
	if len(candidates) == 0 && r.provider != providerDraw && r.provider != providerNaNOnly {
		r.annotate("warning", "PR_AGENT_PROVIDER=%s, but no %s member answered; drawing from the rest of the pool.", r.provider, r.provider)
		candidates = slices.Concat(r.nan, r.anthropic)
	}
	return candidates
}

// draw picks the first reviewer by weight from the candidates. Each holds
// `weight` points of [0, total). A member that answered but has no weight is
// never drawn first; it stays in the NaN chain as a fallback.
func (r *router) draw(res *Result, candidates []string) error {
	if len(candidates) == 0 {
		return nil
	}
	weights, total := make([]int, len(candidates)), 0
	for i, m := range candidates {
		weights[i] = max(r.weightOf(m), 0)
		total += weights[i]
	}
	if total == 0 {
		r.note("No member that answered has a weight in " + r.source + ", so nothing was drawn.")
		return nil
	}
	point, err := r.point(total)
	if err != nil {
		return err
	}
	for i, w := range weights {
		if point -= w; point < 0 {
			res.Reviewer = candidates[i]
			break
		}
	}
	if strings.HasPrefix(res.Reviewer, "anthropic/") {
		res.First = providerAnthropic
		if len(r.nan) > 0 {
			res.Second = providerNaN
		}
	} else {
		res.First = providerNaN
		// Empty under nan-only, which never probes it.
		if len(r.anthropic) > 0 {
			res.Second = providerAnthropic
		}
	}
	return nil
}

// point is the draw: PR_AGENT_DRAW when a test pins it, else a random point.
func (r *router) point(total int) (int, error) {
	if r.in.Draw == "" {
		if r.env.Intn != nil {
			return r.env.Intn(total), nil
		}
		return rand.IntN(total), nil
	}
	n, isNum := digits(r.in.Draw)
	if !isNum {
		return 0, UsageError("PR_AGENT_DRAW must be a point, got: " + r.in.Draw)
	}
	if n >= total {
		return 0, UsageError(fmt.Sprintf("PR_AGENT_DRAW %s is outside the total weight %d", r.in.Draw, total))
	}
	return n, nil
}

// attempts fills in the two attempts' models. The NaN attempt starts on the
// drawn model when it was drawn, and on the first NaN member otherwise; the rest
// stay behind it in declared order. The Anthropic attempt runs the risk member
// when it was routed there, and the drawn member otherwise (first or second).
func (r *router) attempts(res *Result) {
	if res.First != none && len(r.nan) > 0 {
		res.NanModel = r.nan[0]
		if res.First == providerNaN {
			res.NanModel = res.Reviewer
		}
		for _, m := range r.nan {
			if m != "" && m != res.NanModel {
				res.NanFallbacks = append(res.NanFallbacks, m)
			}
		}
	}
	if res.AnthropicModel == "" && len(r.anthropic) > 0 && (res.First == providerAnthropic || res.Second == providerAnthropic) {
		res.AnthropicModel = r.anthropic[0]
	}
	if res.AnthropicModel != "" {
		res.AnthropicEffort = cmpOr(r.block(res.AnthropicModel).ReasoningEffort, "medium")
	}
}

// answers probes one Anthropic member; on anything but a 2xx it adds the note
// naming the fix and reports false.
func (r *router) answers(ctx context.Context, model string) bool {
	code := Probe(ctx, r.env.Client, cmpOr(r.in.APIBase, "https://api.anthropic.com"), r.in.AnthropicKey, model,
		ParseTimeout(r.in.ProbeTimeout))
	ok, note := Verdict(model, code, r.in.ProbeTimeout)
	if !ok {
		r.note(note)
	}
	return ok
}

func (r *router) annotate(level, format string, args ...any) {
	if r.env.Out != nil {
		_, _ = fmt.Fprintf(r.env.Out, "::%s::%s\n", level, fmt.Sprintf(format, args...))
	}
}

// note records why a member was out of the draw, for the final guard.
func (r *router) note(text string) {
	r.annotate("warning", "%s", text)
	r.notes = append(r.notes, text)
}

// block is the first pool member's pr_agent block whose model is m.
func (r *router) block(m string) spec.PRAgentBlock {
	for _, e := range r.pool.Pool {
		if e.PRAgent != nil && e.PRAgent.Model == m {
			return *e.PRAgent
		}
	}
	return spec.PRAgentBlock{}
}

func (r *router) weightOf(m string) int { return r.block(m).Weight }

// drawnAnthropic is the first Anthropic member with a weight: the one probed
// for the draw.
func (r *router) drawnAnthropic() string {
	for _, e := range r.pool.Pool {
		if e.PRAgent != nil && strings.HasPrefix(e.PRAgent.Model, "anthropic/") && e.PRAgent.Weight > 0 {
			return e.PRAgent.Model
		}
	}
	return ""
}

// riskModel is the first member that reviews first on a risky PR.
func (r *router) riskModel() string {
	for _, e := range r.pool.Pool {
		if e.PRAgent != nil && e.PRAgent.Route == "risk" {
			return e.PRAgent.Model
		}
	}
	return ""
}

// allowedIDs lists the allowlisted ids, space separated, for a message.
func allowedIDs() string {
	ids := make([]string, len(allowed))
	for i, m := range allowed {
		ids[i] = m.ID
	}
	return strings.Join(ids, " ")
}

// digits reads s as a count; isNum is false when s is not all digits. A count
// too big for an int reads as the largest one.
func digits(s string) (n int, isNum bool) {
	if s == "" || strings.Trim(s, "0123456789") != "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		n = math.MaxInt
	}
	return n, true
}

// InputsFromEnv reads the route's inputs from the variables the workflow sets.
func InputsFromEnv(getenv func(string) string) Inputs {
	return Inputs{
		NanOutcome: getenv("NAN_OUTCOME"), NanModel: getenv("NAN_MODEL"), NanFallbacks: getenv("NAN_FALLBACKS"),
		PreflightExit: getenv("NAN_PREFLIGHT_EXIT"), AnthropicKey: getenv("PR_AGENT_ANTHROPIC_API_KEY"),
		Provider: getenv("PR_AGENT_PROVIDER"), ChangedLines: getenv("PR_CHANGED_LINES"), Labels: getenv("PR_LABELS"),
		Draw: getenv("PR_AGENT_DRAW"), APIBase: getenv("ANTHROPIC_API_BASE"), ProbeTimeout: getenv("ROUTE_PROBE_TIMEOUT"),
		Repository: getenv("GITHUB_REPOSITORY"),
	}
}
