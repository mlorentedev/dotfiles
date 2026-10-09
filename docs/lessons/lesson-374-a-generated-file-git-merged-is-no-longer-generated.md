---
id: "lesson-374-a-generated-file-git-merged-is-no-longer-generated"
type: lesson
status: active
title: "A generated file git merged is no longer generated"
created: "2026-10-09"
---

# A generated file git merged is no longer generated

## Context
#2188 was green and held for the owner. Another actor then merged main into its branch through
GitHub's update-branch. Both sides had added lessons, so both sides had regenerated
`docs/lessons/_index.md`.

## The Trap
git merged the two versions of the index line by line, with no conflict. The result was a file
no generator wrote. It held every entry, but out of order, and `dotf lessons fmt --check` in
hygiene went red on a PR whose own diff had not changed. Nothing in the merge looked wrong: no
conflict markers, no failed step, only a later check that disagreed with the file.

## The Solution
Regenerate, do not hand-merge: `dotf lessons fmt` on the branch, commit, push (`6797d51a`).

CI's `fmt --check` is the guard: it compares the committed file with what the generator would
write, so it catches a merge-made index whatever path the merge took. No new guard was needed.

## Takeaways
- **After any merge into a branch that carries a generated file, rerun its generator.** A clean
  merge proves that git found no overlapping hunks, not that the result is what the generator
  produces.
- **A red `--check` on a PR whose diff did not change points at the base.** Look for an
  update-branch merge before debugging the change itself.
