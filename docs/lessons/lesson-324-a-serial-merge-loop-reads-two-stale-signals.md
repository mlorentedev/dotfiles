---
id: "lesson-324-a-serial-merge-loop-reads-two-stale-signals"
type: lesson
status: active
title: "A serial merge loop reads two stale signals"
created: "2026-09-30"
owner: manu
tags: [lesson, github, merge, triage]
---

# A serial merge loop reads two stale signals

## What happened

Seven PRs were landed one at a time before cutting release 0.62.0. Branch protection requires a branch to be up to date, so each merge puts every other PR behind: update-branch, wait for CI (about ten minutes with the Windows job), merge, next.

Two signals were read at the wrong moment.

1. **The merge state reads `UNKNOWN` for several seconds after main moves.** The loop read `mergeStateStatus` once, right after the previous merge. It saw `UNKNOWN`, not `BEHIND`, skipped update-branch and waited on checks that were already green. The PR stayed `BEHIND`, and the merge step found it that way.
2. **update-branch invites new review output.** CodeRabbit edited its comment on #1897 while the new head's CI ran. The merge command ran `dotf pr triage-queue` for display and then merged whatever it printed, so #1897 merged while the queue still listed it. Read afterwards, the comment was "No actionable comments" plus a pause notice, so no finding was lost. The gate had still not been applied, and the next edit could have carried one.

## Rule

- After main moves, poll `mergeStateStatus` until it is no longer `UNKNOWN` before deciding whether to update the branch.
- Gate the merge on the queue: refuse when `dotf pr triage-queue` lists the PR, or when any check on the current head is not passing. A queue printed next to a merge is not a gate. The command exits 1 both when the queue lists PRs and when it could not be computed. A list without this PR lets the merge go ahead; an error with no list blocks it.
- Re-read the head sha after update-branch and pass it to `gh pr merge --match-head-commit`. The sha from before the update names a commit that is no longer the PR.
- Update one PR at a time. Updating them all at once runs the Windows job for every PR on every merge, and saves no time.

## References

- `docs/runbooks/release-dotf.md`, "Landing several PRs before a release"
- #1897 (the post-merge triage records the slip)
