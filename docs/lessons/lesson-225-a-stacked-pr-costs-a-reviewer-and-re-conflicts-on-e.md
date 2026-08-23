# Lesson 225 — A stacked PR costs a reviewer, and re-conflicts on every squash

**Date:** 2026-08-23
**Area:** git / CI / review process
**Severity:** medium — two of three PRs in a chain were read by one reviewer agent instead of two, and nothing said so

## What happened

#1034 was split into three PRs, each stacked on the previous: #1203 (base `main`), #1206 (base #1203's branch), #1207 (base #1206's branch). The chunking itself was right — 139 conversions do not belong in one diff. The *stacking* was the part that cost, in two ways that were both invisible until they bit.

**1. Every squash-merge re-conflicts the rest of the chain.** This repo squash-merges. A squash replaces the parent PR's commits with one new commit, so the child's commits stop being ancestors of `main`. GitHub then auto-retargets the child to `main`, and the parent's entire diff reappears in it as new work:

| Event | Consequence |
|---|---|
| #1203 merged as `c660613` | #1206 retargeted to `main`, `mergeable: false, state: dirty` |
| #1206 merged as `1e21d26` | #1207 retargeted to `main`, `mergeable: false, state: dirty` |

Both were fixed the same way — `git rebase --onto origin/main <old-parent-tip> <branch>`, which replays only the branch's own commits — and both were reported by the human as *"hay conflictos"* before anyone had thought to look. The conflict is not a merge problem to be solved once; it is a scheduled event, one per merge in the chain.

**2. CodeRabbit does not review a stacked PR at all.** Not a quota window that reopens later — configuration:

> Review skipped — auto reviews are disabled on base/target branches other than the default branch.

So #1206 and #1207 were read by **one** reviewer agent (PR-Agent), while #1203 was read by two. Every check was green on all three, and nothing on the PR page distinguished "one reviewer approved" from "two did". This is a fourth distinct mechanism by which a green check does not mean reviewed, after the three in [Lesson 213](lesson-213-a-reviewer-that-reports-success-while-publishing.md) and the release-branch auto-pause in #1196.

## Why the mistake is easy

Stacking is the obvious answer to "these PRs depend on each other", and here they *looked* dependent: chunks 2 and 3 both used the library that chunk 1 introduced.

They were not. Measured after the fact: chunk 2 and chunk 3 touched **disjoint sets of test files**. Their only shared surface was one line-range in the guard's quarantine list. Only chunk 1 had to land first, because it added `tests/lib/refute.bash`. Chunks 2 and 3 could have been cut from `main` *after* chunk 1 merged, as two independent PRs — no retargeting, no rebase per merge, two reviewers each.

The tell was available before the first stack was created: *do these branches modify the same files, or merely rely on the same already-merged code?*

## The rule

**Stack only when the diffs overlap; otherwise serialise off `main`.** Depending on a merged artifact is not a reason to stack — it is a reason to wait for the merge.

When stacking is genuinely required:

- **Budget one rebase per merge in the chain**, and do it immediately after the parent lands, not when someone reports a conflict. `git rebase --onto origin/main <old-parent-tip> <branch>` is the form; a plain `git rebase main` replays the parent's commits too and conflicts against itself.
- **Say in the PR body that the reviewer coverage is halved.** A stacked PR's `## Review triage` should record *"read by one reviewer agent only, by configuration"* explicitly. Disclosure over silence, the same posture as merging unreviewed.
- **Prefer disjoint files over a shared ratchet.** The one line the chunks contended over was the quarantine list. A per-file marker would have made even that independent — worth considering the next time a ratchet spans a multi-PR conversion.

## Evidence

- #1203 (`c660613`), #1206 (`1e21d26`), #1207 (`f6b4d36`) — the chain
- `mergeable_state: dirty` observed twice, once after each parent merge
- CodeRabbit's skip notice on #1206 and #1207; its full review on #1203
- #1196 — the release-branch variant of the same "green but unreviewed" class
