---
id: "IDEAS-007-reconciliation"
type: spec
status: done
created: "2026-06-01"
tags: [spec, reconciliation, verify-before-act]
---

# IDEAS-007 — Reconciliation against shipped reality

> **Verdict: the 4-layer design is already realised — by other means.** IDEAS-007 (GH #103, filed 2026-05-27) proposed a cross-provider harness. Between then and now the repo shipped **ADR-009** (AGENTS.md as cross-agent SSOT), **ADR-010** (harness parity), and the **`ai/<provider>/` overlay structure**. Those collectively implement Layers 1–2 and satisfy the CLAUDE.md-slim criterion. The remaining layers (registry + runtime discovery) are rejected as YAGNI — no runtime consumer exists. This document is the evidence-backed close-out; the machine-readable rule-by-rule audit is the sibling [`audit.json`](./audit.json).

## Why this is a reconciliation, not an implementation

This spec triggered the repo's recurring **verify-before-act** lesson (`pattern-verify-against-source-of-truth`): a backlog item described a problem that a *later* architectural decision had already solved. Implementing the spec literally would have meant renaming a deployed, working convention and building a mechanism with no caller — i.e. *manufacturing* technical debt in the name of "completing the spec." The honest, professional outcome is to map the spec onto reality, reject the dead layers with rationale, and capture the one genuine improvement as a separate ticket.

## Layer mapping (proposed → shipped)

| IDEAS-007 layer (as written) | Shipped reality on `main` | Status |
|---|---|---|
| **L1** Cross-provider contract `AGENTS.md` | `AGENTS.md` (450 LOC), canonical SSOT per ADR-009 | ✅ done |
| **L2** Per-provider `.agent/<id>/INSTRUCT.md` | `ai/<provider>/<FILE>`: `ai/claude/CLAUDE.md`, `ai/agy/AGY.md`, `ai/copilot/copilot-instructions.md`, `ai/hermes/AGENTS.md`; OpenCode reads `AGENTS.md` natively | ✅ done (different name, same architecture) |
| **L3** Global registry `~/.config/agent-harness/` | — | ❌ rejected (YAGNI, no consumer) |
| **L4** Migration path | Already executed via ADR-009/010; per-agent files are thin pointers that delegate to `AGENTS.md` | ➖ moot |

**Naming equivalence:** `ai/claude/CLAUDE.md` *is* the `.agent/claude-code/INSTRUCT.md` the spec asked for. It opens with a pointer to `AGENTS.md` and carries only Claude-specific extensions (claude-mem, skills, MEMORY.md, model tier). The `ai/<provider>/` location is wired into `setup-linux.sh` deploy + healthcheck verification — it is the *better* realisation because it is deployed and drift-checked.

## Acceptance criteria — line by line

| # | Criterion (from `proposal.md`) | Disposition |
|---|---|---|
| 1 | Five open questions answered with rationale | ✅ Resolved in [`audit.json`](./audit.json) `decisions` (Q1 discovery, Q2 backwards-compat, Q3 co-location, Q4 STATE.md = out of scope/MEMORY-001, Q5 MVS). |
| 2 | Phase-1 audit: `rule \| bucket \| target_layer \| fires_observed` | ✅ Delivered — [`audit.json`](./audit.json), 24 rule-groups across `AGENTS.md` + `ai/claude/CLAUDE.md`. |
| 3 | Classify each sensitive category (@unitedideas) | ✅ [`audit.json`](./audit.json) `sensitive_categories`: MCP refs (reference-only, stay), hooks/lifecycle (Claude-only, isolated), per-agent restrictions (registry — rejected), memory artifacts (context-not-instruction). |
| 4 | Migration report referencing the checklist shape | ➖ Migration already executed (ADR-009/010); this document **is** the migration report. The external checklist URL is preserved as a reference in `proposal.md`. |
| 5 | R2 telemetry decision documented | ✅ Path **(c)** qualitative audit adopted; `fires_observed` annotated from session review, no instrumentation. Path (a) telemetry **not** triggered — the split is already validated correct, so there is nothing to instrument-then-refactor. |
| 6 | `.agent/claude-code/INSTRUCT.md` exists (L2 PoC) | ✅ Satisfied by `ai/claude/CLAUDE.md` (the shipped equivalent). No new file — that would duplicate a working overlay. |
| 7 | `CLAUDE.md` slims to a pointer + 4-pt smoke test | ✅ Already a pointer (`ai/claude/CLAUDE.md:3`), deploy-verified (`setup-linux.sh:498`). No cutover → the smoke test's risk (breaking a fresh session) is structurally absent; existing SessionStart hook + pointer already in production every session. |
| 8 | Bats covers a runtime discovery mechanism | ❌ **Rejected** — no discovery mechanism is built, because none has a consumer (see below). Nothing to test. |

## Rejected as YAGNI (with rationale)

Per the Decision Hierarchy (*Explicit > Implicit*, *boring tech*) and the Atomic-PRs rule:

1. **L3 registry** (`~/.config/agent-harness/providers/*.yaml`) — no runtime consumer; every provider already self-discovers its own native config file.
2. **Runtime discovery mechanism** (binary-name → provider detector + bats) — **zero callers**. Each agent loads its own deployed file natively:
   - Claude Code → `~/.claude/CLAUDE.md` (`setup-linux.sh:497`)
   - OpenCode → `~/.config/opencode/AGENTS.md` (`setup-linux.sh:619`)
   - Copilot → `~/.copilot/copilot-instructions.md` (`setup-linux.sh:702`)
   - AGY → `$GEMINI_HOME/AGY.md` (`setup-linux.sh:452`)

   Building a detector nothing invokes is dead code — the exact debt this spec was meant to avoid.
3. **Renaming `ai/<provider>/` → `.agent/<id>/INSTRUCT.md`** — pure churn on a deployed convention (≈6 `setup-linux.sh` sites + healthcheck), zero functional gain.

## Deferred follow-up — the one genuine scalability win

The audit surfaced real DRY debt, but **not** where the spec placed it. `setup-linux.sh` carries ~5 near-identical per-provider deploy blocks (copy file → `grep`-verify pointer). A **data-driven provider manifest** (`provider | src | dst | verify-string`, looped once) would collapse ~80–100 LOC to ~30 + a manifest, making "add a provider" a one-line change.

- **Why not here:** it is a `setup-linux.sh` refactor — a *different logical change* than this instruction-boundary reconciliation. Bundling it would violate Atomic PRs.
- **Disposition:** capture as its own `REFACTOR-*` spec when actioned. **Defer-with-trigger** (YAGNI): do it when the 6th provider is added or the blocks drift. 5 providers today, stable.
- **Bonus:** the stale `claude-opus-4-7` id (`ai/claude/CLAUDE.md:71`, flagged in the audit) could fold into the same manifest if model tiers move into it; otherwise it is a trivial standalone bump.

## Net change in this PR

Documentation + spec artifacts only — **no production code, config, or deploy change**:

- `specs/IDEAS-007-…/audit.json` — Phase-1 rule audit (new)
- `specs/IDEAS-007-…/reconciliation.md` — this document (new)
- `specs/IDEAS-007-…/verification.md` — evidence map (filled)
- spec archived to `specs/archive/…`; GH #103 closed with a pointer here.
