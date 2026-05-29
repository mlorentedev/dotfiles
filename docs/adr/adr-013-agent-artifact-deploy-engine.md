---
id: "ADR-013-agent-artifact-deploy-engine"
type: adr
status: proposed
owner: manu
date: "2026-05-28"
supersedes: [adr-001-skill-based-ai-workflow, adr-008-skills-ecosystem-overhaul]
extends: [adr-012-deploy-strategy-copy-with-drift-assertion]
tags: [architecture, decision, deploy, harness, cross-agent, dotfiles, harness-001]
created: "2026-05-28"
---

# ADR-013: Agent-artifact deploy engine (manifest-driven, generate-and-commit)

> Proposed by epic HARNESS-001 (`specs/HARNESS-001-unified-cross-agent-harness/`, GH [#162](https://github.com/mlorentedev/dotfiles/issues/162)). **Status `proposed`** — supersession of ADR-001/008 takes effect only when this ADR is accepted (PR-1 of the engine lands).

## Status

Proposed

## Date

2026-05-28

## Context

The repo deploys several *kinds* of AI-agent artifact — behavioural instructions (`AGENTS.md`, `CLAUDE.md`, `AGY.md`, Copilot), enforced `00_meta/patterns` rules, skills/commands, and MCP config — and each kind currently has its own bespoke, per-agent deploy path. ADR-001 chose custom `SKILL.md` files deployed by a glob loop; ADR-008 overhauled the skills ecosystem; both are **skill-specific** and say nothing about instructions, patterns, or MCP config. ADR-012 fixed the *mechanism* (atomic copy + drift assertion, not symlinks) but not the *source-of-truth-to-N-targets fan-out*.

The gap surfaced as a real regression: the no-attribution policy (#156) silently disappeared from the deployed instructions because **nothing propagates `00_meta/patterns` into the files agents read**, and agent harness defaults override absent rules at runtime. This is structural, not a one-off — every artifact kind repeats the deploy decision ad-hoc, so the SSOT promise breaks without detection until the next incident. External precedent (`ruler`, AGENTS.md standard) converges on one answer: a declarative manifest that compiles a single source into per-agent artifacts and **commits** them.

## Decision

Adopt a single **agent-artifact deploy engine**:

1. **One declarative manifest** (lean format: JSON — parsed by `jq` on Linux + native `ConvertFrom-Json` on Windows, no new dependency) describes every agent: `kind` (`native` AGENTS.md reader | `pointer`+overlay), source path(s), overlay, and target deploy path.
2. **Cross-OS compiler** (`compile-harness.sh` + `compile-harness.ps1`) reads the manifest, concatenates shared + overlay by precedence, and emits per-agent artifacts. Every generated file carries a `<!-- GENERATED FROM <source> — DO NOT EDIT -->` source-marker.
3. **Generate-and-commit** (not symlink, not render-on-demand): generated artifacts are committed, building on ADR-012's copy-deploy. This is what lets CI verify them — the dotfiles CI has zero visibility into the private vault.
4. **Enforced patterns**: `00_meta/patterns/*.md` with frontmatter `enforce: true` compile into a generated override block in the deployed instructions (the #156 attribution case at minimum).
5. **CI drift + contradiction guard** fails on hand-edited artifacts (run-twice-and-diff) or instructions contradicting an enforced pattern.

Skills (SDD-008), the `.agent/<id>/` registry (IDEAS-007), and work/personal mode (#159) become **consumers** of this engine rather than independent deploy paths.

## Consequences

**Positive**

- One place defines the source→target contract; a rule added once is enforced across all agents and both OSes.
- Drift and silent-override regressions (the #156 class) become CI failures, not surprises.
- New agents are a manifest entry + (if `pointer`) an overlay, not a new bespoke deploy script.

**Negative / costs**

- Bootstrap depends on the vault clone being present for source content (reuse SDD-008's setup-time preflight).
- The compiler must assert line-cap invariants post-compile (`ai/claude/CLAUDE.md` ≤ 80, `ai/agy/AGY.md` ≤ 50) or concatenation can silently breach them.
- Full engine exceeds the ~300-LOC atomic-PR limit → delivered as a PR sequence (tracer-bullet first; see the epic `tasks.md`).

**Supersession**

- Supersedes ADR-001 (custom skills workflow) and ADR-008 (skills ecosystem overhaul): both are subsumed by the generalized engine, of which skills are one consumer. Effective on acceptance.
- Extends ADR-012 (copy + drift assertion): the engine is generate-and-commit on top of that copy substrate.
