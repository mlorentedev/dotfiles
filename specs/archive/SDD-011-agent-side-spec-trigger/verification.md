---
tags: [spec, verification, templates]
created: "2026-05-13"
---

# Verification - SDD-011-agent-side-spec-trigger

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof.

- [x] **AC1** (rule in vault SSOT) -> `00_meta/skills/spec/SKILL.md` `## Agent-Side Activation Rule` with the three sub-parts (Checks the agent runs / How to phrase the proposal / When NOT to propose). Committed to vault `master` as `e2931be`.
- [x] **AC2** (always-on trigger + pointer, no duplication) -> `AGENTS.md` Discipline Gate: `**Proactive proposal (agent-side).**` paragraph pointing to the skill's `"Agent-Side Activation Rule"` with explicit `do not duplicate it here`. `grep` confirms the trigger sentence + skill pointer; the checklist is NOT copied into AGENTS.md.
- [x] **AC3** (deterministic propagation, no drift) -> `compile-harness.sh --refresh` regenerated `harness/skills/spec/SKILL.md` (+42, line 32 `## Agent-Side Activation Rule`); `git diff --stat` minimal (AGENTS.md + record only, no other skill records or enforced blocks); `compile-harness.sh --check` -> `[check] OK: no harness drift`.
- [x] **AC4** (deploys to agents) -> `tests/skills-pipeline.bats` test `SDD-011: deployed /spec carries the Agent-Side Activation Rule (claude + opencode)` greps the heading in both `~/.claude/skills/spec/SKILL.md` and `~/.config/opencode/commands/spec.md` after `--deploy`.

## Test status

- Test suite: `~/.local/bin/bats tests/skills-pipeline.bats` -> `1..6` all `ok` (the SDD-011 assert is test #2).
- Offline drift gate: `./scripts/compile-harness.sh --check` -> `[check] OK: no harness drift`.
- Manual smoke: `--refresh` from an isolated vault worktree pinned at the source commit produced a minimal `AGENTS.md (+2)` + `harness/skills/spec/SKILL.md (+42)` diff; no unintended record/enforced-block churn.
- No regressions: yes — the 5 pre-existing skills-pipeline asserts still pass alongside the new one.

## Decisions made during implementation

- **Two-surface design.** The detailed rule is single-sourced in the `/spec` skill (loads on invocation); `AGENTS.md` (always-on) carries only the trigger + a pointer. Rationale: the proactive proposal must fire BEFORE `/spec` is invoked, so the trigger has to live in the always-loaded instructions, while SSOT of the detail stays in the skill. Pointer says `do not duplicate it here` to lock the boundary.
- **Test is a regression guard, not strict red-green.** The change is prose-in-a-skill, not algorithmic; the bats assert guards against a future record regen from a SKILL.md missing the section (incident -> guard).
- **Vault commit recovery.** The vault commit first landed on a parallel feature branch (`rfd-001/pdf-modifier-mcp`) because another session had staged work checked out; recovered by cherry-picking onto `master` (`e2931be`) from an isolated worktree, without touching the active checkout. The record for this PR was regenerated from an isolated detached worktree at the source commit, so the dotfiles diff never depended on the vault checkout state.

## Promotion candidates

- [ ] Lesson for `90-lessons.md`? **yes** — "Don't commit to a shared vault that has another session's staged work checked out; use an isolated `git worktree` to read/operate on a specific commit." (Capture at archive.)
- [ ] ADR-worthy decision? **no** — the two-surface (always-on trigger + on-demand detail) pattern is already implied by ENGINE-001/SDD-008; no new ADR.
- [ ] New pattern candidate for `00_meta/patterns/`? **no** — folds into the existing `pattern-cross-agent-skill-pipeline.md`.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/SDD-011-agent-side-spec-trigger/` -> `specs/archive/SDD-011-agent-side-spec-trigger/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (the lesson)
