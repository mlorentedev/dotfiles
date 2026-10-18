# Run night-cross-os-20261018T1834Z

**Verdict:** run night-cross-os-20261018T1834Z ended `queue-empty` at about 21:30Z, after about 3h. 6 items: 4 done, 2 parked plus one half-parked (#2678's CI half). 4 PRs merged, 0 unmerged. Nothing irreversible is waiting on you: no release, tag or publish was touched, and nothing under `.github/**` changed.

## Waiting on you

1. **PLAT-003 archive.** Every acceptance criterion is met (AC8 landed in #2789). Launch `dotf spec review PLAT-003-cross-os-verification` from a worktree; the archive needs an independent reviewer. `features.json` still marks f7 and f8 `pending`; it was not touched.
2. **#2752 parked** (issuecomment-6253778551, board Blocked). Your pick was A (land renumbers a colliding lesson). A needs a spec (well over 50 LOC; WIP 13 of 10) and widens land's authority, so it is yours to land.
   - (A) Spec it once a slot frees; you land it.
   - (B) `dotf pr land` refuses when the PR adds a lesson whose number is taken on the base, and names the next free number. About 30-40 LOC; narrows authority, so it can land under the grant. Recommended.
3. **#2678 CI half parked** (issuecomment-6254098256, board Blocked). The pre-commit half landed in #2793.
   - (A) Edit `ci.yml`'s `lint-powershell` job to run `dotf lint powershell` and delete `TestLintMatchesCIsLintPowershellJob`. One implementation. Recommended; an agent can prepare it for you to land.
   - (B) Keep the inline job and the parity test.
4. **#2250 parked** (issuecomment-6253878904, board Blocked). It needs a new merge path in `prland`: stack detection, `merge-async` pinned to the head, and a poll. That is over 50 LOC with no spec slot, and it replaces the call that merges.
   - (A) Spec it; you land it.
   - (B) Wait for #2249 (gitea-mq), which supersedes the merge half. What remains is skipping the retarget for a stacked PR, about 20 LOC. Recommended if #2249 is scheduled.

## Done

| Item | PR | Merge | Notes |
|---|---|---|---|
| PLAT-003 AC8 (#2344) | #2789 | `2475d5ae` | merged by mlorentedev at 19:18Z, before this run's land; noted on #2344 |
| #2678 pre-commit half | #2793 | `a73599e9` | hook runs `dotf lint powershell`, skips without pwsh; real-pre-commit test; lesson 492 |
| #1719 PR 1 | #2796 | `f40a76a6` | `spec review` and `spec init` closures from 22 to 9 or less; cap 22 to 21 |
| #1719 PR 2 | #2800 | `b4b1b2ad` | harness suggest, secrets probe and migrate, `laneNextDryRun` to 13 or less; cap 15; closes #1719 |

#2793, #2796 and #2800 landed through `dotf pr triage-queue --pr N && dotf pr land N`. All four carry a `## Review triage` comment. pr-agent reviewed every PR. CodeRabbit posted only rate-limit notices, so it reviewed none of them.

## Findings worth knowing

- **#2793's first CI round** failed two repo-wide bats guards: `bats-pwsh-gate` (use `require_pwsh`) and `guard-bats-negation` (no bare `!`). Neither suite names the file it checks, so the reference grep from lesson 347 cannot find them. Lesson 492: run `bats tests/guard-*.bats tests/bats-*.bats` whenever a bats file changes.
- **#2796 split a doc comment.** Its helpers were inserted between `chooseForLaunch`'s comment and the function. pr-agent caught the same mistake in #2800 (`printDistribution`), and #2800 fixes both.
- **pr-agent's broken-pwsh finding on #2793 was declined.** A pwsh that cannot run is a broken install, and `dotf lint powershell` fails loudly on it with pwsh's own stderr. The reason is recorded in the config comment.

## Filed

- #2788: `dotf vm linux --ref` needs a branch, tag or full SHA, because the guest fetches the ref by name.

## Notes

- **GraphQL limit:** the shared limit ran out across sessions around 19:05Z, and two waiters ended early on it. Waiting moved to REST check-runs; no decision was made on a failed read.
- **Not mine:** the repo-wide `dotf pr triage-queue` at the end lists peer PRs #2809, #2807 and #2799, none of them this run's.
- **Denied, not retried:** the PATH-manipulation probe for #2793's no-pwsh live case. The stubbed bats test covers that case, and the PR discloses it.
- **Wake-lock:** this run stopped only its own `caffeinate`; the orchestrator's shared one was left alone.
- **Worktrees:** the four PR worktrees are removed. This run branch and the launch worktree `feat-cross-os-verification` remain; remove them once you have read this. The run branch is never merged.
