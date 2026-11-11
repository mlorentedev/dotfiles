# run lanes-20261111t0748z

Armed 2026-11-11T07:48Z, expires 2026-11-11T19:48Z. Objective: land HARNESS-187 (#3261), then #3262, #3263, #3265, #3293 in order. Interim entries below; the disarm entry closes the file.

## Iteration 1: #3261 (HARNESS-187, lane identity in the worktree git dir)

- Branch `feat/3261-lane-identity`, worktree `feat-3261-lane-identity`. Nine commits through `5dca2592`+1; production lines 172 of the 300 cap, 13 files.
- Second advisory pass found two defects in the first implementation, both fixed before review: a refused `lane init` wrote the marker first (guard armed with no lanes), and `lane switch`/`lane park` still reached another repository's lane by path. The AC3 test asserts both.
- Full Go suite ok, pinned lint 0 issues on both OSes, shellcheck clean.
- Follow-up filed: #3318 (two repositories sharing a basename cannot both have lanes).
- Pooled review round 1 (nan/glm5.3-flash): FAIL on one REAL Major, `lane.Identity` erroring on a main checkout whose common git dir holds the `dotf lane next` prompt directory of the same name; fixed with a regression test. Round 2 (nan/mimo-v2.6-flash): PASS; two minors applied (unreadable record fails closed; both sides read one line), one ticketed, rest declined with reasons in verification.md.
- Follow-ups filed: #3320 (dotf-lane name shared by the prompt directory and the record; on the board as HARNESS-189, P3), #3324 (rule 6b message names the main checkout for any detached leaseless worktree; board fields pending, GraphQL quota exhausted when filed).
- Spec archived; PR #3325 opened as a draft and marked ready once the shared GraphQL quota allowed it; CI and reviewer triage follow.
- Peer coordination: feat-worktree-one-root session took lesson 580 on #3312; this run holds 579 and ADR-061.

## Iteration 2: #3262 (HARNESS-190, dotf lane doctor)

- Branch `feat/3262-lane-doctor`, worktree `feat-3262-lane-doctor`, stacked on `feat/3261-lane-identity` until #3325 lands; rebased onto main after that, then reviewed and archived, since the review binds to a commit.
- The committed envelope does not list `cli/internal/doctor/**`, so the check lives in `cli/internal/lane` (`AuditLanes`) behind `dotf lane doctor`, the home the issue allows; the `dotf doctor` sweep entry is #3327 with two options. Production lines 254 of the 300 cap, 11 files.
- Measured on a scratch repository before writing: after `mv` of a lane, git keeps the admin dir with a `gitdir` pointing at the old path and the record intact, lists the old path as prunable, and refuses `worktree add` at it. The audit enumerates `<repo>/.git/worktrees/*/` for that reason (lesson 581), and `gitdir` is relative under `worktree.useRelativePaths` (git 2.54 measured).
- Advisory pass before review found four gaps, all applied: a second reader of the record (extracted `readIdentityRecord`, shared with `Identity`, unreadable is a finding), the relative gitdir, an empty `WORKTREE_ROOT` resolving to the cwd, and drift tests asserting full paths that only pass on macOS by substring luck.
- Spec HARNESS-190 at `verifying`; features f1..f4 verified; pinned lint 0 issues on both OSes; full suite pending at the time of this entry.
