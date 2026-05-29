---
id: adr-011-model-tier-policy
type: adr
status: active
created: "2026-05-19"
---

# ADR-011: Model Tier Policy (Task-Aware Model Selection Across Agents)

## Status

Accepted — 2026-05-19. Shipped in PR [#60](https://github.com/mlorentedev/dotfiles/pull/60) (AI-019).

## Context

The user works through a wide cognitive load range in the same week — deep root-cause debugging, mechanical skill ports, single-file typos, architectural decisions, syntax lookups. Every modern AI coding agent supports multiple model tiers (Opus / Sonnet / Haiku, Pro / Flash / Flash-Lite, V4-Pro / Qwen3.6-Plus / V4-Flash, etc.), each calibrated for a different reasoning-vs-cost trade-off. Without an explicit policy, agents default to "whatever model the user picked last," producing two consistent failure modes:

- **Top-tier flagship burning tokens on trivial work.** A 5-line README typo running on Opus or DeepSeek V4 Pro is ~10× the cost of the same task on Sonnet or V4 Flash, with no measurable quality gain.
- **Mid-tier flailing on hard work.** A concurrency bug or schema redesign on Sonnet often produces shallow analysis the user has to redo on Opus anyway — *worse* than starting at Top tier.

`AGENTS.md` already encoded the **task-class branching** behaviour ("Operating Mode: Adaptive — Low Cognitive Load → Fast Lane; High Cognitive Load → Socratic Guardrail") via [ADR-009](adr-009-multi-agent-runtime.md). But the rule said nothing about **which model corresponds to each class**, leaving the cost/capability axis unmanaged. [ADR-010](adr-010-agent-harness-parity.md) (per-primitive parity matrix) similarly addressed *what agents can do*, not *how to pick the right model intensity*.

## Decision

Introduce a **provider-agnostic 3-tier policy** in `AGENTS.md` (canonical SSOT for cross-agent rules) with concrete model identifiers in per-agent overlay files. Three pieces:

### 1. Tier mapping (provider-agnostic)

| Tier | Task class |
|---|---|
| **Top** | Hard debugging, root-cause analysis, distributed systems, concurrency, security review, schema design, novel architecture, complex refactors with semantic risk |
| **Mid** | Mechanical refactors, single-file fixes, documentation, boilerplate, regex / JSON parsing, test scaffolding, comment-only edits |
| **Low** | Syntax lookups, quick questions, autocomplete, one-line transforms |

The classifier is the *task*, not the *language* or the *file*. A 3-line edit to a distributed cache invalidation algorithm is Top tier. A 200-line port of test fixtures is Mid tier.

### 2. "Propose, don't force" heuristic

Agents SHOULD detect class shifts mid-session and **propose** a tier change in a single sentence. The user decides. Auto-switching is explicitly forbidden because:

- It breaks the user's cost expectations (silent escalation = bill shock).
- It breaks the user's capability expectations (silent de-escalation = mysterious quality drop).
- The proposal *is* the value — making the trade-off visible at the moment of context shift.

### 3. Per-agent overlay files own model identifiers

Per [ADR-009](adr-009-multi-agent-runtime.md), `AGENTS.md` is the canonical SSOT; per-agent files in `ai/<agent>/` are pointer-style overlays (≤70 lines post AI-013, raised to ≤80 in AI-019 for CLAUDE.md). Each overlay gains a **Model Tier** subsection listing the literal model IDs that map to Top / Mid / Low for that provider:

- `ai/claude/CLAUDE.md`: `claude-opus-4-7` / `claude-sonnet-4-6` / `claude-haiku-4-5-20251001`. Selection via subagent frontmatter `model:` or main-session `/model` slash.
- `ai/opencode/opencode.jsonc`: `opencode-go/deepseek-v4-pro` / `opencode-go/qwen3.6-plus` / `opencode-go/deepseek-v4-flash`. Selection via TUI `/models` picker or `qq` / `qf` wrappers.
- `ai/gemini/GEMINI.md`: `gemini-2.5-pro` / `gemini-2.5-flash` / `gemini-2.5-flash-lite` (TBD pending empirical validation on next Gemini session). Selection via per-prompt `--model` flag.
- `ai/copilot/copilot-instructions.md`: TBD pending AI-017 / AI-018 (Windows-empirical work on Copilot CLI v2 schema).

Rotation surface is isolated to overlay files. When a provider sunsets a model or releases a new flagship, the overlay gets a one-line edit; `AGENTS.md` stays stable.

## Alternatives considered

- **Single global default with manual overrides.** Rejected: this is the current implicit behaviour and is exactly the failure mode the ADR addresses.
- **Auto-switching script.** Rejected: would require running a heuristic against every prompt to decide model, adding latency and surprising users with silent escalations / de-escalations. Cost / capability expectations dominate the value of an in-line proposal.
- **Pattern in `00_meta/patterns/pattern-model-tier-policy.md` instead of ADR.** Premature: the pattern is currently dotfiles-only. Promote to `00_meta/` when a second project (kubelab, etc.) adopts the same policy.
- **Token-budget tracking in `pattern-token-budgeting.md`.** Different concern. Tier policy is about model *capability* match; budgeting is about *consumption* limits. Both could co-exist later; budgeting deferred.

## Consequences

### Positive

- **Cross-agent consistency.** Six agents (Claude, OpenCode, Copilot, Gemini, Codex, Cursor) all read `AGENTS.md` and apply the same tier semantics. Behaviour stays uniform when the user context-switches between agents.
- **Rotation cost is bounded.** Provider model rotations are ~quarterly; the overlay-only-edit pattern means rotation cost ≈ one-line edit per overlay × 4 overlays = 4 lines per rotation.
- **Cost / capability axis becomes deliberate.** The user makes the trade-off consciously, not by accident-of-last-pick.
- **Pairs cleanly with existing rules.** "Operating Mode: Adaptive" (Identity section) and "Competence Retention Protocol" already gate behaviour by cognitive load; this ADR extends the same axis to model selection.

### Negative

- **Coordination cost on new agent additions.** A new agent (say, Codex v2) needs both an overlay file AND a tier mapping. Increases onboarding from 1 file to 2.
- **TBD entries for Gemini and Copilot** until empirical validation. Mitigated by explicit `(TBD — verify on next session)` annotations that won't rot silently — they self-flag for the next contributor.
- **Risk of over-proposing.** Bad heuristics → frequent "should we switch?" interruptions. Mitigated by language in `AGENTS.md` ("propose on class *shift*, not periodically") and by the user always being free to ignore.

### Neutral / future

- **`opencode.jsonc` uses native `//` comments** instead of a `_modelTierComment` JSON key. JSONC native comments are diff-friendly and match the existing convention in that file (schema URL + 6 prior comment blocks). If OpenCode ever rejects line comments (unlikely; the file has used them since AI-011), fall back to a companion `ai/opencode/MODEL_TIERS.md`.
- **Quantitative validation pending.** No measurement of token savings yet. After ~1 month of use, retrospectively analyse claude-mem timeline + OpenCode usage logs to confirm the policy is actually changing behaviour and saving cost.

## References

- Implementation PR: [#60](https://github.com/mlorentedev/dotfiles/pull/60) (`feat(sdd,agents): ship SDD-003 spec-gate + AI-019 model-tier policy`)
- Spec: `specs/archive/AI-019-model-tier-policy/{proposal,tasks,verification}.md` (in repo)
- Sibling ADRs: [ADR-009](adr-009-multi-agent-runtime.md) (AGENTS.md SSOT), [ADR-010](adr-010-agent-harness-parity.md) (per-primitive parity matrix)
- Behavioural precedent: `AGENTS.md` "Operating Mode" + "Competence Retention Protocol" sections.
- Architecture context: [AUDIT-004](dotfiles-architecture-map.md) for where each per-agent overlay lives in the deploy graph.
