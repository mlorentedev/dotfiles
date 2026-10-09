---
id: "lesson-368-two-checks-of-one-property-must-share-one-predicate"
type: lesson
status: active
title: "Two checks of one property must share one predicate"
created: "2026-10-08"
---

# Two checks of one property must share one predicate

## Context
The age identity `~/.config/age/key.txt` decrypts every age secret and the DR escrow. Two
commands judge its health: `dotf secrets verify` (the secrets layer) and `dotf doctor` (the
post-setup health check). On the Mac bring-up (2026-10-08) the key was restored at mode 0644.

## The Trap
`verify` FAILED the key: `has mode 0644, expected 0600`. `doctor` reported it `file-authority on
disk` and passed. Both looked at the same file and the same declaration. Doctor had written its
own, weaker question ("does the path exist?") instead of asking the secrets layer's ("is this
root what was declared?"). Its verdict was green, and doctor is the check people actually run.

The weaker copy was not a bug when it was written; it went stale when `verify` gained the mode
rule and doctor did not. Two predicates for one property drift apart, and the reader trusts
whichever one they ran.

## The Solution
When a second command reports on a property another layer already checks, it calls that layer's
predicate. It does not re-derive it. Doctor now asks `secrets.CheckFileAuthorityMode`, which
shares `checkKeyMode` with `verify`, and `--fix` calls `secrets.RepairFileAuthorityMode` (#2203).
The two commands can no longer disagree about the mode, because there is only one answer.

When reviewing a health check, look for a reimplemented condition: a `pathExists` or `Stat`
standing in for a richer check somewhere else is the shape this takes.
