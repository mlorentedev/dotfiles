---
id: "ADR-027-cross-harness-agent-pipeline"
type: adr
status: accepted
owner: manu
date: "2026-06-21"
supersedes: []
extends: [adr-013-agent-artifact-deploy-engine, adr-026-agent-config-ssot-topology]
tags: [architecture, decision, agents, harness, ssot, determinism, hooks, agent-pipeline, cross-agent]
created: "2026-06-21"
---

# ADR-027: Cross-harness agent pipeline — neutral definitions, hook-enforced consumption

> Fills the under-specified "Common agent library" row of ADR-026 with a concrete model: a provider-agnostic agent-definition library (SSOT in the vault) compiled to each harness's native agent format, where consumption of skills/patterns is made **deterministic via hooks** rather than left to the model's memory. Sibling to `pattern-cross-agent-skill-pipeline` (SDD-008); reuses the same record+render-per-target engine.

## Status

> **Amended by [ADR-032](adr-032-cross-harness-agent-orchestration.md) (2026-08-09).** Four
> changes, all recorded there: Copilot is promoted from the `catalog` render kind to a full
> `agent-md` target (its "no per-agent mechanism" premise is obsolete — Copilot CLI now has
> `.github/agents/*.agent.md`, `/fleet` parallel dispatch and automatic delegation); the neutral
> schema gains `surface` and `delegates_to`; `model-map.json` gains `pools` and `harnesses`
> blocks, because a tier→id map is not sufficient once more than one pool is reachable; and the
> orchestration policy §3 left open — who decides to fan out, and under what budget — is defined.
> Everything else below stands.

Accepted (model). The **authoring** of definitions and the **engine** (render + offline CI) are NOT gated. Only the **cross-machine auto-deploy** step inherits ADR-026's `knowledge#120` gate. Lands via the existing `HARNESS-001` deploy-engine epic — not a new infrastructure track.

## Date

2026-06-21

## Context

ADR-026 reserved a row for a "Common agent library (definitions, doctrine, templates shared across agents)" in `00_meta/agents`, but left it under-specified. On-disk reality, verified this session: `00_meta/agents` is **100% autonomous-Hermes machinery** (`doctrine.md`, `_template/` with `config-desired.yaml`/`cronjobs.yaml`/reconciliation scripts) — **zero invocable agent definitions**. Consumption is effectively zero across every harness: even Claude Code has no `~/.claude/agents/` directory. The skill pipeline (SDD-008) solved skill *distribution* four ways; agents have no equivalent.

But the deeper pain the user named is not distribution — it is **underutilization**: agents "remember to use" a skill/pattern/runbook only sometimes. That is a **determinism** problem. A library that is perfectly distributed but consumed probabilistically is still infrautilized. Harness-engineering's core lesson applies: *the gap between "usually" and "always" is where systems fail, and determinism lives in the harness (code that runs regardless of the model), not in the prompt.*

A multi-reference audit (Regla del 3) of how each harness defines a launchable agent established convergence: Claude Code, OpenCode, and Copilot all use **markdown = YAML frontmatter + body system-prompt** — structurally identical to a `SKILL.md`, so the existing render engine transfers. Two harnesses diverge: **Antigravity (agy)** moved to **pure YAML** (`name`/`description`/`instructions`); **pi** has no declarative agent format (subagents are a TypeScript Pi Package). Gemini CLI is dropped (deprecated in favor of Antigravity).

## Decision Drivers

- **Agnostic + reusable** definitions: one neutral SSOT, all provider distinctions confined to the compile layer.
- **Deterministic consumption**: enforcement in the harness (hooks), not the prompt.
- **Coexistence**: invocable personas and autonomous agents under one SSOT without mixing state.
- **Reuse the proven engine** (record + render-per-target), do not invent a parallel mechanism.
- **pi first-class** (heavily used), not a degraded catalog-only target.

## Decision

### 1. The `agents` render target — neutral definition is SSOT

