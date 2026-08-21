---
id: "ADR-035-model-map-routing-registry"
type: adr
status: accepted
owner: manu
date: "2026-08-21"
supersedes: []
extends: [adr-032-cross-harness-agent-orchestration, adr-027-cross-harness-agent-pipeline]
issue: mlorentedev/dotfiles#1124
tags: [architecture, decision, agents, orchestration, harness, routing, registry, budget, forge]
created: "2026-08-21"
---

# ADR-035: `model-map.json` — the shape of the routing registry, its budget contract, and its forge boundary

> ADR-032 specified *what* the routing map contains and was accepted on 2026-08-09. Twelve days
> later the file still does not exist, and building it surfaced four questions ADR-032 does not
> answer: which ticket owns it, what **shape** it takes among the registries this repo already
> runs, whether its declared budget is enforced or advisory, and whether a forge concept may ever
> enter it. This ADR answers those four and nothing else. **It does not re-open ADR-032.**

## Status

Accepted.

## Date

2026-08-21

## Context

### What exists, measured 2026-08-21

| Layer | State |
|---|---|
| **Definition** — what an agent is, how it renders per harness | Exists. `harness/agents/`, schema, manifest, render pipeline — **one agent populated** (`curator`) |
| **Policy** — pools, tiers, chains, concurrency | **Specified in ADR-032 §3, never built.** `harness/model-map.json` and `harness/capability-map.json` both absent |
| **Execution** — who launches work and collects it | **Nothing.** `dotf agent` is `unknown command` |

### The four questions

**1. Two open issues claim the same filename with different semantics.** #560 (HARNESS-044,
from ADR-027 §2) defines `model-map.json` as a *render-time* map: neutral tier `top|mid|low` → each
harness's model id. #1124 (from ADR-032 §3) defines it as a *dispatch-time* policy map: pools,
harnesses, tiers, chains, concurrency. Three independent sources already record that these are
different shapes — ADR-032 §3's own text (*"`tiers` resolves at compile time and feeds each
harness's native render"*), the `$comment` in `harness/reviewer-pool.json` (*"H-044's model-map.json
maps a neutral tier to a provider per agent — a different shape"*), and both issue bodies.

**2. This repo already runs four declarative registries, and they follow two different idioms.**

| Registry | `$comment` | `version` | JSON Schema | Go loader | bats guard | CI/script |
|---|---|---|---|---|---|---|
| `reviewer-pool.json` | yes | — | **no** | yes | yes | yes |
| `review-attestation.json` | yes | — | **no** | yes | yes | yes |
| `triggers.json` | — | yes | **no** | yes | **no** | **no** |
| `manifest.json` | — | yes | **no** | yes | yes | yes |

*Policy registries* carry a `$comment` array holding the rationale and the measured evidence, and
no `version`. *Engine configs* carry `version` and no `$comment`. **No registry has a JSON Schema,
and no registry has a `dotf doctor` check** — although the repo does schematise agent and skill
*frontmatter*. `triggers.json` has a loader and no guard at all.

**3. A declared budget that nothing decrements is advisory.** ADR-032 §3 already declares
`concurrency`, `rpm` and `reserve_interactive`. By this repository's own frame — every mitigation
that held was enforced, every one that failed was advisory — a config stating a reserve while
nothing reads it has the same shape as `ignore_pr_source_branches` sitting inert.

**4. Multi-forge is a stated goal and retrofitting is expensive.** The owner intends repos on
GitLab and Gitea. Five of thirteen `dotf` nouns are GitHub-coupled, with thirty `gh` invocations in
shell and ten GitHub-Actions-only workflows.

### The evidence base, and what an adversarial review did to it

The orchestration evidence report (vault `40_resources/04-tools/`) was reviewed before this session
consumed it, by two non-Anthropic models from `harness/reviewer-pool.json` — `nan/deepseek-v4-flash`
and `agy/gemini-3.1-pro-high` — independently, same charge. **Both returned SOUND-WITH-GAPS** and
converged on three weaknesses. See vault `agent-orchestration-evidence-report-review`.

