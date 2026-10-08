---
id: "lesson-364-a-regenerate-from-ssot-step-must-know-which-way-its-clone-is-off"
type: lesson
status: active
title: "A regenerate-from-SSOT step must know which way its clone is off"
created: "2026-10-08"
---

# A regenerate-from-SSOT step must know which way its clone is off

## Context
`setup-linux.sh` ran `compile-harness.sh --refresh`, which rewrites the committed harness records
from the local vault clone. Any change it left was announced as "Harness records changed by
--refresh from the vault -- commit them", with the `git commit` line to run (OPS-003, #295).

## The Trap
"The record differs from the vault" was read as "the vault is ahead", and that holds only for a
clone that is current. On the macOS bring-up (#2013) the vault clone was one commit behind
`origin/master`. The refresh removed a sentence #2155 had shipped from
`harness/skills/spec/SKILL.md`, and setup asked for that revert to be committed. The drift report
made it worse: the louder the announcement, the more a stale clone looks like news. Nothing failed,
and the instruction was wrong.

## The Solution
`dotf harness refresh` checks the vault against its upstream before it reads it, with the
fast-forward logic `dotf update` already used (`update.Sync`). A clone that is behind is
fast-forwarded first. One that is ahead is read, because its extra commits are the newer state.
One that is diverged, dirty, offline or has no upstream is not read: the committed records stand,
and the warning names the reason (#2162).

A generator that reads a local copy of its source of truth inherits that copy's age. Before
reading, establish which side is newer. A diff on its own does not say.