A new manifest block `agents`, parallel to `skills`. SSOT = `00_meta/agents/definitions/<name>/AGENT.md`. The compile engine renders each record to each harness's native path. Render kinds:

- **`agent-md`** → markdown + native frontmatter (claude, opencode).
- **`agent-yaml`** → transpose to pure YAML, `instructions:` = body (agy / Antigravity).
- **`catalog`** → injected into an instructions file for harnesses with no per-agent mechanism (copilot; per-repo `.github/agents/` is a future option).

Adding a harness is a `deploy[]` row plus map entries, never a new loop.

### 2. Agnosticism contract — `capabilities` + `model` tier, not native names

The definition frontmatter is **100% provider-neutral**. The two fields that would otherwise leak provider specifics are abstracted and resolved at compile time by two maps:

- **`capabilities: [read, search, edit, shell, web]`** (neutral vocabulary) → `capability-map.json` → each harness's tool/permission names.
- **`model: top | mid | low`** (neutral tier, per ADR-011) → `model-map.json` → each harness's model id.

This is the one thing the skill pipeline did not need; it is what makes a single definition deployable everywhere without loss.

### 3. Determinism via hooks — a first-class pipeline output

A definition's `skills:` / `enforce:` declarations are **not decorative**. At deploy they **compile to each harness's deterministic hook primitive**, on three levels:

- **Presence (inject):** the role and its forced skills are placed in context every turn — Claude `SessionStart`, OpenCode `chat.system.transform`, pi `session_start`.
- **Action (gate):** progress is blocked unless the required step ran — Claude `PreToolUse`/`Stop` (exit 2), OpenCode `tool.execute.before`, pi `tool_call` (cf. the `pi-permission-system` precedent that already gates skills).
- **Invocation (dispatch):** a command/router spawns the role deterministically (`/review` → reviewer), not the model deciding if it remembers.

Cross-provider hook capability is confirmed for claude/opencode/pi; **agy hook capability is the one open verification item**. The declaration is agnostic; the hook emission is provider-specific (Claude JSON hooks vs OpenCode/pi TS plugins) and lives in the compile layer. A hook cannot be forgotten, bypassed, or reasoned around — this is the mechanism that converts an infrautilized library into one that is impossible to skip.

### 4. Coexistence by `kind` — invocable vs autonomous

- **`kind: invocable`** — a persona launched on demand by a session. Definition lives in `definitions/<name>/`; it has no `80_agents/` state.
- **`kind: autonomous`** — a long-running, self-reconciling agent (`steward` genre). Its **persona** is cataloged as a `definitions/<name>/AGENT.md` entry that **points at** its private live state in `80_agents/<agent>/`. The state is never duplicated into the catalog.

Today's instance: `hermes-nan` (runs on NaN) becomes a `kind: autonomous` catalog entry. New agents of either kind are added the same way. Boundary rule: **definition (what it is) in `definitions/`; live state (memory/crons/config) in `80_agents/`**.

### 5. pi is first-class via one reusable adapter

pi has no declarative agent format, so a single reusable **`pi-agents` Pi Package** (dotfiles `ai/pi/`) scans the rendered `~/.pi/agent/agents/*.md`, parses the neutral frontmatter, and registers each as a pi subagent (capabilities → pi allowlist; Single/Parallel/Chain). The definitions stay agnostic markdown; the pi-specific glue is one isolated adapter — not per-agent TypeScript. This keeps pi exactly as first-class as claude/opencode.

### 6. Roster derived from inventory, not invented

The set of agents is derived from clustering the actual `00_meta/{skills,patterns,runbooks}` surface by **workflow phase** (decide → plan → build → verify → crystallize → ship), single-word names: `architect`, `planner`, `builder`, `reviewer`, `curator`, `shipper`, plus the autonomous `steward`. SSOT for the roster: `00_meta/agents/ROSTER.md`. Each invocable role bundles ≥3 skills (no single-skill wrapper).

## Edit workflow (the practical contract)

