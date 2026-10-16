---
id: "lesson-449-ancestry-reads-a-fresh-branch-as-merged-and-a-phase-table-hides-how-rules-compete"
type: lesson
status: active
title: "Ancestry reads a fresh branch as merged, and a one-row-per-phase table hides how the rules compete"
created: "2026-10-16"
owner: manu
tags: [lane, worktree, status, testing]
---

# Ancestry reads a fresh branch as merged, and a one-row-per-phase table hides how the rules compete

**Context:** `dotf lane status` (CLI-127, #2550) derives a phase for every worktree. Landed means the PR merged, and the existing `worktree.RealGitRunner.IsPRMerged` answers that by asking first whether the branch is an ancestor of `origin/main`.

**Problem:** Two defects, neither visible to the fixture tests that passed.
1. A branch that has no commit of its own yet has its tip already in the base, so ancestry says merged. A lane started a second ago would read Landed. `Classify` hides this behind a 15-minute age gate that a status table does not have.
2. The head of the landing order was picked among every open PR. The fixture had one row per phase and every phase was right. The first run against this machine showed the open PR of an unadopted worktree taking the head, and since that row then derived to Unadopted, no lane was Landing at all.

**Solution:** `lane.ForgeGit` answers `IsPRMerged` from the forge's REST endpoint alone, so a branch merged without a pull request reads as unmerged: the safe side, because `worktree sweep` re-checks every gate itself. `markHead` picks the lowest-numbered PR among the lanes that already derived to Review. `TestStatus/LandingHeadIsPickedAmongReviewLanes` pins the second; `TestForgeGit_MergedComesFromTheForgeAlone` pins the first.

**Why:** Ancestry means "holds no work the base lacks", which a branch with no work also satisfies; it is not the same as "merged". And a table with one row per phase proves each rule alone, not the order in which they compete for the same row. Derive in two passes where a phase depends on the other rows, and run the read-only verb against the real machine before calling it done: a live fleet has the mixed states a fixture has to be told about.
