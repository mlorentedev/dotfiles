---
id: "lesson-374-a-generated-file-git-merged-is-no-longer-generated"
type: lesson
status: active
title: "A generated file git merged is no longer generated"
created: "2026-10-09"
---

# A generated file git merged is no longer generated

## Context
#2188 was green and held for the owner. main was then merged into its branch (`a4a8f205`). Both
sides had added lessons, so both sides had regenerated `docs/lessons/_index.md`.

## The Trap
The merge conflicted on the index: both sides appended rows at the end of the same table
(`git merge-tree --write-tree 5521b4de 90549a9e` reproduces the conflict). It was resolved by
keeping both sides' rows. The result was a file no generator wrote. It held every entry, but out
of order, and `dotf lessons fmt --check` in hygiene went red on a PR whose own change had not
moved. The resolution looked right: no markers were left, every lesson was listed, and only a
later check disagreed with the file.

## The Solution
Regenerate, do not hand-merge: `dotf lessons fmt` on the branch, commit, push (`6797d51a`).

CI's `fmt --check` is the guard: it compares the committed file with what the generator would
write, so it catches a hand-merged index however the conflict was resolved. No new guard was needed.

## Takeaways
- **Resolve a conflict in a generated file by regenerating it, never by keeping both sides.**
  Keeping both sides preserves every entry but not the generator's order, so the file passes a
  read and fails the `--check`. The same holds for a merge that applied cleanly: no conflict
  proves only that the hunks did not overlap.
- **A red `--check` on a PR whose diff did not change points at the base.** Look for an
  update-branch merge before debugging the change itself.
