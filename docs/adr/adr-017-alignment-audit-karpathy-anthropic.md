---
id: "ADR-017-alignment-audit-karpathy-anthropic"
type: adr
status: accepted
owner: manu
date: "2026-06-06"
extends: [adr-009-multi-agent-runtime]
tags: [architecture, decision, philosophy, alignment, autonomy, agents, karpathy, anthropic]
created: "2026-06-06"
---

# ADR-017: Philosophy-alignment audit vs Karpathy and Anthropic guidance

> First architecture-session of a recurring idea-adoption loop. Audits the repo's operating philosophy (`AGENTS.md`, Standing Orders, the `harness/skills/` layer, the dual-memory model) against three external references and records which gaps to close, defer, or reject. Seeds the loop's rejection list.

## Status

Accepted

## Date

2026-06-06

## Context

A recurring "idea adoption" working mode was established: external ideas/articles are triaged into lanes (0 reject / 1 light reversible / 2 full architecture session), decided serially, with evidence gathered in parallel and the "rule of three" (>=2 independent references) required for any cross-cutting adoption. The first cluster (Philosophy) audits whether the repo's flows align with external thought leadership.

Three references were read and contrasted against the current `AGENTS.md` digest:

- **Karpathy** — No Priors podcast, "Code Agents, AutoResearch, and the Loopy Era of AI" (early 2026). *Confidence: medium.* The originating link (a third-party tweet) was unfetchable (X returned 402); the underlying interview was triangulated from the transcript. Deltas resting on Karpathy alone therefore carry **single-source + medium-confidence** weight.
- **Anthropic** — "How Anthropic enables self-service data analytics with Claude" (engineering blog). Confidence: high.
- **Anthropic** — "When AI builds itself" (recursive self-improvement essay, Favaro/Clark, 2026). Confidence: high.

## Findings

**Strong alignment (no action).** The repo already embodies the core of all three sources in ~11 places, including: orchestrator-over-worker with explicit per-agent ownership (worktree isolation + the `dispatching-parallel-agents` skill); roles-as-durable-markdown (`AGENTS.md` + `harness/skills/*` = "tune behaviour by editing config, not re-prompting"); gate-autonomy-on-cheap-verification (shellcheck + bats + spec-gate, `verification-before-completion`); incident -> guard ("red-team thyself"); adversarial review before archive; thin-router-over-curated-docs (`pattern-loader` ~= Anthropic's "structure beats raw access"); SSOT dedup; colocation + CI integrity to fight staleness; human-on-taste / AI-on-perspiration (the Competence Retention Protocol); hand-authored definitions; and "invest in instructions/memory/tools, not a smarter base model".

**Gaps considered.** Seven gaps surfaced; six were distilled into candidate deltas (below). The seventh — ablation-at-PR-granularity (empirically proving each knowledge artifact moves agent accuracy) — is deliberately skipped as net-negative ceremony at solo scale.

## Decision

| # | Delta | Sources | Decision |
|---|---|---|---|
| 1 | **Autonomy-escalation triggers** — stop and surface on repeated failure / taste decision / unreviewable diff | Karpathy + RSI (2) | **Adopt now** — new `AGENTS.md` "Autonomy Boundaries" clause |
| 2 | Provenance footer when answering from vault/claude-mem | analytics (1, weak) | **Defer (trigger)** — reopen if the vault becomes multi-author or a staleness incident occurs |
| 3 | Loop-contract block for unattended/parallel runs | Karpathy (1, med-conf) | **Bind to the agent-harness cluster** — design it alongside the orchestrator, not in a vacuum |
| 4 | Permission / blast-radius scoping for new agents | Karpathy (1, med-conf) | **Bind to the agent-harness cluster** |
| 5 | Self-acceleration telemetry in `/insights` | RSI + Karpathy (2) | **Defer (trigger)** — needs metric design to avoid the vanity-volume trap; reopen once a real orchestrator exists to measure |
| 6 | Make adversarial review a *blanket-mandatory* gate | RSI + analytics (2) | **Reject** — capability already exists, scoped to high-stakes SDD changes |

Only delta 1 changes a file this cycle (a short clause). Deltas 3-5 become tracked follow-up issues; delta 2 is deferred; delta 6 is rejected and recorded here as rejection-list precedent.

**On delta 6 (reject rationale).** The principle (an AI review pass catches real bugs) is sound and two sources support it — but the repo *already* pairs the `adversarial-review` skill with the `/spec archive` lock, scoped to high-stakes changes. Anthropic's own data (+6% accuracy for +32% tokens, +72% latency) plus the minimize-new-ceremony and atomic-PR constraints argue against forcing it on *every* change. The scoped version that already exists is the correct cost/benefit point; making it blanket-mandatory would add cost without earning it.

## Consequences

- **Positive:** the one strong, source-converged, low-cost gap (escalation) is closed; the rest are correctly sequenced (3-4 to where they will be designed well) or deferred with explicit reopen triggers, honouring the minimize-new-ceremony constraint. The audit itself is recorded so it is not re-run.
- **Negative / debt:** deltas 3-5 stay open; if the agent-harness work stalls, the autonomy guardrails (loop-contract, permission-scoping) remain unwritten while parallel-agent use grows — revisit at the first genuinely unattended run.
- **Rejection-list precedent:** delta 6 establishes that "the capability already exists, scoped to high-stakes cases" is sufficient grounds to reject a "make it blanket-mandatory" proposal. Do not re-debate without new evidence that the scoped version is insufficient.

## Reopen Triggers

- **Delta 2 (provenance footer):** the vault gains a second author, OR a documented incident where a stale vault/memory claim produced a wrong answer.
- **Delta 5 (telemetry):** a real orchestrator / unattended-run exists whose acceleration is worth measuring, AND the 2-3 early-warning metrics are chosen (candidates: AI-authored-change ratio, days-to-stale, autonomy-escalations per week).
- **Deltas 3-4:** start of the agent-harness architecture work.

## References

- `docs/adr/adr-009-multi-agent-runtime.md` (AGENTS.md as cross-agent SSOT).
- Karpathy, No Priors: "Code Agents, AutoResearch, and the Loopy Era of AI" (2026).
- Anthropic: "How Anthropic enables self-service data analytics with Claude".
- Anthropic Institute: "When AI builds itself" (recursive self-improvement, 2026).
