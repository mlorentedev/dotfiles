// Package nanquota answers "which NaN model we route traffic to is near its
// quota" from NaN's own numbers (AI-047, #1766).
//
// NaN meters each model separately, per month, and a spent quota answers 402
// until the period resets. GET /v1/usage reports tokens per model but no
// limits, so the limits come from a declared table (harness/nan-quotas.json)
// that carries its source and the date it was read.
//
// Everything here is pure: the caller fetches, this package classifies. That
// keeps the classification testable against a recorded response and keeps the
// key out of this package entirely.
package nanquota

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Level is a finding's severity, mapped by the caller onto its own report.
type Level int

const (
	Pass Level = iota
	Info
	Warn
	Fail
)

// Finding is the verdict for one bound model.
type Finding struct {
	Model string
	Level Level
	Msg   string
}

// ModelTotal is one entry of totals.by_model.
type ModelTotal struct {
	Model       string `json:"model"`
	TotalTokens int64  `json:"total_tokens"`
}

// Usage is the part of GET /v1/usage the check reads. totals covers the whole
// requested window even when data is paged (measured 2026-09-26: totals under
// limit=3 equal totals unpaged), so data rows are never summed here.
type Usage struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Totals    struct {
		ByModel []ModelTotal `json:"by_model"`
	} `json:"totals"`
}

// Quota is one metered model's declared limit.
type Quota struct {
	Tokens int64  `json:"tokens"`
	Period string `json:"period"`
	Note   string `json:"note,omitempty"`
}

// Table is harness/nan-quotas.json. It is closed-world: every bound NaN model
// is either metered or unmetered, and one in neither list is its own finding.
type Table struct {
	Checked   string           `json:"checked"`
	Source    string           `json:"source"`
	Metered   map[string]Quota `json:"metered"`
	Unmetered []string         `json:"unmetered"`
}

// ParseUsage decodes a GET /v1/usage body.
func ParseUsage(body []byte) (Usage, error) {
	var u Usage
	if err := json.Unmarshal(body, &u); err != nil {
		return Usage{}, fmt.Errorf("parse /v1/usage: %w", err)
	}
	// Absent is not zero: read as zero, a spent model would PASS.
	if u.Totals.ByModel == nil {
		return Usage{}, fmt.Errorf("parse /v1/usage: no totals.by_model")
	}
	return u, nil
}

// ParseModels decodes a GET /v1/models body into the set of served ids.
func ParseModels(body []byte) (map[string]bool, error) {
	var doc struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse /v1/models: %w", err)
	}
	// An empty set would FAIL every binding at once; that is NaN answering
	// oddly, not NaN dropping every model.
	if len(doc.Data) == 0 {
		return nil, fmt.Errorf("parse /v1/models: no models listed")
	}
	served := make(map[string]bool, len(doc.Data))
	for _, m := range doc.Data {
		served[m.ID] = true
	}
	return served, nil
}

// ParseTable decodes and validates the declared quota table.
func ParseTable(raw []byte) (Table, error) {
	var t Table
	if err := json.Unmarshal(raw, &t); err != nil {
		return Table{}, fmt.Errorf("parse quota table: %w", err)
	}
	if _, err := time.Parse(time.DateOnly, t.Checked); err != nil {
		return Table{}, fmt.Errorf("quota table: checked %q is not a YYYY-MM-DD date", t.Checked)
	}
	// Closed-world: an omitted list would make every model in it read as
	// undeclared, so each must be stated, even when empty.
	switch {
	case t.Source == "":
		return Table{}, fmt.Errorf("quota table: source is required")
	case t.Metered == nil:
		return Table{}, fmt.Errorf("quota table: metered is required (use {} for none)")
	case t.Unmetered == nil:
		return Table{}, fmt.Errorf("quota table: unmetered is required (use [] for none)")
	}
	for id, q := range t.Metered {
		if q.Tokens <= 0 || q.Period == "" {
			return Table{}, fmt.Errorf("quota table: %s needs positive tokens and a period", id)
		}
	}
	for _, id := range t.Unmetered {
		if _, dup := t.Metered[id]; dup {
			return Table{}, fmt.Errorf("quota table: %s is declared both metered and unmetered", id)
		}
	}
	return t, nil
}

// Age is how long ago the table's limits were read.
func (t Table) Age(now time.Time) (time.Duration, error) {
	checked, err := time.Parse(time.DateOnly, t.Checked)
	if err != nil {
		return 0, err
	}
	return now.UTC().Sub(checked), nil
}

