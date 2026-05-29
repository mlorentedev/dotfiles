---
id: "HARNESS-001-unified-cross-agent-harness"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-05-28"
tags: [spec, proposal, harness-001, epic, cross-agent, cross-os, deploy-engine, ruler]
template_version: "1.0"
---

# HARNESS-001-unified-cross-agent-harness

> **Epic / umbrella spec.** Owns the shared *deploy-engine core* and the cross-agent architecture; the four consumers (SDD-008, IDEAS-007, #156, #159) implement verticals on top and each carries its own spec + atomic PR. Tracker: GH [#162](https://github.com/mlorentedev/dotfiles/issues/162).

## Why

The no-attribution policy regressed **silently** (#156): nothing propagates `00_meta/patterns` rules into the instruction files agents actually read at session start, and agent harness defaults override them at runtime. That is not a one-off — it is the symptom of a missing pipeline. Today every agent artifact (instructions, enforced patterns, skills, MCP config) is deployed ad-hoc and per-agent (symlink here, mechanical copy there, hand-edit elsewhere), so the SSOT promise breaks without anyone noticing until the next incident. We need **one deploy engine** that compiles every agent artifact from a single declarative manifest, commits the generated output, and is guarded by CI against drift — so a rule added once is enforced everywhere, deterministically, across Claude / OpenCode / Antigravity (`agy`) / Copilot / Pi and across Linux / Windows. Vault: `10_projects/dotfiles/11-tasks.md` (HARNESS-001 epic).

## What

Observable behavior after the epic completes (engine first, then consumers):

1. **Declarative manifest** at repo root (`harness/manifest.json` — format TBD, see Risks) defines, per agent: `kind` (`native` AGENTS.md reader | `pointer`+overlay), source path(s), overlay file, and target deploy path.
2. **Cross-OS compiler** — `compile-harness.sh` + `compile-harness.ps1` read the manifest and emit per-agent artifacts (`~/.claude/CLAUDE.md`, repo `AGENTS.md` pointers, `.github/copilot-instructions.md`, `ai/agy/AGY.md`, opencode), each carrying a `<!-- GENERATED FROM <source> — DO NOT EDIT -->` marker. **Generated artifacts are committed** (so CI and vault-less machines work).
3. **Precedence concatenation** — shared `AGENTS.md` subset + per-agent overlay, applied in declared order. Pi/OpenCode consume `AGENTS.md` natively; Claude/agy/Copilot get pointer + overlay.
4. **Enforced patterns** — any `00_meta/patterns/*.md` with frontmatter `enforce: true` is injected as a generated `## Overrides of harness defaults` block into the deployed Claude + AGENTS instructions (attribution at minimum, the #156 case).
5. **CI drift + contradiction guard** — a workflow re-runs the compiler and fails if committed artifacts differ (hand-edit / forgot-to-recompile), or if a deployed instruction contradicts an enforced pattern.
6. **Consumers ride the engine** — skills (SDD-008), the `.agent/<id>/` registry (IDEAS-007), and work/personal SSOT mode (#159) deploy through the same compiler; no bespoke per-artifact deploy survives.
7. **Cross-OS + invariant parity** — bats (Linux) + PSScriptAnalyzer/Pester (Windows) cover the compiler; line-cap invariants hold post-compile (`ai/claude/CLAUDE.md` ≤ 80 lines, `ai/agy/AGY.md` ≤ 50 lines).

## Engine vs consumers

The umbrella owns the **engine**; everything else is a consumer with its own spec, ID, and atomic PR.

| Layer | Owner | Deliverable |
|---|---|---|
| **Engine core** | **this spec (HARNESS-001)** | manifest + schema, `compile-harness.{sh,ps1}`, source-markers, precedence concat, committed artifacts, drift/contradiction CI guard, ADR-013 |
| Skills vertical | SDD-008 (#141) | skill SSOT → per-agent skill artifacts *through the engine* (existing spec stays skill-specific) |
| Harness structure | IDEAS-007 (#103) | `.agent/<id>/INSTRUCT.md` + agent registry (`native` \| `pointer`) consumed by the manifest |
| Enforced patterns | #156 | `enforce: true` propagation + CI contradiction guard; restores no-attribution for real |
| Work/personal mode | #159 | switch knowledge SSOT (vault vs repo `docs/` + Project) by repo type, via the manifest |

## Out of scope

Siblings tracked separately — **not** children of this engine:

- Pi coding-agent install (#161, separate sprint).
- DevOps CLI-over-MCP + scoped permissions profile (#160).
- Cross-agent session-memory bridge (#117) and subagents-as-vault-artifact (#118).
- MCP-config manifest unification (#163) — adjacent; may reuse the manifest later, tracked on its own.
- The *content* of each consumer (skills migration, `.agent/` registry entries, work/personal switch logic) — those live in the child specs; this epic ships only the engine core + umbrella architecture.

## Risks / open questions

- **🟡 Manifest format** — TOML (`ruler` precedent) vs YAML vs JSON. **Lean: JSON**, parsed by `jq` (Linux) + native `ConvertFrom-Json` (Windows) — no new dependency, supports the nested per-agent records, boring tech (Decision Hierarchy #3/#4). Lock at PR-1 start.
- **🟡 PR decomposition** — the full engine exceeds the ~300-LOC atomic-PR limit. Mitigation: **PR-1 = attribution tracer-bullet** (the smallest payload — one `enforce: true` pattern — driven through the *entire* pipeline: manifest → marker → compile → commit → drift CI), then one atomic PR per consumer. Recorded in `tasks.md`.
- **🟢 CI-without-vault-access** — RESOLVED by committing generated artifacts (ruler model). The dotfiles CI has zero visibility into the private `mlorentedev/knowledge` repo; committed output is what CI diffs.
- **🟡 Bootstrap dependency** — the compiler needs the vault clone present for source content. Reuse SDD-008's setup-time preflight (exit 2 + actionable message) rather than inventing a new check.
- **🟡 Line-cap invariants** — `ai/claude/CLAUDE.md` ≤ 80 (opencode.bats #34) and `ai/agy/AGY.md` ≤ 50 MUST be asserted *post-compile*, or CI fails. Concatenation can silently blow these.
- **🔴 Tooling bug found this session (separate ticket)** — `init-spec.sh` vault-gate false-negatives inside git worktrees: `REPO_NAME=$(basename "$REPO_ROOT")` resolves to `<repo>-<feature>`, not the canonical repo name. Bypassed here with `--force-no-vault` (entry exists under canonical `dotfiles`). Needs its own BUG ticket + bats guard per the incident→guard rule.

## Acceptance criteria

Epic-level outcomes. Each child spec additionally carries its own testable ACs; these gate the *engine*.

- [ ] **AC1** — Declarative manifest exists with a schema; `compile-harness.{sh,ps1}` parse it and fail with a clear error on a malformed manifest.
- [ ] **AC2** — Linux and Windows compilers produce equivalent artifacts for the same manifest (cross-OS parity test green).
- [ ] **AC3** — Every generated artifact carries a `<!-- GENERATED FROM <source> — DO NOT EDIT -->` marker and is committed to the repo.
- [ ] **AC4** — CI drift guard fails when any generated artifact is hand-edited without recompiling (run-twice-and-diff).
- [ ] **AC5** — The no-attribution rule (#156 Exhibit A) appears in deployed `CLAUDE.md` + `AGENTS.md` as a generated override block, sourced from an `enforce: true` pattern — not hand-maintained.
- [ ] **AC6** — SDD-008 skills are produced by this engine; no bespoke skill-deploy path remains.
- [ ] **AC7** — Post-compile, `ai/claude/CLAUDE.md` ≤ 80 lines and `ai/agy/AGY.md` ≤ 50 lines (asserted in CI).
- [ ] **AC8** — ADR-013 recorded (supersedes adr-001 + adr-008, extends adr-012); child-spec index + dependency graph present in `tasks.md`.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` (HARNESS-001 epic anchor)
- Tracker: GH [#162](https://github.com/mlorentedev/dotfiles/issues/162) (epic); consumers [#141](https://github.com/mlorentedev/dotfiles/issues/141), [#103](https://github.com/mlorentedev/dotfiles/issues/103), [#156](https://github.com/mlorentedev/dotfiles/issues/156), [#159](https://github.com/mlorentedev/dotfiles/issues/159)
- ADR: `docs/adr/adr-013-agent-artifact-deploy-engine.md` (this epic); builds on `docs/adr/adr-012-deploy-strategy-copy-with-drift-assertion.md`, `adr-009-multi-agent-runtime.md`, `adr-010-agent-harness-parity.md`; supersedes `adr-001-skill-based-ai-workflow.md`, `adr-008-skills-ecosystem-overhaul.md`
- Child spec: `specs/SDD-008-skill-pipeline/`, `specs/IDEAS-007-cross-provider-agent-harness/`
- External: `ruler` (https://github.com/intellectronica/ruler) — manifest + generate-and-commit precedent; AGENTS.md standard (https://github.com/agentsmd/agents.md)
