---
id: "AGENTS-001-subagents-as-vault-artifact"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
---

# AGENTS-001-subagents-as-vault-artifact

> **Naming**: file lives at `<repo>/specs/AGENTS-001-subagents-as-vault-artifact/proposal.md`. `AGENTS-001-subagents-as-vault-artifact` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: ([#118](https://github.com/mlorentedev/dotfiles/issues/118)) — Add `vault/00_meta/agents/<name>.md` as SSOT artifact type for subagent definitions (Claude-only deploy per ADR-010 gap-3 decision). **Scope:** (a) extend canonical schema with `agent-frontmatter.schema.json` (fields: `name`, `description`, `model: opus|sonnet|haiku`, `tools: []`, `targets: [claude]`); (b) extend `render-all.sh` (SDD-008) with `render-claude-agents.sh` copying vault agents → `~/.claude/agents/` with provenance header; (c) author 2-3 initial subagents (`vault-curator`, `harness-auditor`, `iris-worker-evaluator`); (d) target manifest honors `targets: [claude]` only (ADR-010 explicit). **Esfuerzo:** ~3-5h. **Tier:** P1, post-SDD-008. **Depends_on:** SDD-008. -->

Claude Code subagents (`vault-curator`, `harness-auditor`, etc.) are today defined as ad-hoc markdown files in `~/.claude/agents/`, divorced from the vault SSOT discipline that governs everything else. ADR-010 gap-3 surfaced this: subagents lack a canonical source-of-truth type, so they drift between sessions and aren't tracked across `setup-{linux.sh,windows.ps1}` deploys. SDD-008's skill-pipeline pattern (compile-once, deploy-everywhere) maps naturally — agents become another vault artifact type rendered to `~/.claude/agents/` at setup time. Claude-only because OpenCode/Antigravity have no equivalent subagent runtime (ADR-010 explicit).

## What

Three deliverables landing as one atomic PR (post-SDD-008):

1. **Canonical schema** — new `agent-frontmatter.schema.json` with required fields: `name`, `description`, `model: opus|sonnet|haiku`, `tools: []`, `targets: [claude]` (single-target for now).
2. **Renderer extension** — `scripts/skills/render-claude-agents.sh` (reusing SDD-008's render-all pipeline) copies `vault/00_meta/agents/*.md` → `~/.claude/agents/` with provenance header (`# Generated from vault/00_meta/agents/<name>.md — DO NOT EDIT`). Both setup scripts invoke it.
3. **Initial corpus** — author 2-3 subagents to validate end-to-end: `vault-curator`, `harness-auditor`, `iris-worker-evaluator`. Each has frontmatter + body following the schema.

CI drift gate (mirroring SDD-008's pattern): `render-claude-agents.sh --check` fails when deployed copy diverges from vault source.

## Out of scope

- **Multi-target subagents** — `targets: [claude, opencode]` reserved for future spec when other providers add subagent runtimes (none today).
- **Subagent invocation tooling** — invocation flow remains Claude Code's native `Agent` tool; this spec is artifact lifecycle only.
- **Vault migration of existing subagents** — if `~/.claude/agents/` has hand-written entries today, the migration is a separate AGENTS-002.
- **Inter-subagent orchestration** — fan-out / pipeline patterns are runtime concerns, not artifact concerns.

## Risks / open questions

- **R1**: SDD-008 must land first (the render-all dispatcher is its deliverable). Spec implementation cannot start until that ships.
- **R2**: Schema rigidity vs flexibility. Should `tools: []` enforce a registry (whitelist) or accept arbitrary names? Recommendation: free-form for v1; tighten if abuse emerges.
- **R3**: Provenance header may conflict with Claude Code's expected frontmatter parser. Verify header lives OUTSIDE the YAML block, as a body-leading comment, so it doesn't break Claude's parsing.
- **R4**: Drift gate behavior on missing vault source. `render-all --check` should FAIL if `~/.claude/agents/foo.md` exists without a corresponding `vault/00_meta/agents/foo.md` — orphan policy enforced.

## Acceptance criteria

- [ ] `agent-frontmatter.schema.json` exists in vault `00_meta/`.
- [ ] `scripts/skills/render-claude-agents.sh` exists and integrates with SDD-008's render-all dispatcher.
- [ ] 2-3 subagent markdown files exist in `vault/00_meta/agents/`.
- [ ] `setup-{linux.sh,windows.ps1}` invoke the renderer; `~/.claude/agents/` populates with provenance header.
- [ ] CI drift gate: `render-all --check` exits non-zero on vault-source drift OR orphan in deployed dir.
- [ ] Bats covers happy-path + drift-gate.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` → AGENTS-001 (GH #118).
- GH: <https://github.com/mlorentedev/dotfiles/issues/118>.
- Dependency: SDD-008 (skill pipeline, GH #141).
- ADR: ADR-010 gap-3 (subagents lack vault SSOT).
