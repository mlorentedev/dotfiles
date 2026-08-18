---
id: lesson-191-continuing-work-on-a-branch-after-its-pr-squash-me
type: lesson
status: active
created: "2026-08-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 191: Continuing work on a branch after its PR squash-merged reopens the whole original diff

**Context**: HARNESS-069 (#917, PR #927) merged via squash, producing commit `147e4ed` on `main`. Adversarial review round 1 then found a real Blocker in the merged code (`setup-windows.ps1`'s twin never got the strip-rule fix). Rather than branching fresh off the now-updated `main`, the fix was committed directly on top of the *same local branch* (`feat/harness-record-provenance`) whose earlier commits had already been merged.

**Problem**: a squash merge creates a brand-new commit on `main` with a different SHA than any commit on the source branch — the original commits are never main's ancestors. Continuing to commit on that same branch means its history still contains the pre-squash commits, so `git diff origin/main..HEAD` (what a PR would show) recomputes against the merge-base *before* the squash, showing the entire original feature diff a second time, plus the new work. This surfaced at push time as a local `sdd-spec-gate` pre-push hook failure ("Production diff: 174 LOC... No specs/<feature-id>/ folder touched") that made no sense until traced: the diff included 38 already-merged record changes the gate couldn't map to a spec folder (since the archive step had since renamed it), not just the 4 new commits.

**Solution**: `git branch -D` the reused local branch (after confirming it added nothing not already reachable — `git merge-base --is-ancestor` returns false here precisely *because* of the squash, so containment has to be checked by diffing branch-tip-vs-merge-base against the actual squash commit's own diff, not by ancestry). Create a fresh branch off `origin/main`, then `git cherry-pick` only the follow-up commits onto it (commits already fully reflected upstream cherry-pick as empty and are skipped automatically). Verify the new branch's diff against `origin/main` contains only the intended follow-up changes before pushing.

**Rule**: once a PR merges (especially via squash, the repo default), the branch is done — checked out follow-up work never continues on it, even in the same worktree, even for "just one more fix." Branch fresh off updated `main` for every new unit of work, per [[feedback_dotfiles_use_worktree]] and [[feedback_cleanup_merged_worktrees]]; those rules exist for exactly this failure mode, not only for tidiness. If work is accidentally continued on a stale branch, `git cherry-pick` the follow-up commits onto a fresh branch from `origin/main` rather than trying to rebase or reset the reused branch into shape.

**Tags**: `git`, `github`, `workflow`, `worktree`
