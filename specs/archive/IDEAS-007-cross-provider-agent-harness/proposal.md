---
id: "IDEAS-007-cross-provider-agent-harness"
type: spec
status: archived
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
---

# IDEAS-007-cross-provider-agent-harness

> **Outcome (2026-06-01): reconciled, not implemented.** The 4-layer design below was already realised by ADR-009/010 + the `ai/<provider>/` overlay structure that shipped after this proposal was written. The registry + runtime-discovery layers are rejected as YAGNI (no consumer). See [`reconciliation.md`](./reconciliation.md) and [`audit.json`](./audit.json) — the proposal text is retained verbatim as the historical record.
>
> **Naming**: file lives at `<repo>/specs/IDEAS-007-cross-provider-agent-harness/proposal.md`. `IDEAS-007-cross-provider-agent-harness` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: ([#103](https://github.com/mlorentedev/dotfiles/issues/103)) — Promote `.agent/<provider-id>/INSTRUCT.md` design from GH issue to `specs/IDEAS-007-cross-provider-agent-harness/`. **Steps:** (a) `init-spec.sh IDEAS-007-cross-provider-agent-harness`; (b) fill proposal from issue body (4-layer architecture); (c) resolve 5 open questions (discovery mechanism, backwards compat existing CLAUDE.md+AGENTS.md, co-location vs global, STATE.md format YAML/TOML/JSON, MVS rules); (d) link issue #103 ↔ spec; (e) comment on #103 noting promotion. **Esfuerzo:** ~2h (decision-heavy, low LOC). **Tier:** P0.5, post-SDD-008. **Depends_on:** SDD-008 (skill pipeline foundation), vault PATTERN-001 (lifecycle formalization). -->

Today's dual-file convention (`AGENTS.md` + `CLAUDE.md`) is ambiguous: `AGENTS.md` is the cross-provider SSOT, yet `CLAUDE.md` ships rules that reference Claude-specific MCP/skills/hooks and are not portable. As OpenCode, Antigravity, Copilot, and Codex enter the runtime, the current shape forces each agent to either load instructions it can't honor or skip the file entirely. The four-layer design (cross-provider AGENTS.md + per-provider `.agent/<id>/INSTRUCT.md` + global provider registry + migration path) draws the right boundary — proposed at GH #103 and reviewed publicly with substantive critique from two external contributors (see References).

## What

Promote GH #103's design into a vault-grounded spec with five resolved decisions:

1. **Discovery mechanism** — env var (`AGENT_HARNESS_PROVIDER=opencode`) vs runtime-detection (binary name → provider). Proposal recommends **runtime-detection** for low-friction, with env-var override.
2. **Backwards compatibility** — existing `AGENTS.md` + `CLAUDE.md` continue working during migration. `.agent/claude-code/INSTRUCT.md` is added; `CLAUDE.md` slims down to a pointer.
3. **Co-location vs global** — `.agent/` lives **per-repo** (co-located) for project rules; `~/.config/agent-harness/` for the global registry of provider capabilities. Both, not either-or.
4. **STATE.md format** — **YAML** for human-readability + native multi-line strings; JSON-schema validation at runtime.
5. **Minimal viable subset** — initial cross-cutting rules: identity, decision hierarchy, security halts, response protocol. Provider-specific tooling lives in `INSTRUCT.md`.

Phase 1 audit (per public-review feedback) adopts a **three-bucket classification**: `maps_directly`, `maps_weakly`, `requires_manual_validation` — making the migration safer than a binary shared/extension split. To partially address @m13v's "splitting moves the rot" critique without taking on instrumentation cost, the audit also records a qualitative `fires_observed` column (`yes` / `no` / `unknown`) per rule — based on manual review of recent sessions, NOT telemetry. This catches dead lines without coupling the spec to MEMORY-001.

## Out of scope

- **Instrumenting AGENTS.md line attribution** (per @m13v's critique) — see R-X below; treated as an open question with pragmatic-rejection default.
- **Migrating other repos** — dotfiles is the proof-of-concept. Rollout to `mlorentedev/*` repos is a follow-up batch (separate IDEAS-007b or per-repo specs).
- **Bringing other agent maintainers into the design loop** — design is opinionated; consensus-seeking with OpenCode/Antigravity teams may follow, but it does not block this PR.
- **STATE.md ingestion across sessions** — that's MEMORY-001's territory; this spec only fixes the INSTRUCTION boundary, not the MEMORY boundary.

## Risks / open questions

- **R1 (incorporated from @unitedideas)**: Some Claude-only rules (MCP env+secret refs, hooks, lifecycle, per-agent restrictions) do NOT map to a clean shared/extension split. Phase 1 audit MUST use the three-bucket classification. Without it, the rollout flattens nuance into noise.
- **R2 (incorporated from @m13v)**: Cross-provider SSOT is not validated by usage telemetry — no signal on which lines of `AGENTS.md` actually fire per turn. Three defensible answers:
  - **(a) Add Phase 0**: instrument session-end hook (via MEMORY-001) to log which AGENTS.md lines were quoted/referenced over 50 sessions; refactor with attribution data in hand. Cost: probably equal to or greater than IDEAS-007 itself.
  - **(b) Pragmatic-rejection**: dotfiles is a personal toolkit, not a team product; the refactor will surface line-fire patterns empirically as agents diverge. Accept the lack of telemetry; document the limit.
  - **(c) Qualitative audit (default for this spec)**: during Phase 1, manually annotate each rule with `fires_observed: yes/no/unknown` based on recent session review. No instrumentation cost; catches ~80% of dead-line value at ~5% of (a)'s cost. Surfaces candidates for deletion before they get reorganized.
  - **Decision**: default to (c). Path (a) reserved for if (c) reveals enough ambiguity to justify telemetry investment — owned in the spec, not glossed over.
- **R3**: Backwards compat during migration. Existing Claude sessions must not break when `CLAUDE.md` slims down. Mitigation: ship the pointer (`CLAUDE.md` → `.agent/claude-code/INSTRUCT.md`) before removing content from `CLAUDE.md`.
- **R4**: Multi-machine sync. `.agent/` per-repo follows git; `~/.config/agent-harness/` syncs via dotfiles. Both work today.

## Acceptance criteria

- [ ] Five open questions answered (with rationale) in PR body.
- [ ] Phase 1 audit ships a CSV/JSON file with **four** columns: `rule`, `bucket (direct/weak/manual)`, `target_layer (AGENTS.md / .agent/<id>/INSTRUCT.md / registry)`, `fires_observed (yes/no/unknown)`.
- [ ] Audit explicitly classifies each sensitive category surfaced by @unitedideas, with rationale:
  - **MCP env+secret refs** → reference-only, never copy values; stay in `AGENTS.md` as references; secrets resolved at runtime via `load-secrets.sh`.
  - **Claude Code hooks + lifecycle** → Claude-only, move to `.agent/claude-code/INSTRUCT.md`.
  - **Per-agent restrictions** (e.g. allowed-tools whitelists) → registry-level metadata in `~/.config/agent-harness/`.
  - **Session/memory artifacts** (handoffs, MEMORY.md content) → context, NOT instructions; documented as "load as context only, never quote as rule" in the registry schema.
- [ ] Migration report references @unitedideas's checklist shape: <https://bringyour.ai/codex-import-checklist.json>.
- [ ] R2 telemetry decision documented (path a, b, or c chosen).
- [ ] `.agent/claude-code/INSTRUCT.md` exists in dotfiles repo (Layer 2 proof-of-concept).
- [ ] `CLAUDE.md` slims down to a pointer with backwards-compat preserved. **Smoke test (MUST pass before merge):** (1) launch fresh Claude Code session in dotfiles repo; (2) verify SessionStart hook injects vault health + claude-mem context (grep `vault_health` + `claude-mem-context` in transcript); (3) verify `agy` and `opencode` CLIs spawn without error; (4) verify `~/.claude/CLAUDE.md` → `AGENTS.md` pointer resolves via existing `read AGENTS.md FIRST` instruction. Any of the 4 failing = blocks merge.
- [ ] Bats covers the discovery mechanism (env-var override + runtime-detection).

## References

- Vault: `10_projects/dotfiles/11-tasks.md` → IDEAS-007-promote-to-spec (GH #103).
- GH: <https://github.com/mlorentedev/dotfiles/issues/103>.
- **Public review feedback (incorporated)**:
  - @unitedideas (2026-05-27): three-bucket classification + boundary cases (hooks, MCP env refs, per-agent restrictions, memory artifacts). Reference essay: <https://bringyour.ai/agents-md-claude-md>. Checklist JSON shape: <https://bringyour.ai/codex-import-checklist.json>.
  - @m13v (2026-05-27): "splitting by provider just moves the rot" — calls for per-line attribution audit before refactor. Folded into R2.
- Dependencies: SDD-008 (skill pipeline foundation), vault ADR-010 (multi-agent runtime), vault PATTERN-001 (lifecycle formalization).
- Sister specs: AGENTS-001 (subagent SSOT), MEMORY-001 (cross-agent memory bridge).

<!-- archived 2026-06-02 — PR: https://github.com/mlorentedev/dotfiles/pull/208 -->
