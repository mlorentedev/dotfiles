package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Model limit drift: `ai/pi/models.json` declares a context window and an output
// cap per model, and the provider publishes the real ones.
//
// THE DEFECT, measured 2026-09-21. All SEVEN declared models disagreed with the
// catalog, and the two directions are not one bug:
//
//	nan/qwen3.8-flash      262144/16384   real 262144/131072  — 8x the output thrown away
//	nan/deepseek-v4-flash  1048576/65536  real 1000000/384000 — context 48K OVER the real one
//
// Nothing could have caught it. The numbers are hand-written in the repo, the
// provider publishes them elsewhere, and the two are never compared. They were
// plausible round powers of two, which is exactly why they survived review: 2^20
// reads as "1M" and is 48576 tokens more than the 1000000 the provider grants.
//
// WHY DOCTOR AND NOT CI, the same split checkModelPins documents next door. The
// truth side is a machine-local file — opencode's models.dev cache — and CI has
// no machine. Committing a snapshot of the catalog would only move the drift: a
// second hand-maintained copy of numbers somebody else publishes is the defect
// this check exists to report, one indirection further away.
//
// SEVERITY FOLLOWS CONSEQUENCE. Declaring MORE than the provider grants makes it
// reject a request the config invited, so it fails. Declaring LESS silently
// forfeits capability that was paid for, so it warns. Reporting both alike would
// either cry wolf over a conservative cap or bury a request-breaking overstatement.
//
// IT NEVER WRITES. The fix is a repo edit that belongs in a reviewed PR, next to
// the comment explaining which model changed and why — not a silent rewrite of a
// file whose values a human chose.

// catalogRelPath is opencode's models.dev cache: the same catalog opencode itself
// routes on, so a disagreement is a real inconsistency in this machine's toolchain
// rather than a difference of opinion between two directories.
const catalogRelPath = ".cache/opencode/models.json"

// piModelsRelPath is the DECLARATION half, read from the checkout rather than the
// deploy dir: a drift finding is only actionable where it can be committed.
const piModelsRelPath = "ai/pi/models.json"

// declaredModel is one entry of ai/pi/models.json, narrowed to the two fields this
// check compares. Everything else in the file (cost, modalities, compat) is out of
// scope — the provider publishes those too, but only these two change what a
// request is allowed to ask for.
type declaredModel struct {
	ID            string `json:"id"`
	ContextWindow int    `json:"contextWindow"`
	MaxTokens     int    `json:"maxTokens"`
}

// declaredProvider holds one provider's models as an ARRAY, which is what pi's
// config format uses. The catalog keys them by id instead, so the comparison is a
// loop on this side and a map hit on the other.
type declaredProvider struct {
	Models []declaredModel `json:"models"`
}

// declaredCatalog is the top of ai/pi/models.json: providers keyed by name, which
// is the level the per-provider scoping in lookupCatalogModel depends on.
type declaredCatalog struct {
	Providers map[string]declaredProvider `json:"providers"`
}

// catalogLimit is models.dev's shape: limit.context and limit.output.
type catalogLimit struct {
	Context int `json:"context"`
	Output  int `json:"output"`
}

// catalogModel is one models.dev entry. Only `limit` is read: the catalog also
// publishes cost and modalities, and neither changes what a request may ask for.
type catalogModel struct {
	Limit catalogLimit `json:"limit"`
}

// catalogProvider keys its models by id, unlike declaredProvider's array. The
// asymmetry is the upstream formats', not a choice made here.
type catalogProvider struct {
	Models map[string]catalogModel `json:"models"`
}

// checkModelLimits compares every model declared in ai/pi/models.json against the
// provider catalog and reports each disagreement with the severity its direction
// earns.
func checkModelLimits(sys *System, cfg *Config, rep *Report) {
	rep.Section("Model limit drift")

	if cfg.RepoDir == "" {
		// Without a checkout there is nothing a finding could be committed
		// against. Skipping says that; passing would claim the comparison ran.
		rep.Skip("not inside a checkout, so " + piModelsRelPath + " cannot be resolved")
		return
	}
	decl, ok := readDeclaration(cfg.RepoDir, rep)
	if !ok {
		return
	}
	cat, ok := readCatalog(sys.home(), rep)
	if !ok {
		return
	}

	compared, findings := 0, 0
	// Deterministic order: a report whose lines move between runs cannot be
	// diffed, and this one is read by a human comparing two machines.
	provNames := make([]string, 0, len(decl.Providers))
	for p := range decl.Providers {
		provNames = append(provNames, p)
	}
	sort.Strings(provNames)

	for _, prov := range provNames {
		for _, m := range decl.Providers[prov].Models {
			entry, ok := lookupCatalogModel(cat, prov, m.ID)
			if !ok {
				// A model the catalog does not carry is legitimate — a private
				// or brand-new one. Silent, like an unrouted catalog row next
				// door: absence of published truth is not a finding.
				continue
			}
			fields, found := compareModel(rep, prov, m, entry.Limit)
			if fields > 0 {
				compared++
			}
			findings += found
		}
	}

	if compared == 0 {
		rep.Skip("no declared model has a published limit in the cached catalog, so nothing was compared")
		return
	}
	if findings == 0 {
		rep.Pass(fmt.Sprintf("%d models match the provider catalog", compared))
	}
}

