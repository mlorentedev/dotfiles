---
tags: [spec, tasks]
created: "2026-08-29"
---

# Tasks - HARNESS-093-reviewer-pool-random

## Setup

- [x] Branch: `feat/reviewer-pool-random` (worktree `dotfiles-wt-pool`) from `origin/main`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] No open questions left in `proposal.md`

## Implementation

- [x] [P] [AC1] Failing test `TestDrawReviewerReachesEveryMemberAndResolveHonoursAnExplicitChoice`
- [x] [AC1] `spec.DrawReviewer(entries, draw)`; `ResolveReviewer` refuses an empty name (the launcher draws instead)
- [x] [P] [AC2] Failing test `TestSpecReviewDrawsAPoolMemberAndSaysSo` (draw pinned to the last member → "random draw"; `--reviewer` → "requested")
- [x] [AC2] `dotf spec review`: `reviewerDraw = rand.IntN` seam, launch line prints how the member was chosen, `--reviewer` help + Long updated; existing cmd tests pinned to index 0 in an `init`
- [x] [P] [AC3] `harness/reviewer-pool.json`: `nan/glm5.3-flash`, `nan/qwen3.8-flash` (role `member`, `why` states what is and is not established); `$comment` says order is provenance, not precedence
- [x] [AC3] `tests/reviewer-pool.bats`: unique ids, ≥4 members incl. the four NaN models, every pi member `reasoning: true` in `ai/pi/models.json`, the launcher and the skill say "drawn at random"
- [x] [AC3] Skill usage line updated in the committed record and the vault source (`00_meta/skills/adversarial-review/SKILL.md`)
- [x] [AC4] Box: six `--dry-run` launches drew agy/gemini-3.1-pro-high, nan/glm5.3-flash and nan/qwen3.8-flash; `--reviewer nan/glm5.3-flash` prints `'--model' 'glm5.3-flash'`

## Verification

- [x] Go loop: build, vet, test (`spec`, `cmd`), golangci-lint (0 issues)
- [x] bats: reviewer-pool 4/4
- [x] `verification.md` records the evidence; `features.json` per AC
