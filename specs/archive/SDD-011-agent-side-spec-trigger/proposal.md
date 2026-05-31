---
id: "SDD-011-agent-side-spec-trigger"
type: spec
status: archived
created: "2026-05-31"
tags: [spec, proposal, sdd-008-followup, agent-behavior, skill-content, harness-001]
template_version: "1.0"
---

# SDD-011-agent-side-spec-trigger

> **Naming**: file lives at `<repo>/specs/SDD-011-agent-side-spec-trigger/proposal.md`. `SDD-011-agent-side-spec-trigger` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: *(P1, queued 2026-05-30, cross-agent, SDD-required, spec: `specs/SDD-011-agent-side-spec-trigger/`)*: Deferred sub-task of [[SDD-008-skill-pipeline]] (proposal §What #5 — NOT shipped in PR #179, which delivered only the deploy pipeline). Make the agent apply the Skip-SDD heuristic *itself* and proactively propose `/spec init <id>` when it detects a non-trivial change mid-conversation (a Discipline Gate trigger is met), instead of waiting for the human to ask. **Two-surface design (SSOT-preserving):** (1) a detailed "Agent-Side Activation Rule" section in `vault/00_meta/skills/spec/SKILL.md` — the checks the agent runs, the proposal format, and when NOT to propose; (2) a short always-on trigger + pointer in `AGENTS.md` Discipline Gate so the behavior can fire *before* `/spec` is invoked (the skill body only loads on invocation, so the always-on file must carry the trigger). Propagated to the committed record via `compile-harness.sh --refresh`; `--check` green. **Done when:** record reflects the new section, a bats assert proves it deploys, PR references archived SDD-008 + epic HARNESS-001 #162. **Anti-scope:** empirical validation that agents apply the heuristic correctly across N sessions is observational (already carved out in SDD-008 Out-of-scope). Part of HARNESS-001 #162. -->

Today the SDD Discipline Gate is enforced *reactively*: the rules live in `AGENTS.md`, CI fails post-hoc on un-spec'd PRs, and the `/spec` skill only runs once a human types `/spec init`. The agent never volunteers a spec — so the human carries the whole burden of noticing "this change is now non-trivial; it needs a spec" at exactly the right moment. That moment is easy to miss mid-flow, which is precisely how scope creep and un-specced 200-LOC PRs happen. SDD-008's proposal (§What #5) called for closing this gap by making the agent apply the Skip-SDD heuristic itself and propose `/spec init` proactively; SDD-008 shipped only the deploy pipeline, leaving this behavior unbuilt. This spec ships it.

## What

Concrete, observable behavior changes after this PR:

1. **The `/spec` skill gains an "Agent-Side Activation Rule"** (in the vault SSOT `00_meta/skills/spec/SKILL.md`). It defines: (a) the checks the agent runs against the Discipline Gate when work is being scoped in conversation, (b) the exact shape of a proactive proposal (state which trigger fired + the checks run + the suggested `/spec init <id>`), and (c) explicit "when NOT to propose" guardrails (trivial change, already inside an active spec, user already declined this session).
2. **`AGENTS.md` Discipline Gate carries a short always-on trigger** instructing agents to apply that heuristic proactively and surface the proposal *before* `/spec` is invoked, pointing at the skill section for the detail (no rule body duplicated — the always-on file is a trigger + pointer, the skill is the SSOT of the detail).
3. **The change propagates deterministically** to every agent through the existing SDD-008 pipeline: `compile-harness.sh --refresh` regenerates the committed record `harness/skills/spec/SKILL.md`; `--deploy` renders it to each agent's native path; `--check` stays green (offline drift gate).
4. **Observable as agent-initiated spec proposals** in transcripts where the human previously had to initiate — the agent says "this looks like a Discipline-Gate trigger (≈X LOC / public contract); I ran the Skip-SDD checks; propose `/spec init <id>`?" unprompted.

## Out of scope

Things this PR explicitly does NOT include.

- **Empirical validation** that agents actually apply the heuristic correctly across N real sessions — observational work, already carved out in SDD-008's Out-of-scope.
- **Redefining the Skip-SDD criteria / trigger thresholds** — those remain owned by `AGENTS.md` Discipline Gate + `pattern-spec-driven-development.md`. This PR references them; it does not change the thresholds.
- **New PowerShell/Windows logic** — the rule is content carried by the existing cross-OS pipeline; no `Deploy-SkillRecord` or setup-script change is required. Windows consumes the regenerated record like any other skill.

## Risks / open questions

- **Over-triggering (annoyance) vs under-triggering (uselessness).** A proactive proposer that fires on every two-line edit is worse than silence. **Mitigation (in-scope):** the rule ships explicit "when NOT to propose" guardrails and a once-per-thread debounce (don't re-propose for the same change after a decline). Not a blocker.
- **Two-surface drift.** The trigger (AGENTS.md, always-on) and the detail (skill, on-demand) could drift. **Mitigation:** AGENTS.md carries only a one-line trigger + pointer, never a copy of the checks — SSOT stays in the skill. Verified by an acceptance criterion asserting the pointer exists and the detailed checklist lives only in the skill.
- **Record counted by spec-gate.** `harness/skills/*` is not matched by the gate's `*generated*` exclusion, so the regenerated record counts toward diff LOC. **Resolved:** this spec folder satisfies the gate regardless of LOC (SPEC_TOUCHED=1).

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] **AC1 — Rule in vault SSOT**: `vault/00_meta/skills/spec/SKILL.md` contains an `## Agent-Side Activation Rule` section that names (a) the checks the agent runs, (b) the proposal format, (c) when NOT to propose. **Verify:** grep the three sub-parts in the skill file.
- [ ] **AC2 — Always-on trigger + pointer, no duplication**: `AGENTS.md` Discipline Gate instructs agents to proactively propose `/spec init` and points to the skill's Agent-Side Activation Rule; the detailed checklist is NOT duplicated in AGENTS.md. **Verify:** grep AGENTS.md for the proactive-trigger sentence + skill pointer.
- [ ] **AC3 — Deterministic propagation, no drift**: after `compile-harness.sh --refresh`, the committed record `harness/skills/spec/SKILL.md` contains `Agent-Side Activation Rule`, and `compile-harness.sh --check` exits 0. **Verify:** grep record + `--check` exit code.
- [ ] **AC4 — Deploys to agents**: after `compile-harness.sh --deploy` to a throwaway HOME, the rendered Claude skill and OpenCode command for `spec` both carry the `Agent-Side Activation Rule` heading. **Verify:** bats assert in `tests/skills-pipeline.bats`.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` (SDD-011 backlog entry)
- Parent (archived): `specs/archive/SDD-008-skill-pipeline/proposal.md` §What #5 (original trigger wording) + §Out-of-scope (empirical-validation carve-out)
- Related patterns: `00_meta/patterns/pattern-cross-agent-skill-pipeline.md` (the deploy pipeline), `00_meta/patterns/pattern-spec-driven-development.md` (the Discipline Gate this extends)
- Epic: HARNESS-001 unified cross-agent harness ([#162](https://github.com/mlorentedev/dotfiles/issues/162))

<!-- archived 2026-05-31 — PR: https://github.com/mlorentedev/dotfiles/pull/181 -->
