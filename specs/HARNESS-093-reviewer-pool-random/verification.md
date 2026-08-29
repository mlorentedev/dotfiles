---
tags: [spec, verification]
created: "2026-08-29"
---

# Verification - HARNESS-093-reviewer-pool-random

## Evidence

Run on the Windows work box, 2026-08-29, worktree `dotfiles-wt-pool`, branch
`feat/reviewer-pool-random`, `dotf` built from the branch.

- [x] **AC1** → `TestDrawReviewerReachesEveryMemberAndResolveHonoursAnExplicitChoice` (draw index 0 and 1
  select members 0 and 1; an out-of-range draw errors; an empty name is refused; an explicit id
  resolves), `TestResolveReviewerRefusesAModelOutsideThePool`, `TestResolveReviewerRefusesWhenThereIsNoPool`
  (now through `DrawReviewer`).
- [x] **AC2** → `TestSpecReviewDrawsAPoolMemberAndSaysSo` (draw pinned to the last member prints
  `Reviewer:   nan/glm5.3-flash (pi, random draw)`; `--reviewer nan/deepseek-v4-flash` prints
  `(pi, requested)`); the eight pre-existing `TestSpecReview*` tests pass with the draw pinned to index 0
  by an `init` in the test file, so they keep meaning what they meant.
- [x] **AC3** → `tests/reviewer-pool.bats` 4/4: unique ids; the four NaN models present (five members
  with `agy/gemini-3.1-pro-high`); every `pi` member has `reasoning: true` in `ai/pi/models.json`;
  `spec.go` uses `rand.IntN` and documents the draw; the committed skill record says "drawn at random".
  The vault source `00_meta/skills/adversarial-review/SKILL.md` carries the same line (committed to the
  vault's master; the record's `generated_sha` is refreshed by the next `compile-harness.sh --refresh`
  on a Linux box — ENGINE-001 keeps `--refresh` Linux-only).
- [x] **AC4** → box: six `dotf spec review HARNESS-093-reviewer-pool-random --dry-run` launches →
  `agy/gemini-3.1-pro-high` ×1, `nan/glm5.3-flash` ×2, `nan/qwen3.8-flash` ×3 (all "random draw");
  `--reviewer nan/glm5.3-flash` → `(pi, requested)` and the command carries `'--model' 'glm5.3-flash'`.

## Test status

```text
go build ./... && go vet ./... && go test ./internal/spec/ ./internal/cmd/ -run 'Reviewer|SpecReview'   -> ok
golangci-lint run ./internal/spec/ ./internal/cmd/   -> 0 issues
bats tests/reviewer-pool.bats   -> 4/4
```

- No regressions in the existing suite: yes.

## Decisions made during implementation

- **Every member is drawn, agy included.** The pool is the allow-list; a member that should not be
  drawn should not be in it. agy's login-based auth failing on a box reaches the launcher's
  existing "launch died — try another member" path, which names an alternative.
- **`role` stays as provenance.** `primary`/`fallback`/`member` no longer steer selection; the
  launch line prints how the member was chosen (`random draw` / `requested`) instead of the role.
- **What the new entries do not claim.** Both `why` fields say the models are reasoning-class by
  profile and admitted for bucket spread, and that a calibration against a review of known outcome
  has not been done — so nobody reads them as measured equivalence to deepseek.
- **Randomness is a seam.** `reviewerDraw = rand.IntN` in production; tests pin it.

## Promotion candidates

- [ ] Lesson: no.
- [ ] ADR-worthy decision: no — HARNESS-071's pool, a selection policy change recorded in the file.
- [ ] Pattern: no.

## Archive checklist

- [ ] `dotf spec review HARNESS-093-reviewer-pool-random` PASS (the first review drawn under the new policy)
- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/HARNESS-093-reviewer-pool-random/`
- [ ] Bitácora #1370 closed with the PR link
