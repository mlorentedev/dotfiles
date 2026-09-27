package doctor

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
	"github.com/mlorentedev/dotfiles/cli/internal/nanquota"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// NaN quota (AI-047, #1766). NaN meters each model per month and answers 402
// once a quota is spent. On 2026-09-26 qwen3.8-flash stood at 83% with nothing
// watching it; the only alarm was hermes's digest, which summed three models
// against one 500M pool and could not see any client but hermes.
//
// This section reads NaN's own numbers (GET /v1/usage, per member) for every
// NaN model harness/model-map.json binds, and compares them with the declared
// table in harness/nan-quotas.json. It also checks each bound id against
// GET /v1/models, the class that kept `qwen3-rerank` bound for weeks while NaN
// answered 401.
//
// Severity: a spent quota and an unserved binding FAIL, because every agent
// routed there is already failing. Everything this section cannot know (no key
// on the box, NaN down, an unexpected answer) is a SKIP or a WARN, so an outage
// or a CI runner without the key never reads as a breach.
const (
	nanBaseURL        = "https://api.nan.builders/v1"
	nanKeyVar         = "NAN_API_KEY"
	nanQuotaFile      = "harness/nan-quotas.json"
	nanQuotaWarnPct   = 80
	nanQuotaTableLife = 90 * 24 * time.Hour
)

func checkNaNQuota(sys *System, cfg *Config, rep *Report) {
	rep.Section("NaN quota")

	table, bound, ok := loadNaNQuotaInputs(cfg, rep)
	if !ok {
		return
	}
	reportNaNTableAge(sys, table, rep)
	key, ok := resolveNaNKey(sys, cfg, rep)
	if !ok {
		return
	}
	usage, served, ok := fetchNaNUsage(sys, key, rep)
	if !ok {
		return
	}
	for _, f := range nanquota.Evaluate(bound, served, usage, table, nanQuotaWarnPct) {
		reportNaNFinding(f, rep)
	}
}

// loadNaNQuotaInputs reads the two repo files the section compares against. A
// missing or invalid one is a WARN: the checkout is broken, not the quota.
func loadNaNQuotaInputs(cfg *Config, rep *Report) (nanquota.Table, []string, bool) {
	raw, err := os.ReadFile(filepath.Join(cfg.DotfilesDir, nanQuotaFile))
	if err != nil {
		rep.Warn(fmt.Sprintf("%s unreadable (%v) — NaN quotas are not being watched", nanQuotaFile, err))
		return nanquota.Table{}, nil, false
	}
	table, err := nanquota.ParseTable(raw)
	if err != nil {
		rep.Warn(fmt.Sprintf("%s: %v — NaN quotas are not being watched", nanQuotaFile, err))
		return nanquota.Table{}, nil, false
	}
	m, err := harness.LoadModelMap(cfg.DotfilesDir)
	if err != nil {
		rep.Warn(fmt.Sprintf("model map unreadable (%v) — NaN quotas are not being watched", err))
		return nanquota.Table{}, nil, false
	}
	bound := nanquota.BoundModels(m, "nan")
	if len(bound) == 0 {
		rep.Skip("model-map.json binds no NaN model")
		return nanquota.Table{}, nil, false
	}
	return table, bound, true
}