| I want to change… | I edit… | How it reaches the runtime |
|---|---|---|
| What an agent **is** (persona, forced skills, capabilities) | `00_meta/agents/definitions/<name>/AGENT.md` (neutral) | engine renders + deploys to each harness's native agent path |
| A neutral capability → native tool mapping | dotfiles `harness/capability-map.json` | engine resolves at render |
| A neutral model tier → native id | dotfiles `harness/model-map.json` | engine resolves at render |
| An **autonomous** agent's live state | vault `80_agents/<agent>/` | read via Hive (it cannot reach dotfiles) |

## Reconciliation with prior decisions

- **Extends ADR-026.** Materializes the "Common agent library" row with a concrete dual-paradigm model; does not change directionality (consumers read the committed render, repos stay self-contained).
- **Extends ADR-013.** The `agents` target is one more manifest-governed surface with generate-and-commit records; reuses the ENGINE-001 engine and the offline-CI drift discipline.
- **Sibling to `pattern-cross-agent-skill-pipeline`.** Same record+render-per-target shape; the agent pipeline adds the capability/model maps and the hook-emission step.
- **Consistent with HARNESS-032 / `knowledge#499`** (`00_meta` agnosticism): definitions are agent-agnostic content under `00_meta/agents`, not per-agent `00_meta/ai/<agent>` carve-outs, so no guard exception is created.
- **Not a contradiction of the autonomous doctrine** (`00_meta/agents/doctrine.md`): autonomous agents keep their reconciliation loop and `80_agents/` write-zone; this ADR adds the invocable genre and a shared catalog, it does not retire the autonomous machinery.

## Consequences

**Positive**

- Determinism kills the underutilization class: forced skills are present + gated by hooks, not remembered.
- One neutral SSOT per agent; provider distinctions are isolated in two maps + render kinds.
- The proven skill-pipeline engine is reused; new harness = one row + map entries.
- pi is first-class; autonomous and invocable agents coexist without duplicated state.

**Negative / debt**

- New render kinds, including a non-trivial **MD→YAML transpose** for agy.
- The engine grows a **hook-emission** responsibility (per-provider hook config), a new surface to maintain and drift-check.
- **agy hook capability unverified** — open item; until confirmed, agy gets presence-level enforcement only.
- A third drift axis (definition ↔ render-commit) on top of ADR-026's two; needs the same ADR-012-style assertion on the committed render.
- Library consolidation (patterns→skills, merges, deprecations) is deferred to an evidence-based pass run by `curator` after deploy — intentionally not a blocker.

## Implementation gate

`knowledge#120` (reliable cross-machine vault sync) gates **only the cross-machine auto-deploy** step. The following are explicitly **not** gated and proceed now:

1. Author the neutral schema + the first definitions (`curator` dogfood, `hermes-nan` autonomous catalog entry).
2. Extend the manifest with the `agents` block + `capability-map.json` + `model-map.json`; teach `compile-harness.sh` the new render kinds and hook emission; extend `compile-harness.bats`.
3. Build the `pi-agents` adapter.
4. Run the 4-level validation (offline CI → on-machine load per harness → e2e dogfood of forced-skill hooks → `/adversarial-review`).

## References

- ADR-011 (model-tier policy), ADR-013 (manifest + generate-and-commit deploy engine), ADR-026 (agent-config SSOT topology — extended here), ADR-012 (copy + drift assertion).
- `pattern-cross-agent-skill-pipeline` (sibling engine), `00_meta/agents/ROSTER.md` (roster SSOT), `00_meta/agents/doctrine.md` (autonomous genre), HARNESS-032 / `knowledge#499` (`00_meta` agnosticism), `knowledge#120` (vault sync — deploy gate), `HARNESS-001` epic (deploy engine).
- Architecture session 2026-06-21 (this decision): per-harness agent-format audit (claude/opencode/copilot frontmatter-convergent; agy YAML; pi adapter), determinism-via-hooks (harness-engineering / loop-engineering), workflow-phase roster derivation.
