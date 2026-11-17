package doctor

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
	"github.com/mlorentedev/dotfiles/cli/internal/nanquota"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// NaN models (AI-047, #1766). Every NaN model harness/model-map.json binds must
// be served: an unserved binding FAILs, the class that kept `qwen3-rerank` bound
// for weeks while NaN answered 401. Tokens per model are reported as INFO.
//
// No quota is judged. Since 2026-11 NaN bills one pool of credits per billing
// period, priced per model and token type, and GET /v1/usage reports raw tokens
// with no cached split and no limit, so no honest share can be computed from it.
// The per-model monthly table this section used to compare against
// (harness/nan-quotas.json) described a billing NaN no longer has.
//
// Everything this section cannot know (no key on the box, NaN down, an
// unexpected answer) is a SKIP or a WARN, so an outage or a CI runner without
// the key never reads as a broken binding.
const (
	nanBaseURL = "https://api.nan.builders/v1"
	nanKeyVar  = "NAN_API_KEY"
)

func checkNaNQuota(sys *System, cfg *Config, rep *Report) {
	rep.Section("NaN models")

	bound, ok := loadNaNBindings(cfg, rep)
	if !ok {
		return
	}
	key, ok := resolveNaNKey(sys, cfg, rep)
	if !ok {
		return
	}
	usage, served, ok := fetchNaNUsage(sys, key, rep)
	if !ok {
		return
	}
	for _, f := range nanquota.Evaluate(bound, served, usage) {
		reportNaNFinding(f, rep)
	}
	rep.Info("tokens are not credits: the credit balance is on NaN's dashboard (ai/nan/README.md)")
}

// loadNaNBindings reads the routing map. An unreadable one is a WARN: the
// checkout is broken, not NaN.
func loadNaNBindings(cfg *Config, rep *Report) ([]string, bool) {
	m, err := harness.LoadModelMap(cfg.DotfilesDir)
	if err != nil {
		rep.Warn(fmt.Sprintf("model map unreadable (%v) — NaN bindings are not being checked", err))
		return nil, false
	}
	bound := nanquota.BoundModels(m, "nan")
	if len(bound) == 0 {
		rep.Skip("model-map.json binds no NaN model")
		return nil, false
	}
	return bound, true
}

// resolveNaNKey resolves NAN_API_KEY through the same Loader seam the PAT
// section uses. The key is never exported into the shell (ADR-028), so the
// environment is not read. Every miss is a SKIP that says the bindings are not
// being watched here, which is the honest state of a CI runner.
func resolveNaNKey(sys *System, cfg *Config, rep *Report) (string, bool) {
	e, ok := nanKeyEntry(cfg)
	if !ok {
		rep.Skip(nanKeyVar + " is not declared in the registry — NaN bindings are not being checked here")
		return "", false
	}
	if e.Backend == secrets.BackendBW {
		if state, err := sys.BWServeReadable(); state != secrets.BWServeReady {
			rep.Skip(fmt.Sprintf("%s is bw-backed and the bw serve daemon is %s (%v) — NaN bindings are not being checked here; run `dotf secrets unlock`",
				nanKeyVar, state, err))
			return "", false
		}
	}
	key, err := sys.ResolveSecret(e)
	if errors.Is(err, secrets.ErrSecretAbsent) {
		rep.Skip(nanKeyVar + " is not provisioned on this machine — NaN bindings are not being checked here")
		return "", false
	}
	if err != nil {
		rep.Warn(fmt.Sprintf("%s did not resolve (%v) — NaN bindings are not being checked; `dotf secrets verify` isolates the cause", nanKeyVar, err))
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

// fetchNaNUsage reads the usage window NaN defaults to (30 rolling days) and the served ids. No message built
// here includes the key or a response body: NaN's error bodies are not ours to
// echo, and a body is where a misbehaving proxy would reflect a header.
func fetchNaNUsage(sys *System, key string, rep *Report) (nanquota.Usage, map[string]bool, bool) {
	hdr := map[string]string{"Authorization": "Bearer " + key, "User-Agent": "dotf-doctor", "Accept": "application/json"}
	usageBody, ok := nanGet(sys, "/usage", hdr, rep)
	if !ok {
		return nanquota.Usage{}, nil, false
	}
	modelsBody, ok := nanGet(sys, "/models", hdr, rep)
	if !ok {
		return nanquota.Usage{}, nil, false
	}
	usage, err := nanquota.ParseUsage(usageBody)
	if err != nil {
		rep.Warn("NaN's /v1/usage answer did not parse — NaN check inconclusive")
		return nanquota.Usage{}, nil, false
	}
	served, err := nanquota.ParseModels(modelsBody)
	if err != nil {
		rep.Warn("NaN's /v1/models answer did not parse — NaN check inconclusive")
		return nanquota.Usage{}, nil, false
	}
	return usage, served, true
}

func nanGet(sys *System, path string, hdr map[string]string, rep *Report) ([]byte, bool) {
	status, body, err := sys.HTTPGetBody(nanBaseURL+path, hdr)
	switch {
	case err != nil:
		rep.Skip(fmt.Sprintf("NaN is unreachable (%v) — NaN not checked", err))
		return nil, false
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		rep.Warn(fmt.Sprintf("NaN refused %s (HTTP %d) — if the vault cache is stale run `dotf secrets unlock`; otherwise rotate %s", path, status, nanKeyVar))
		return nil, false
	case status != http.StatusOK:
		rep.Warn(fmt.Sprintf("unexpected HTTP %d from NaN %s — NaN check inconclusive", status, path))
		return nil, false
	}
	return body, true
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
