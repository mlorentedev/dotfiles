---
id: "lesson-465-a-path-limited-git-log-prunes-the-branch-that-added-a-path-absent-from-head"
type: lesson
status: active
title: "A path-limited git log prunes the branch that added a path absent from HEAD"
created: "2026-10-17"
---

# A path-limited git log prunes the branch that added a path absent from HEAD


**Context:** `dotf spec review` resolves the review base by following a spec folder back through its renames to the commit that first added it (`ResolveReviewBase`). A consuming repository scaffolded a spec on a branch, renamed it there, and merged the branch with `--no-ff`. The review refused with "cannot resolve a review base" although the history held a clean base (#2642).

**Problem:** The lookup was `git log --diff-filter=A -- <old path>` from HEAD. The old name does not exist at HEAD, so the merge commit is TREESAME to its first parent for that path, and git's default history simplification follows only that parent. The side branch where the folder was added is pruned and the log comes back empty. An empty log read as "never committed", which the caller must treat as "cannot review".

**Solution:** `--full-history` keeps every side branch. Merge commits do not match `--diff-filter=A` without `-m`, so only the real adding commits are listed. A regression test builds the exact shape: add, rename on a branch, `--no-ff` merge, implementation on main.

**Why:** `git log -- <path>` answers "what changed this path along a simplified history", not "what ever touched it". Any lookup of a path that no longer exists at the starting commit, a rename source above all, needs `--full-history`, or it silently answers nothing and the empty result reads as a finding. The first symptom was a refusal, not a wrong answer, which is the cheap way to find out.
