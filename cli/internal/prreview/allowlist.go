// Package prreview holds the decisions CI makes about a pull request's AI
// review: which pool member reviews it first, and on which provider.
package prreview

import "slices"

// Model is an Anthropic model CI may spend on, with the arithmetic that bounds
// the spend: its highest prices per million tokens (the rates above any price
// step) and the most one review of it may cost. The workflow's caps times these
// prices must stay under CeilingUSD, so a cap raised past a ceiling fails a test
// instead of the month's budget.
type Model struct {
	ID               string  `json:"model"`
	InputUSDPerMTok  float64 `json:"input_usd_per_mtok"`
	OutputUSDPerMTok float64 `json:"output_usd_per_mtok"`
	CeilingUSD       float64 `json:"ceiling_usd"`
}

// allowed is the allowlist. The Console cannot restrict a key to a model, so
// this list is the restriction: a pool edit naming another model fails the
// route instead of paying for it. Adding a model here is a deliberate cost
// decision, made with its price row and ceiling in a reviewed change. Prices
// are Anthropic's published rates per million tokens, at their highest: Haiku
// 5.5's above 100,000 prompt tokens, Sonnet 5.5's with no price step. The same
// rows live in tests/pr-agent-config.bats until the workflow switches to this
// command (TestAllowlistEqualsTheBatsPrices holds them equal). The ceiling is
// the most one review may cost, a limit the owner sets; the costs actually
// measured per review are a different thing and live in the pool's `why`.
var allowed = []Model{
	{ID: "anthropic/claude-haiku-5-5", InputUSDPerMTok: 0.50, OutputUSDPerMTok: 2.50, CeilingUSD: 0.25},
	{ID: "anthropic/claude-sonnet-5-5", InputUSDPerMTok: 2.00, OutputUSDPerMTok: 10.00, CeilingUSD: 1.00},
}

// Allowlist returns the models CI may spend on, in declared order.
func Allowlist() []Model { return slices.Clone(allowed) }

// Allowed reports whether id is exactly an allowlisted model.
func Allowed(id string) bool {
	return slices.ContainsFunc(allowed, func(m Model) bool { return m.ID == id })
}