// PeriodStart is the first day of the current quota period, as the start_date
// /v1/usage takes. NaN's docs say "month"; its usage dates are UTC. Treating the
// period as the UTC calendar month is an assumption the API cannot confirm: it
// reports usage, not the period it meters against.
func PeriodStart(now time.Time) string {
	u := now.UTC()
	return time.Date(u.Year(), u.Month(), 1, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)
}

// BoundModels lists, sorted and deduplicated, every model id the routing map
// binds to pool: tiers.<tier>.<pool>, "<pool>:<id>" chain entries, and services
// whose pool is pool.
func BoundModels(m map[string]any, pool string) []string {
	set := map[string]bool{}
	for _, tier := range asMap(m["tiers"]) {
		if id, ok := asMap(tier)[pool].(string); ok {
			set[id] = true
		}
	}
	for _, chain := range asMap(m["chains"]) {
		for _, e := range asSlice(chain) {
			if s, ok := e.(string); ok && strings.HasPrefix(s, pool+":") {
				set[strings.TrimPrefix(s, pool+":")] = true
			}
		}
	}
	for _, svc := range asMap(m["services"]) {
		sm := asMap(svc)
		if id, ok := sm["model"].(string); ok && sm["pool"] == pool {
			set[id] = true
		}
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Evaluate classifies every model that can spend the account's quota. The
// quota belongs to the API key, not to a binding, so the watched set is the
// union of what model-map.json binds, what the table meters, and what has
// usage this period: a consumer outside model-map.json (pi, opencode, a
// one-shot wrapper) spends the same budget. warnPct is the share of the quota
// at which a metered model WARNs; at 100% it FAILs, because NaN then answers
// 402.
func Evaluate(bound []string, served map[string]bool, u Usage, t Table, warnPct float64) []Finding {
	used := map[string]int64{}
	for _, mt := range u.Totals.ByModel {
		used[mt.Model] += mt.TotalTokens
	}
	unmetered := map[string]bool{}
	for _, id := range t.Unmetered {
		unmetered[id] = true
	}
	isBound := map[string]bool{}
	for _, id := range bound {
		isBound[id] = true
	}
	watched := map[string]bool{}
	for _, id := range bound {
		watched[id] = true
	}
	for id := range t.Metered {
		watched[id] = true
	}
	for id, n := range used {
		if n > 0 {
			watched[id] = true
		}
	}
	ids := make([]string, 0, len(watched))
	for id := range watched {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	out := make([]Finding, 0, len(ids))
	for _, id := range ids {
		m := model{id: id, bound: isBound[id], served: served[id], used: used[id], unmetered: unmetered[id]}
		m.quota, m.metered = t.Metered[id]
		if f, ok := classify(m, warnPct); ok {
			out = append(out, f)
		}
	}
	return out
}

type model struct {
	id                                string
	bound, served, unmetered, metered bool
	used                              int64
	quota                             Quota
}

// classify reports one watched model. ok is false for an unbound model the key
// cannot see: it spends nothing, and nothing routes to it.
func classify(m model, warnPct float64) (f Finding, ok bool) {
	id := m.id
	if !m.served {
		if m.bound {
			return Finding{id, Fail, fmt.Sprintf("%s is bound in model-map.json but NaN does not serve it (absent from /v1/models)", id)}, true
		}
		// /v1/models filters premium models by the key's tier, so absence is not
		// retirement; either way this key cannot spend it.
		return Finding{}, false
	}
	if m.unmetered {
		return Finding{id, Pass, fmt.Sprintf("%s: unmetered, %s tokens this period", id, millions(m.used))}, true
	}
	if !m.metered {
		return Finding{id, Warn, fmt.Sprintf("%s is neither metered nor unmetered in harness/nan-quotas.json — declare it, or its quota is never watched", id)}, true
	}
	pct := float64(m.used) * 100 / float64(m.quota.Tokens)
	msg := fmt.Sprintf("%s: %s / %s tokens this %s (%.0f%%)", id, millions(m.used), millions(m.quota.Tokens), m.quota.Period, pct)
	switch {
	case pct >= 100:
		return Finding{id, Fail, msg + " — spent: NaN answers 402 until the period resets"}, true
	case pct >= warnPct:
		return Finding{id, Warn, msg + " — move routed traffic off it before it runs out"}, true
	}
	return Finding{id, Info, msg}, true
}

func millions(n int64) string { return fmt.Sprintf("%.1fM", float64(n)/1e6) }

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}
