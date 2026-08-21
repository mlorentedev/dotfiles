# Lesson 216 — A three-dot diff does not answer "what is main missing"

**Date:** 2026-08-21
**Context:** Auditing a leftover worktree during session orientation.

## What happened

A worktree held a branch with two commits that `git log origin/main..HEAD` reported as unpushed. To
judge whether real work was at risk, this ran:

```bash
git diff --stat origin/main...HEAD     # three dots
```

It reported **39 files, +266/−439**. That was read as "39 files of unmerged work living only on
this disk", and reported to the user as a risk.

It was wrong. The correct check:

```bash
git diff origin/main HEAD              # two dots
```

Empty. The branch's content was byte-identical to `origin/main` — its PR had squash-merged hours
earlier, from that exact branch, with those exact 39 files.

## Why

The two forms answer different questions:

| Form | Question answered |
|---|---|
| `git diff A...B` | *What did B introduce since it diverged from A?* Compares **merge-base(A,B) → B** |
| `git diff A B` | *What differs between A and B right now?* |

After a squash merge the branch's commits never appear in main's history, so `A..B` still lists
them. And the three-dot diff still shows the branch's full contribution, because it is measured
from the merge base — **whether or not that contribution has since landed in A by another route.**

Squash-merge makes both signals look like unmerged work. That is lesson 210 (`git branch --merged`
says no about every branch that landed) arriving through a second door.

## The lesson

**A non-empty result reads as a finding just as readily as an empty one reads as clean.** The
repo's prohibited-pattern table already warns that certain shell constructs fail silently by
returning empty, and warns to re-run before believing an empty sweep. This is the mirror case: 39
files of output was more convincing than zero would have been, and was equally meaningless for the
question actually being asked.

Before reporting that work is at risk, ask the question in the form that answers it: *is this
content present in main?* — `git diff <target> HEAD`, empty means yes.

## See also

- `docs/lessons/lesson-210-under-squash-merge-git-branch-merged-says-no-about.md`
- `.claude/CLAUDE.md` — the prohibited-pattern table's note on silent-empty results
