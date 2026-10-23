---
id: "lesson-519-a-differential-test-that-passes-first-time-is-proven-by-mutating-the-port"
type: lesson
status: active
title: "A differential test that passes the first time is proven by mutating the port"
created: "2026-10-23"
---

# A differential test that passes the first time is proven by mutating the port

## Context

TOOL-031 A3 ports `scripts/pr-agent-publish-guard.sh` to Go as `dotf pr guard` (#2893). As in A1 and A2
(lesson-516), a differential test runs the script and the port against the same inputs: here, the stub `gh` of the
bats suite on PATH for both.

## Problem

The differential test passed on its first run, over 43 cases. Three deliberate mutations of the port then also
passed:

- Retrying twice instead of three times. This was not observable by design: the test sets the backoff to 0, and
  the call count stays the same. A unit test on the backoff covers it.
- `updated_at > started` instead of `>=`. No case had a comment stamped exactly at the run's start.
- `--jq '.head.ref'` instead of `--jq '.head.sha'`. The bats stub answers by the call's path alone and ignores
  every other argument, so a port that changed a filter still got the right answer.

A green differential test only shows that the two implementations agree on what the harness can see. The second
and third gaps were in the harness, not in the port.

## Solution

The stub copy in the Go test logs each call's full arguments next to its subject, and both runs compare that log.
A case with a comment stamped at the start covers the boundary. After the fix, `>` instead of `>=` fails 2
subtests, `.head.ref` instead of `.head.sha` fails 4, and `--slurp` instead of `--paginate` fails 32. The retry
count still passes the differential test, by design, and fails the backoff unit test.

## Rule

When a differential test passes on its first run, mutate the port in two or three places it should catch: a
boundary, an argument the fake ignores, a count. Then check that the test fails on each. A fake that answers from
part of a request has to record the rest, or the comparison cannot see it.
