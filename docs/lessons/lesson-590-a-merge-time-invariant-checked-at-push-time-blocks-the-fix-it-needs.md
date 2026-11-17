---
id: "lesson-590-a-merge-time-invariant-checked-at-push-time-blocks-the-fix-it-needs"
type: lesson
status: active
title: "A merge-time invariant checked at push time blocks the fix it needs"
created: "2026-11-12"
---

# A merge-time invariant checked at push time blocks the fix it needs

## Context

The SDD gate's archive-on-merge rule: a PR that closes an issue must archive the issue's spec in
the same PR. The pre-push adapter (BUG-061) resolves the open PR's body so the rule applies locally
as it does in CI, which is right for the LOC credit it was added for.

## Problem

On #3355 the rule and two other correct artifacts formed a cycle. The spec archives only on a
passing adversarial review; round 1 failed on a test that is red on `test (windows-latest)` only;
the fix was committed and the push was refused, since the PR body said `Fixes #3263` and the spec
was active; round 2 then failed on "the Windows proof does not exist", which was true. The way
through was to switch the body to `Refs #3263` for one push and restore it with the archive: it
works and misdescribes the PR for a few hours (#3364).

## Solution

Check the invariant where it holds. The pre-push adapter now hands the gate `SDD_PR_DRAFT`, and
on a draft PR the archive-on-merge violation is a warning naming the archive command; CI's
`spec-gate` never sets the variable, and `dotf gate spec` deliberately does not read the event's
own `draft` field, so the merge stays refused until the spec is archived. A draft cannot merge,
so nothing is lost at push time, and the review fix reaches the CI leg that proves it. The rule:
a gate that runs at more than one moment needs to know which invariant belongs to which moment,
and a push-time copy of a merge-time rule needs an exit for the state "not mergeable yet".

## Related

- #3364, the measurement on #3355; BUG-061, why the adapter resolves the PR at push time.
- The parity suite (`tests/lib/spec-gate-parity.bash`, CI-018) holds the script and `dotf gate
  spec` to the same output, so the warning text lives in both.