// readDeclaration loads ai/pi/models.json from the checkout. It is REPO content,
// so failing to read it is a failure of the check: absent → SKIP (this checkout
// does not carry it), unreadable or unparseable → FAIL. An unreadable declaration
// used to WARN, and a WARN leaves doctor's exit at 0 — a gate reading the exit
// status accepted a declaration nobody had read (HARNESS-136 review, round 1).
func readDeclaration(repoDir string, rep *Report) (declaredCatalog, bool) {
	var decl declaredCatalog
	raw, err := os.ReadFile(filepath.Join(repoDir, piModelsRelPath))
	switch {
	case os.IsNotExist(err):
		rep.Skip(piModelsRelPath + " is not in this checkout")
		return decl, false
	case err != nil:
		rep.Fail(fmt.Sprintf("%s exists but cannot be read, so no model could be compared: %v", piModelsRelPath, err))
		return decl, false
	}
	if err := json.Unmarshal(raw, &decl); err != nil {
		// C15 again: an unparseable declaration is not an empty one, and both
		// would otherwise print "no drift".
		rep.Fail(fmt.Sprintf("%s is not valid JSON, so no model could be compared: %v", piModelsRelPath, err))
		return decl, false
	}
	return decl, true
}

// readCatalog loads opencode's cached models.dev catalog. It is a per-machine
// CACHE, not repo content, so its problems WARN: the declaration may be perfectly
// right and this machine simply unable to say so.
func readCatalog(home string, rep *Report) (map[string]catalogProvider, bool) {
	raw, err := os.ReadFile(filepath.Join(home, filepath.FromSlash(catalogRelPath)))
	switch {
	case os.IsNotExist(err):
		rep.Skip("opencode's model catalog is not cached on this machine (" +
			catalogRelPath + "); run opencode once to populate it")
		return nil, false
	case err != nil:
		rep.Warn(fmt.Sprintf("%s unreadable: %v", catalogRelPath, err))
		return nil, false
	}
	var cat map[string]catalogProvider
	if err := json.Unmarshal(raw, &cat); err != nil {
		rep.Warn(fmt.Sprintf("%s is not valid JSON, so nothing could be compared: %v", catalogRelPath, err))
		return nil, false
	}
	return cat, true
}

// compareModel reports each limit that disagrees with the catalog and returns how
// many limits were actually comparable and how many disagreed. A limit the catalog
// does not publish (0) is not a claim that it is zero, so it is neither compared
// nor counted — a model whose row publishes nothing was compared against nothing.
func compareModel(rep *Report, prov string, m declaredModel, lim catalogLimit) (fields, findings int) {
	for _, d := range []struct {
		field            string
		declared, actual int
	}{
		{"contextWindow", m.ContextWindow, lim.Context},
		{"maxTokens", m.MaxTokens, lim.Output},
	} {
		if d.actual == 0 {
			continue
		}
		fields++
		if d.declared == d.actual {
			continue
		}
		findings++
		if d.declared > d.actual {
			rep.Fail(fmt.Sprintf(
				"%s/%s %s is %d, above the provider's %d\n"+
					"    The provider rejects a request this config invites.",
				prov, m.ID, d.field, d.declared, d.actual))
			continue
		}
		rep.Warn(fmt.Sprintf(
			"%s/%s %s is %d, below the provider's %d\n"+
				"    Capability forfeited silently; nothing breaks.",
			prov, m.ID, d.field, d.declared, d.actual))
	}
	return fields, findings
}

// lookupCatalogModel resolves a declared model against the catalog, scoped BY
// PROVIDER — never by bare id.
//
// That scoping is the whole correctness of this check. The bare id `qwen3.8-flash`
// is published by `nan` (262144 context) and by a dozen other providers — alibaba,
// hyper, opencode-go, requesty… — at 1000000 or 1048576: an id-only match picks
// whichever the map iterates first and reports the honest declaration as 4x wrong,
// or blesses a genuinely wrong one. Measured while writing this check — the
// unscoped version claimed nan/qwen3.8-flash should be 1M.
//
// No id rewriting is needed: an aggregator model is written `vendor/model` in the
// declaration and keyed identically in the catalog. A provider the catalog does
// not carry resolves to nothing, which the caller treats as "no published truth"
// rather than as drift.
func lookupCatalogModel(cat map[string]catalogProvider, provider, id string) (catalogModel, bool) {
	if p, ok := cat[provider]; ok {
		if m, ok := p.Models[id]; ok {
			return m, true
		}
	}
	return catalogModel{}, false
}
