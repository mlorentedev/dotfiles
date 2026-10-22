package prreview

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// none is the provider of an attempt that does not run.
const none = "none"

// Result is the route's decision.
type Result struct {
	First           string   `json:"first"`
	Second          string   `json:"second"`
	Reviewer        string   `json:"reviewer"`
	Route           string   `json:"route"`
	NanModel        string   `json:"nan_model"`
	NanFallbacks    []string `json:"nan_fallbacks"`
	AnthropicModel  string   `json:"anthropic_model"`
	AnthropicEffort string   `json:"anthropic_effort"`
	Note            string   `json:"note"`

	// What the summary and the notice report; not part of the output lines.
	Override string   `json:"override"`
	Pool     string   `json:"pool"`
	Version  string   `json:"dotf_version,omitempty"`
	Members  []string `json:"members"`
	risk     string
}

// KeyValues is the route's outputs as `key=value` lines, in the order and with
// the names the workflow reads.
func (r Result) KeyValues() string {
	return fmt.Sprintf("first=%s\nsecond=%s\nreviewer=%s\nroute=%s\nnan_model=%s\nnan_fallbacks=%s\n"+
		"anthropic_model=%s\nanthropic_effort=%s\nnote=%s\n",
		r.First, r.Second, r.Reviewer, r.Route, r.NanModel, compact(r.NanFallbacks),
		r.AnthropicModel, r.AnthropicEffort, r.Note)
}

// JSON is the result as one JSON object.
func (r Result) JSON() (string, error) {
	if r.NanFallbacks == nil {
		r.NanFallbacks = []string{}
	}
	if r.Members == nil {
		r.Members = []string{}
	}
	return marshal(r)
}

// Summary is the draw as Markdown for the job summary.
func (r Result) Summary() string {
	members := "none"
	if len(r.Members) > 0 {
		members = "`" + strings.Join(r.Members, "` `") + "`"
	}
	var b strings.Builder
	b.WriteString("### PR-Agent review pool\n\n")
	fmt.Fprintf(&b, "Pool: `%s`.", r.Pool)
	if r.Version != "" {
		fmt.Fprintf(&b, " dotf: `%s`.", r.Version)
	}
	fmt.Fprintf(&b, "\n\nOverride: `%s`. Members that answered: %s.\n\n", r.Override, members)
	switch {
	case r.First == none:
		b.WriteString("No member answered, so no review attempt runs.\n")
	case r.Route == "risk":
		fmt.Fprintf(&b, "Routed by risk (%s) to `%s`. Second attempt, only if nothing is published: %s.\n", r.risk, r.Reviewer, r.Second)
	default:
		fmt.Fprintf(&b, "Drawn to review first: `%s`. Second attempt, only if nothing is published: %s.\n", r.Reviewer, r.Second)
	}
	if r.Note != "" {
		fmt.Fprintf(&b, "\n%s\n", r.Note)
	}
	return b.String()
}

// Notice is the ::notice:: line naming the reviewer; empty when none reviews.
func (r Result) Notice() string {
	if r.First == none {
		return ""
	}
	return fmt.Sprintf("::notice::PR-Agent reviews first on %s (%s, override: %s, pool: %s); second attempt: %s.\n",
		r.Reviewer, r.Route, r.Override, r.Pool, r.Second)
}

// compact is v as compact JSON, with an empty list as [].
func compact(v []string) string {
	if v == nil {
		v = []string{}
	}
	s, _ := marshal(v)
	return s
}

// marshal is compact JSON without HTML escaping, as jq prints it.
func marshal(v any) (string, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}
