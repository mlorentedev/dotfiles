---
tags: [spec, verification, templates]
created: "2026-08-08"
---

# Verification - HARNESS-056

## Evidence

- [x] AC1 (authored under a stable anchor, registered) -> `features.json` `HARNESS-056-f1`, jq assertion green
- [x] AC2 (reaches every declared surface) -> tests `HARNESS-056: every enforced target injects the definition-of-done` and `...: the injected region reaches the committed instruction files`
- [x] AC3 (the skill executes it) -> `## The Closing Pass` in `verification-before-completion`, asserted by test `HARNESS-056: the checklist binds the standing orders instead of restating them`
- [x] AC4 (caps and drift) -> test `HARNESS-056: the compact doctrine payload carries it and stays under its cap`; `compile-harness.sh --check` reports no drift; `ai/claude/CLAUDE.md` is 78 lines against its 100-line cap
- [x] AC5 (binds, does not restate) -> same test as AC3, plus the explicit precedence rule in the skill: when the checklist and a standing order disagree, the standing order wins

## Test status

- Test suite: `bats tests/compile-harness.bats` -> 44/44; `bats tests/skills-pipeline.bats` -> 18/18 (14 before, 4 added)
- Lint: `shellcheck scripts/compile-harness.sh` -> clean
- Manual smoke test: `--refresh` then `--check` against the real vault; the region appears in `AGENTS.md` (241 -> 253 lines) and `ai/claude/CLAUDE.md` (66 -> 78), and in the deployed compact payload for both capped surfaces.
- No regressions in existing test suite: yes

## Decisions made during implementation

- **Extended `verification-before-completion` rather than creating a closing skill.** Its trigger is already the exact moment being enforced; a second skill would split that moment in two, and the half nobody invokes is the half that rots.
- **The checklist binds, it does not restate.** Each item points at the standing order it enforces and the skill states that the standing order wins any disagreement — a paraphrase would become a competing source of truth and drift from the original.
- **A skip must be named.** Any item may legitimately not apply; the failure mode is silence, so the rule is that a skip is a stated decision.
- **Reverted a whole-file JSON reformat.** Editing the manifest with a JSON dumper rewrote 97 lines of unrelated formatting; redone surgically for a 5-line diff. A diff that large hides the change it contains.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? no — the transferable lesson (doctrine that no code binds does not happen) is the ticket's own premise and is already recorded in the pattern.
- [ ] ADR-worthy decision? no — implements existing doctrine rather than changing a decision.
- [x] New pattern candidate for `00_meta/patterns/`? **update, not new** — `pattern-change-lifecycle.md` now carries the Definition of Done itself, which is the promotion.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-056/` -> `specs/archive/HARNESS-056/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
