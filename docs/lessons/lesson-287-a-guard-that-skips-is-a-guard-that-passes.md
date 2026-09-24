---
id: lesson-287
type: lesson
status: active
created: "2026-09-23"
owner: manu
tags: [lesson, testing, go, silent-failure, sdd, vacuity]
---

# 287 — A guard that skips is a guard that passes

## What happened

SDD-041 added a repo-level test, `TestIssueStateNoActiveSpecIsProseLinked`. It
walks this checkout's `specs/` and fails when any active spec names its tracking
issue only in prose, which is exactly how GOV-004 became a zombie. It was
written test-first, so it should have been red, because GOV-004 had not been
normalised yet. It was green.

It had skipped. The test found the checkout with `spec.RepoRoot(".")`.
`RepoRoot` climbs by `filepath.Dir`, and `filepath.Dir(".")` is `"."`, so from
a relative start it never leaves the first directory. `go test` runs in the
package directory (`cli/internal/spec`), which has no `.git`, so the lookup
failed. The test had been written defensively: *not inside a checkout →
`t.Skip`*. `go test` reports a skipped test as `ok`, and the package stayed
green.

With the start made absolute (`os.Getwd()`) and the skip replaced by a
`t.Fatal`, the same test failed on the first run:
`specs/GOV-004-agents-md-diet links its issue only in prose`.

## Why it matters

A `t.Skip` in a guard converts "I could not look" into the same `ok` that
"I looked and it is clean" produces. It is the in-test form of the `gorun`
rule (#1625 §5): `go test -run X` exits 0 with "no tests to run", and a guard
that skips on a precondition exits 0 with nothing checked. Both are green
before the guard has ever run, and both stay green after the thing they guard
has broken.

The same session measured the stakes. Once the guard really ran, the audit it
backs found 14 zombie specs in dotfiles and 3 in hive, two of which (FEAT-015,
HIVE-119) the synthesis's frontmatter-only count had filed as merely
"unlinked".

## The rule

1. **A guard whose precondition fails must fail, not skip.** Skip only for
   conditions that make the guard *meaningless*, such as a Windows-only probe
   on Linux. "I cannot find what I am supposed to check" is not one of those
   conditions.
2. **A guard asserts that it checked something.** End the walk with
   `if checked == 0 { t.Fatal(...) }`, so an empty or mis-rooted walk is a
   failure rather than a vacuous pass.
3. **Resolve roots from an absolute path.** Anything that climbs with
   `filepath.Dir` must start from `os.Getwd()` or `filepath.Abs`, never `"."`.
4. **Watch the new guard fail before trusting its green**, the same rule as
   mutation testing (lesson 284): the red run is the only evidence that the
   test can see the defect.

Refs: SDD-041 (#1087), epic #1625 §5 (`gorun`), lesson 284.
