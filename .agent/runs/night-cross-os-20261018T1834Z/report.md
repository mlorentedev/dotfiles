# Run night-cross-os-20261018T1834Z

**Verdict (in progress, 19:25Z):** 1 landed (#2789), 3 PRs open (#2793, #2796, #2800), 2 parked (#2752, #2250). Nothing irreversible is waiting on you: no release, tag or publish was touched.

## Waiting on you

1. **PLAT-003 archive.** Every acceptance criterion is met (AC8 landed in #2789, `2475d5ae`). Launch `dotf spec review PLAT-003-cross-os-verification` from a worktree. Note: `features.json` still marks f7 and f8 `pending`; it was not touched.
2. **#2752 parked** (comment 6253778551, board Blocked). Your pick was A (land renumbers a colliding lesson). A needs a spec (well over 50 LOC; WIP 13 of 10) and widens land's authority, so it is yours to land.
   - (A) spec it once a slot frees; you land it.
   - (B) `dotf pr land` refuses when the PR adds a lesson whose number is taken on the base, and names the next free number. About 30-40 LOC; narrows authority, so it can land under the grant. Recommended.
3. **#2250 parked** (comment 6253878904, board Blocked). A new merge path in `prland` (stack detection, `merge-async` pinned to the head, poll), over 50 LOC with no spec slot, and it replaces the call that merges.
   - (A) spec it; you land it.
   - (B) wait for #2249 (gitea-mq), which supersedes the merge half; what remains is skipping the retarget for a stacked PR, about 20 LOC. Recommended if #2249 is scheduled.
4. **#2678 CI half** (see #2793 below): `ci.yml`'s lint-powershell job is frozen for the run.

## Done

| Item | PR | Result |
|---|---|---|
| PLAT-003 AC8 (#2344) | #2789 | landed `2475d5ae` at 19:18Z (merged by mlorentedev); worktree removed |

## Open PRs

| Item | PR | State |
|---|---|---|
| #2678 pre-commit half | #2793 | draft; first CI round failed two repo-wide bats guards (fixed `e9bd51e4`), lesson 492 (`4f334a2f`) |
| #1719 PR 1 | #2796 | draft; `spec review`/`spec init` closures 22 to 9 or less, cap 22 to 21 |
| #1719 PR 2 | #2800 | draft; harness suggest, secrets probe/migrate, laneNextDryRun to 13 or less; cap to 15 after #2796 lands |

## Filed

- #2788: `dotf vm linux --ref` needs a branch, tag or full SHA (the guest fetches the ref by name).

## Notes

- The shared GraphQL limit ran out across sessions around 19:05Z; waiters moved to REST.
- Denied commands (not retried): the PATH-manipulation probe for #2793's no-pwsh live case (covered by the stubbed bats test, disclosed on the PR).
- Wake-lock: this run's own `caffeinate` only; the orchestrator's shared one was left alone.
