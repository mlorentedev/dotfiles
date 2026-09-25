---
tags: [spec, tasks, templates]
created: "2026-09-25"
---

# Tasks - SKILL-001-portfolio-consolidation

> One PR here and one vault commit. The vault half is the SSOT; the records in this PR are generated from it by `compile-harness.sh --refresh`.
>
> **Inline markers**: `[AC<n>]` — this task helps satisfy acceptance criterion `<n>` from `proposal.md`.

## Setup

- [x] Ticket `mlorentedev/dotfiles#1692` filed and on the board
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] Peer dotfiles sessions told before the vault changes land

## Vault

- [x] [AC1] Move the eight skills to `90_archive/skills/`, set `status: archived`, add a banner naming each successor
- [x] [AC4] Fold the evidence rule, the Definition of Done closing pass and the code-level checklist into `adversarial-review`
- [x] [AC4] Fold plan writing and plan execution into `spec`; fold the enrich step into `new-ticket`
- [x] [AC3] Slim the reviewer, planner and builder rosters, keeping every severity; rewrite their "Forced skills" bodies and `ROSTER.md`
- [x] [AC2] Repoint every live reference: `requires:`, the Definition of Done sentence in `pattern-change-lifecycle`, patterns, sibling skills, `skills/README.md`, `CURRENT-STATE.md`

## Repo

- [x] [AC2] Failing test first: `TestEverySkillTheRouterNamesHasARecord` fails naming the retired skills that the triggers and `DefaultSkillDependencies` still named
- [x] [AC2] Remap the three trigger rules (both `triggers.json` copies) and prune `DefaultSkillDependencies`; the test passes
- [x] [AC2] `--refresh` from the vault: eight records dropped, folded skills and personas regenerated, Definition of Done regions regenerated
- [x] [AC3] `TestRoleJoinDrift` went red on the slimmed builder roster (12 of 18 rules resolving): the four domain rules also name a builder-roster skill, back to 16 of 18; `TestResolveRoles` expects reviewer and shipper for `pr-review-triage`
- [x] [AC5] Point the bats tests that read retired records at surviving ones (`skills-pipeline`, `harness-suggest`, `verify-setup`)
- [x] Usage examples, help text and comments that named retired skills (`README.md`, shell rc comments, two runbooks, `resolve-skills`, `persona.go`, the agent schema)
- [x] [AC5] Full verification: build, vet, `go test ./...`, lint, `--check`, full bats, doc checks

## After merge

- [x] [AC6] Announce, then `compile-harness.sh --deploy` from main; confirm no retired skill this pipeline rendered remains in any deploy target, and none is named in the Copilot catalog
- [x] [AC6] `scripts/check-retired-skills.sh` reads every deploy and instruction target from `harness/manifest.json`, with `tests/check-retired-skills.bats` seeding each one (independent review, round 2)

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by a test or a named check
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [ ] Independent review before archiving (a change that closes a spec)

## Machine-readable features

`features.json` (alongside this file) maps each acceptance criterion to one feature with a single shell command whose exit code is the verdict. Only the harness may set `"state": "passing"`.