// resolveNaNKey resolves NAN_API_KEY through the same Loader seam the PAT
// section uses. The key is never exported into the shell (ADR-028), so the
// environment is not read. Every miss is a SKIP that says the quotas are not
// being watched here, which is the honest state of a CI runner.
func resolveNaNKey(sys *System, cfg *Config, rep *Report) (string, bool) {
	e, ok := nanKeyEntry(cfg)
	if !ok {
		rep.Skip(nanKeyVar + " is not declared in the registry — NaN quotas are not being watched here")
		return "", false
	}
	if e.Backend == secrets.BackendBW {
		if state, err := sys.BWServeReadable(); state != secrets.BWServeReady {
			rep.Skip(fmt.Sprintf("%s is bw-backed and the bw serve daemon is %s (%v) — NaN quotas are not being watched here; run `dotf secrets unlock`",
				nanKeyVar, state, err))
			return "", false
		}
	}
	key, err := sys.ResolveSecret(e)
	if errors.Is(err, secrets.ErrSecretAbsent) {
		rep.Skip(nanKeyVar + " is not provisioned on this machine — NaN quotas are not being watched here")
		return "", false
	}
	if err != nil {
		rep.Warn(fmt.Sprintf("%s did not resolve (%v) — NaN quotas are not being watched; `dotf secrets verify` isolates the cause", nanKeyVar, err))
		return "", false
	}
	return key, true
}

func nanKeyEntry(cfg *Config) (secrets.Entry, bool) {
	reg, err := loadRegistry(cfg)
	if err != nil {
		return secrets.Entry{}, false
	}
	for _, e := range reg.Entries("") {
		if !e.IsFile && e.Var == nanKeyVar {
			return e, true
		}
	}
	return secrets.Entry{}, false
}

// fetchNaNUsage reads this period's usage and the served ids. No message built
// here includes the key or a response body: NaN's error bodies are not ours to
// echo, and a body is where a misbehaving proxy would reflect a header.
func fetchNaNUsage(sys *System, key string, rep *Report) (nanquota.Usage, map[string]bool, bool) {
	hdr := map[string]string{"Authorization": "Bearer " + key, "User-Agent": "dotf-doctor", "Accept": "application/json"}
	usageBody, ok := nanGet(sys, "/usage?start_date="+nanquota.PeriodStart(sys.Now()), hdr, rep)
	if !ok {
		return nanquota.Usage{}, nil, false
	}
	modelsBody, ok := nanGet(sys, "/models", hdr, rep)
	if !ok {
		return nanquota.Usage{}, nil, false
	}
	usage, err := nanquota.ParseUsage(usageBody)
	if err != nil {
		rep.Warn("NaN's /v1/usage answer did not parse — quotas inconclusive")
		return nanquota.Usage{}, nil, false
	}
	served, err := nanquota.ParseModels(modelsBody)
	if err != nil {
		rep.Warn("NaN's /v1/models answer did not parse — quotas inconclusive")
		return nanquota.Usage{}, nil, false
	}
	return usage, served, true
}

func nanGet(sys *System, path string, hdr map[string]string, rep *Report) ([]byte, bool) {
	status, body, err := sys.HTTPGetBody(nanBaseURL+path, hdr)
	switch {
	case err != nil:
		rep.Skip(fmt.Sprintf("NaN is unreachable (%v) — quotas not checked", err))
		return nil, false
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		rep.Warn(fmt.Sprintf("NaN refused %s (HTTP %d) — if the vault cache is stale run `dotf secrets unlock`; otherwise rotate %s", path, status, nanKeyVar))
		return nil, false
	case status != http.StatusOK:
		rep.Warn(fmt.Sprintf("unexpected HTTP %d from NaN %s — quotas inconclusive", status, path))
		return nil, false
	}
	return body, true
}

func reportNaNTableAge(sys *System, t nanquota.Table, rep *Report) {
	age, err := t.Age(sys.Now())
	if err == nil && age > nanQuotaTableLife {
		rep.Warn(fmt.Sprintf("%s was read %s, over 90 days ago — re-read %s and update the numbers and the date together",
			nanQuotaFile, t.Checked, t.Source))
	}
}

func reportNaNFinding(f nanquota.Finding, rep *Report) {
	switch f.Level {
	case nanquota.Fail:
		rep.Fail(f.Msg)
	case nanquota.Warn:
		rep.Warn(f.Msg)
	case nanquota.Info:
		rep.Info(f.Msg)
	default:
		rep.Pass(f.Msg)
	}
}
