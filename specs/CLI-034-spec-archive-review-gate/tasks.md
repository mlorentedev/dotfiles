---
tags: [spec, tasks, templates]
created: "2026-08-09"
---

# Tasks - CLI-034-spec-archive-review-gate

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft`
> state; freeze once you start `implementing`.

## Setup

- [ ] Branch created from main: `feat/spec-archive-review-gate` (external worktree
      `../dotfiles-wt-review-gate`, Standing Order #9)
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> The parser and the gate are independent behaviors from the staleness check, so their first test
> tasks carry `[P]`. Within each behavior the TDD chain is sequential.

### Review artifact parsing

- [ ] [P] [AC1] Write failing test: `FindReview` on a spec dir with no `review.md` returns a
      typed "absent" result, not an error
- [ ] [AC1] Implement `review.go`: frontmatter parse into a `Review{Verdict, ReviewedSHA, Reviewer,
      Date, Spec}` struct, reusing the existing frontmatter helper used by `setStatus`
- [ ] [AC2][AC3] Write failing test: verdict values `PASS`, `PASS-WITH-GAPS`, `FAIL` and an
      unrecognized value each map to the right decision (unknown verdict = refuse)
- [ ] [AC2][AC3] Implement verdict classification
- [ ] Refactor: keep the parser free of any filesystem/git knowledge (pure input → decision)

### The archive pre-flight

- [ ] [AC1] Write failing test `TestArchiveBlocksOnMissingReview` — mirrors
      `TestArchiveBlocksOnDrafts`
- [ ] [AC2] Write failing test `TestArchiveBlocksOnFailVerdict`
- [ ] [AC3] Write failing test `TestArchiveProceedsOnPassWithGaps`
- [ ] [AC1][AC2][AC3] Wire the pre-flight into `Archive()` after the tag check, before the move;
      error text names the missing artifact and both declared escapes
- [ ] [AC5] Write failing test `TestArchiveForceWithoutReview` — mirrors
      `TestArchiveForceWithDrafts`
- [ ] [AC5] Add `ForceWithoutReview` to `ArchiveOptions` + the `--force-without-review` flag in
      `cli/internal/cmd/spec.go`

### Declared waiver

- [ ] [AC4] Write failing test: `review: waived` + non-empty `review_waived_reason:` archives
- [ ] [AC4] Write failing test: `review: waived` with empty/absent reason still refuses
- [ ] [AC4] Implement the waiver read from `proposal.md` frontmatter

### Staleness floor

- [ ] [P] [AC6] Write failing test `TestArchiveBlocksOnStaleReview` — `proposal.md` committed
      after `reviewed_sha`
- [ ] [AC6] Implement the staleness check: `git log <reviewed_sha>..HEAD -- <contract files>`,
      scoped to `proposal.md`, `tasks.md`, `features.json`
- [ ] [AC7] Write failing test: the commit introducing `review.md` does NOT trip staleness
- [ ] [AC7] Write failing test: a later `verification.md` change does NOT trip staleness
- [ ] [AC6] Write failing test: an unresolvable `reviewed_sha` (post-rebase) is treated as stale
- [ ] Refactor: staleness behind a small interface so tests inject a fake git rather than shelling
      out in every case

### Producer contract (vault SSOT — NOT the harness render)

- [ ] [AC8] Update `$VAULT_PATH/00_meta/skills/adversarial-review/SKILL.md`: add the `review.md`
      destination + frontmatter schema to "Output format", and make "Completion" instruct writing
      the file
- [ ] [AC8] Update `$VAULT_PATH/00_meta/skills/spec/SKILL.md` `archive` subcommand: document the
      second pre-flight and both escapes
- [ ] [AC8] Run `compile-harness.sh --refresh` so `harness/skills/` renders match, and verify the
      render diff is exactly the vault change (CLI-005 lesson)

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous
      verification command
- [ ] Type checks pass (`go vet ./...`)
- [ ] Lint passes
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `review.md` written for THIS spec by an independent session — dogfood the gate on itself
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

See sibling `features.json`. Pass-state gating applies: the agent cannot write `"state":
"passing"` — only the harness may, after capturing exit code 0 with evidence.
