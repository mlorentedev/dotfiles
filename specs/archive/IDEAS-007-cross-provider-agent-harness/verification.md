---
tags: [spec, verification]
created: "2026-05-13"
completed: "2026-06-01"
---

# Verification - IDEAS-007-cross-provider-agent-harness

> Outcome: **reconciled, not implemented.** The 4-layer design was already realised via ADR-009/010 + the `ai/<provider>/` overlay structure. See [`reconciliation.md`](./reconciliation.md) for the full mapping and [`audit.json`](./audit.json) for the rule-by-rule Phase-1 audit. This file maps each acceptance criterion to its evidence.

## Evidence

| Acceptance criterion | Evidence |
|---|---|
| 1. Five open questions answered | `audit.json` → `decisions` (Q1–Q5 + R2). |
| 2. Phase-1 audit (4 columns) | `audit.json` → `rules` (24 rule-groups, `AGENTS.md` + `ai/claude/CLAUDE.md`). |
| 3. Sensitive categories classified | `audit.json` → `sensitive_categories` (4 categories, each with verdict + rationale). |
| 4. Migration report | `reconciliation.md` — migration already executed via ADR-009/010; this doc is the report. |
| 5. R2 telemetry decision | Path (c) qualitative — `audit.json` → `decisions.r2_telemetry`; path (a) not triggered. |
| 6. `.agent/claude-code/INSTRUCT.md` exists | Satisfied by `ai/claude/CLAUDE.md` (shipped L2 equivalent). |
| 7. CLAUDE.md slim-to-pointer + smoke test | `ai/claude/CLAUDE.md:3` is the pointer; deploy-verified at `setup-linux.sh:498`. No cutover → smoke test risk structurally absent. |
| 8. Bats for discovery mechanism | N/A — discovery mechanism rejected (no consumer); nothing to test. |

## Test status

- Test suite: not run — **zero production/code/config change** in this PR (spec artifacts only). Pre-commit hooks (secret scan + bats) gate the commit regardless.
- Manual smoke test: not required — no cutover. The slim-pointer `CLAUDE.md` and SessionStart hook are already in production every session (this very session loaded them).
- No regressions: yes — no executable or deployed file touched.

## Decisions made during implementation

- **Reconcile, don't re-implement.** Verify-before-act revealed the spec's design had shipped by other means. Implementing it literally would have manufactured debt (a no-caller discovery mechanism; a rename of a deployed convention). Chose the evidence-backed close instead.
- **Rejected L3 registry + runtime discovery as YAGNI** — each provider self-discovers its native config file; a detector would have zero callers.
- **Spun off the one genuine scalability win** (data-driven `setup-linux.sh` provider-deploy manifest) as a deferred `REFACTOR-*` follow-up rather than bundling it (Atomic PRs). Trigger: 6th provider added, or deploy blocks drift.
- **Flagged but did not fix** the stale `claude-opus-4-7` model id (`ai/claude/CLAUDE.md:71`) — correct placement, wrong value; a separate model-overlay bump, not this spec's concern.

## Promotion candidates

- [x] Lesson for `90-lessons.md`? **yes** — "A backlog spec can be obsoleted by a *later* ADR that solves its problem by other means; reconcile + close with evidence rather than implement-as-written." Reinforces `pattern-verify-against-source-of-truth`. (Captured via Hive `capture_lesson`.)
- [ ] ADR-worthy decision? **no** — ADR-009/010 already own the architecture; this is a confirmation, not a new decision.
- [ ] New pattern candidate? **no** — already covered by `pattern-verify-against-source-of-truth`; this is another instance, not a new pattern.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived` (via `archive-spec.sh`)
- [x] Folder moved: `specs/IDEAS-007-…/` → `specs/archive/IDEAS-007-…/`
- [x] Backlog entry in vault `11-tasks.md` ticked with PR link (via Hive)
- [x] GH #103 closed with pointer to this reconciliation
- [x] Deferred follow-up (data-driven provider deploy) recorded in `reconciliation.md` + vault backlog
