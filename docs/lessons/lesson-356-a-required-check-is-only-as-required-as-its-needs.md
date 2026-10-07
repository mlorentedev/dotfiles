---
id: "lesson-356-a-required-check-is-only-as-required-as-its-needs"
type: lesson
status: active
title: "A required check is only as required as its needs"
created: "2026-10-07"
---

# A required check is only as required as its needs

## Context
`ci.yml`'s `lint`, `lint-powershell`, `test` and `test-windows` are required checks, and each
declared `needs: [changes]`, the paths-filter job. `changes` itself was not required (#1877).
The CI-004 checker already refused a required job with a job-level `if:` (lesson 318's class),
and nothing in it looked at `needs`.

## The Trap
A job whose needs did not all succeed is skipped, unless its own `if:` is `always()`. GitHub
reports a skipped job's check as successful, and a successful required check satisfies branch
protection. So a failed `changes` (a paths-filter bug, an action outage) would have turned all
four required checks green without running anything. The protection declared four checks and
enforced none of them against the one failure that skips them all. Reading the required list
cannot show this: each listed job is real, always reports, and has no `if:`.

## The Solution
Treat the required set as closed under `needs`: every job a required job transitively needs is
required too, unless the required job runs under `if: always()` and decides from
`needs.*.result` in a step, as `cli-gate` does. `tests/lib/check-workflow-contexts.py` rule 5
enforces it, and `changes` was added to `forge/branch-protection.json`. When adding a `needs:`
to a required job, the upstream job joins the required list in the same change.
