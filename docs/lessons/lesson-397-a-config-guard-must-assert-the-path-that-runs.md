---
id: "lesson-397-a-config-guard-must-assert-the-path-that-runs"
type: lesson
status: active
title: "A config guard must assert the path that runs"
created: "2026-10-11"
---

# A config guard must assert the path that runs

## Context
PR-Agent was adopted for inline comments on the diff (TOOL-013 AC1). After #1042 measured zero
inline comments, a `[pr_code_suggestions]` section turned dual publishing on, a bats guard
("inline suggestions are actually enabled, not merely claimed") asserted it, and the
`[pr_reviewer]` comment kept saying inline comments were the reason the tool existed.

## The Trap
#1107 then turned `auto_improve` off to halve inference demand, so the automatic path ran `review`
alone. Dual publishing configures `improve`. The guard kept asserting a setting of a command
that no longer ran on its own, so it stayed green while the automatic review posted no inline
comment at all. `review` has its own switch, `[pr_reviewer] inline_key_issues`, and upstream
defaults it to false. The TOOL-013 archive review found it (F1) by reading the setting the
running path consults, not the one the guard named.

The same file had a smaller case of the shape: declaring `[ignore] glob` replaces upstream's list
instead of extending it, so upstream's `vendor/**` default silently stopped applying (F3).

## The Solution
- `inline_key_issues = true`: each key issue with a file in the diff and a line range is posted
  inline from the same `/review` response, so no second inference call undoes #1107.
- The guard now asserts that setting AND that every PR-Agent step runs `review` automatically, so
  turning the path off fails the guard instead of leaving it guarding nothing.
- `vendor/**` is carried in the ignore list, with a test.
- The triage skill reads `pulls/N/comments` when it checks a review ran, because a Guide whose
  findings all went inline has no findings row.

## Takeaways
- A guard on a setting is only as good as its claim that the setting is read. Assert the path
  that reads it, or a later change to the path leaves the guard green over nothing.
- When a config file overrides a list section, check upstream's default for that list: an
  override is a replacement until shown otherwise.