Three consequences bind this ADR:

- **The "Orca abstracts five forges" claim is downgraded to unverified** (graded Blocker). It reads
  a hosted-review-capability check as a forge abstraction — which is the report's own central
  failure domain, committed by the report. **This ADR does not cite it, and does not import
  `agent-fleet-model-analysis.md` §7, which carries the same claim.**
- **The "emergent domain" claim is the weakest in the document** — confounded by single-author
  lineage across every repo in the corpus. So: *the domain-taxonomy evidence is weaker than the
  report states.* The workflow-phase axis is not re-opened here, because it rests on the
  concurrent-write serialisation argument and on subject domains measuring single-repo, not on
  emergence.
- **#1072's "once in sixteen"** is one measurement. It is cited below for **direction, not
  magnitude**; the map-not-director decision survives independently because a declarative map is
  also what C4 requires.

## Options Considered

### Shape of the registry

| Option | C1 honest | C4 enforced | C10 guard | C15 fails loud |
|---|---|---|---|---|
| **A** Policy-registry idiom (`$comment`, no `version`) | ok | ok | bats | **gap** |
| **B** Engine-config idiom (`version`, no `$comment`) | **gap** | ok | bats | **gap** |
| **C** Hybrid + JSON Schema + `dotf doctor` check | ok | ok | ok | **ok** |
| **D** Bare JSON, loader only | ok | **gap** | partial | **gap** |

**B fails C1** for a concrete reason: the `$comment` in `reviewer-pool.json` is where the reasoning
lives — it is the only record of why the provider-diverse fallback precedes the NaN sibling. A
routing file without that place loses ADR-032 §3's dense, measured rationale.

**A and D fail C15**, adopted the same day: an unreadable map must never be indistinguishable from
a permissive one. Nothing in this repo notices an absent registry today.

### Budget enforcement

Full enforcement was considered and rejected: `dotf` cannot see `qq`, the pi TUI, hive embeddings
or CI, so a counter claiming to cover them would state a guarantee it cannot keep.

## Decision

**1. `harness/model-map.json` is the dispatch-time policy registry of ADR-032 §3, and #1124 owns it
alone.** #560 is re-scoped to `harness/capability-map.json` — neutral capabilities → each harness's
tool and permission names — which ADR-032 does not cover and which remains needed.

**2. It takes shape C: both idioms, plus the two guards no registry here has yet.**

```jsonc
{
  "$comment": [ "why the order is the order, and what was measured" ],
  "version": 1,
  "pools":     { /* auth, concurrency, rpm, reserve_interactive, shared_with, probe */ },
  "harnesses": { /* pools[], render, spawn */ },
  "tiers":     { /* resolved at COMPILE time, feeds each harness's render */ },
  "chains":    { /* resolved at RUN time, read only by the level-2 dispatcher */ },
  "services":  { /* embeddings, rerank — non-chat endpoints, per ADR-032 §3 */ }
}
```

- **`$comment`** because the rationale is the artifact, and losing it costs more than the bytes.
- **`version`** because there are two consumer classes with different cadences — the render engine
  at compile time, the dispatcher at run time — and that is exactly the drift `version` exists for.
- **`harness/model-map.schema.json`**, making this the **first registry in the repo with a schema**.
- **A `dotf doctor` check** — present, parseable, schema-valid — making it the **first registry with
  one**. This is not gold-plating; it is the only mechanism by which C15 can hold, since a map that
  cannot be read must fail loudly rather than resolve to a permissive default.

**This sets a precedent the other four registries do not meet, and that is recorded rather than
hidden.** Whether they follow is out of scope here.

**3. Budget enforcement splits in two levels.**

- **Level 1 — declaration.** `pools` carries `concurrency`, `rpm`, `reserve_interactive` and
  `shared_with`. Cheap, and it makes the budget legible, which is more than exists today.
