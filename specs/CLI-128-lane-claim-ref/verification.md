---
tags: [spec, verification, templates]
created: "2026-10-16"
---

# Verification - CLI-128-lane-claim-ref

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

### PR 1 (the claim primitive)

- [x] AC1 -> `TestClaimAcquire` (same and other holder refused; found while writing it: an identical claim commit is "up to date", not a refusal, so every claim carries a nonce, lesson 459)
- [x] AC2 -> `TestClaimedErrorNamesTheHolder`
- [x] AC3 -> `TestClaimConcurrent` (8 claimants, 1 winner, 20 runs green)
- [x] AC6 -> `TestClaimIsInvisibleToADefaultClone`
- [ ] AC4 (`lane next` claims first), AC5 (`lane release`; the primitive is `TestClaimRelease`), AC7 (forge probe) -> PR 2 and 3

The forge probe is `TestClaimRefFormOnARemote`, opt-in through `DOTF_CLAIM_PROBE_REMOTE`; it passes against a local bare remote. A push of a probe ref to the real `origin` was refused by the harness, so GitHub and Gitea are still unprobed and AC7 stays open until the owner runs it.

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
- [ ] Folder moved: `specs/CLI-128-lane-claim-ref/` -> `specs/archive/CLI-128-lane-claim-ref/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
