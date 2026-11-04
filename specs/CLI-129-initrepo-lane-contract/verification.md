---
tags: [spec, verification, templates]
created: "2026-11-04"
id: "CLI-129-initrepo-lane-contract"
type: spec-verification
status: active
owner: manu
---

# Verification - CLI-129-initrepo-lane-contract

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

Filled during implementation. AC8 is already met: the kubelab rollout follow-up is #3154.

- [ ] AC1 -> commit `<hash>` / test `TestBootstrapLane_ConfigLoads`
- [ ] AC2 -> commit `<hash>` / test `TestBootstrapLane_PromptRenders`
- [ ] AC3 -> commit `<hash>` / test `TestBootstrapLane_AgentsSection`
- [ ] AC4 -> commit `<hash>` / test `TestBootstrapLane_Idempotent`
- [ ] AC5 -> commit `<hash>` / test `TestBootstrapLane_KeepsExisting`
- [ ] AC6 -> commit `<hash>` / test `TestBootstrapLane_Force`
- [ ] AC7 -> commit `<hash>` / tests `TestInit_LaneStep`, `TestInitLane_DryRun`
- [x] AC8 -> kubelab rollout follow-up filed as #3154

## Test status

- Test suite: `<command> -> <output / coverage %>`
- Manual smoke test: what was exercised, what was observed
- No regressions in existing test suite: yes / no (if no, document)

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

-
-

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-129-initrepo-lane-contract/` -> `specs/archive/CLI-129-initrepo-lane-contract/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
