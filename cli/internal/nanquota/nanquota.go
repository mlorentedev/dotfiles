// Package nanquota reports what NaN serves and what this account spent, from
// NaN's own numbers (AI-047, #1766).
//
// Since 2026-11 NaN bills every chat model from one pool of credits per billing
// period, priced per model and per token type, with cached input much cheaper.
// GET /v1/usage still reports raw tokens per model over a rolling window, with
// no cached split, no credits and no limit, so nothing here can compute a share
// of a quota: the balance lives on NaN's dashboard (ai/nan/README.md). What the
// API can answer honestly is whether every bound model is served, and how many
// tokens each one moved.
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
)

// Level is a finding's severity, mapped by the caller onto its own report.
type Level int

const (
	Pass Level = iota
	Info
	Warn
	Fail
)

// Finding is the verdict for one model.
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

// ParseUsage decodes a GET /v1/usage body.
func ParseUsage(body []byte) (Usage, error) {
	var u Usage
	if err := json.Unmarshal(body, &u); err != nil {
		return Usage{}, fmt.Errorf("parse /v1/usage: %w", err)
	}
	// Absent is not zero: read as zero, every model would report no usage.
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

// Evaluate reports every model that is bound or moved tokens in the window. A
// bound id NaN does not serve FAILs, because every agent routed there is
// already failing (the class that kept `qwen3-rerank` bound for weeks while NaN
// answered 401). Usage is INFO: tokens are not credits, so no share of anything
// is computed from them. An unbound model the key cannot see is not reported.
func Evaluate(bound []string, served map[string]bool, u Usage) []Finding {
	used := map[string]int64{}
	for _, mt := range u.Totals.ByModel {
		used[mt.Model] += mt.TotalTokens
	}
	isBound := map[string]bool{}
	for _, id := range bound {
		isBound[id] = true
	}
	watched := map[string]bool{}
	for _, id := range bound {
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
		switch {
		case isBound[id] && !served[id]:
			out = append(out, Finding{id, Fail, fmt.Sprintf("%s is bound in model-map.json but NaN does not serve it (absent from /v1/models)", id)})
		case served[id]:
			out = append(out, Finding{id, Info, fmt.Sprintf("%s: %s tokens since %s", id, millions(used[id]), u.StartDate)})
		}
	}
	return out
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