- **Level 2 — enforcement only where `dotf` is the launcher.** The semaphore decrements for work
  `dotf` dispatches and promises nothing about consumers it cannot see. **The honest guarantee is
  `dotf` alone will never be the cause of exhaustion** — never "exhaustion will not happen."
- **When the counter cannot be read: fail closed and loudly**, matching `dotf pr triage-queue`,
  which exits non-zero both when work is pending and when the question could not be answered,
  precisely so an unanswerable state is never mistaken for a clear one.
- **Falsifiable trigger to reopen full enforcement:** a third measured quota exhaustion caused by
  consumers `dotf` does not launch, within a 30-day window.

**4. Routing is a declared map, never a director model.** The orchestration layer is the dumbest
reliable component in the system. #1072 measures the counter-case and is cited for direction only.

**5. The forge boundary.**

- **Decided now:** the orchestration schema never carries a forge concept. Verified 2026-08-21 —
  pools are providers, probes are `env:`/`bin:`, and the `.github/agents/` paths in ADR-032 are
  Copilot's own directory convention, which is portable to a repo hosted anywhere. **This records
  and protects a property the schema already has.**
- **Deferred with a trigger:** the evidence-layer adapter — the thirty `gh` invocations and the five
  coupled `dotf` nouns. **Trigger to reopen: the first repository living outside GitHub.** The
  separation that matters is policy from forge: *"a review must exist"* is forge-free; *knowing
  where to look for one* is not (C14).

**6. `chains` and `reviewer-pool.json` stay separate, and this ADR states why so it is not
re-derived.** Both are ordered fallback lists. The pool is read once per spec and ranks
*independence* above availability; `chains` is read per dispatch and ranks availability. **Falsifiable
merge trigger: a third consumer needing an ordered fallback chain.**

### Amendments to ADR-032 §3

Recorded here rather than edited into an accepted ADR:

- **`harnesses.codex.pools: ["codex"]` references a `codex` pool that `pools` never declares.**
  A dangling reference in the reference schema. The schema must reject it — **and the resolution is
  deletion, not declaration**: the owner confirmed on 2026-08-21 that codex is no longer used, so
  declaring the missing pool would have made a route to nowhere validate. A `~/.codex` directory on
  the account is a leftover, not evidence of a live runtime. (`harness/manifest.json` still carries
  a codex deploy target; that is separate cleanup.)
- **`pools.openrouter` describes a provider deleted upstream** and must not be carried into the
  built file.

## Consequences

### Positive

- The two-issue collision is resolved before either is built, rather than discovered at merge.
- `model-map.json` becomes the first registry that `dotf doctor` can report on, so an absent or
  malformed routing map is caught at diagnostic time rather than under dispatch load.
- The budget becomes legible at level 1 even before level 2 exists.
- The forge decision costs one sentence because the property was already true — the expensive half
  is deferred behind a trigger rather than designed speculatively.

### Negative

- `model-map.json` will be the only registry with a schema and a doctor check, so the repo carries
  two standards until the others catch up — or a decision is taken that they need not.
- Level-2 enforcement guarantees strictly less than it appears to. The wording above is deliberate
  and must survive into the implementation's help text and errors.

### Neutral

- The workflow-phase axis is unchanged and unre-opened; its evidence is weaker than the report
  claimed, and this ADR says so.
- `capability-map.json` remains unbuilt under a re-scoped #560.

## References

- ADR-032 §3 (the schema this shapes), ADR-027 §2 (#560's origin), ADR-011 (model tiers)
- Constraints C1, C4, C6, C9, C10, **C14**, **C15** — vault `10_projects/dotfiles/session-protocol.md`
- Vault `40_resources/04-tools/agent-orchestration-evidence-report-review` — the two adversarial
  reviews and what they downgrade
- `harness/reviewer-pool.json` `$comment` — the third independent record of the #560/#1124 collision
- Issues: #1124 (owns `model-map.json`), #560 (re-scoped to `capability-map.json`), epic #558, #1072
