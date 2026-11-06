---
id: "lesson-549-an-event-payload-is-a-snapshot-and-a-403-is-not-always-a-rate-limit"
type: lesson
status: active
title: "An event payload is a snapshot, and a 403 is not always a rate limit"
created: "2026-11-04"
---

# An event payload is a snapshot, and a 403 is not always a rate limit

## Context

`bitacora-status.yml` moves an assigned issue to In Progress on the bitácora board. It is copied to every
repository, and the first reviews of the copies found three defects in it (#3160 items 3, 4 and 8).

## Problem

The job gated on `github.event.issue.state`, which is the state at assignment time. A run that waits in the
queue while the issue closes still put it on the board. The guard read the Status by field name and compared
the option name, so renaming either made a parked issue read as unset, and the run moved it. No call had a
retry, so one transient error left the issue where it was.

## Solution

Re-read the live issue before the write, and compare by field and option id, the way the write already worked.
When adding the retry, the obvious rule is wrong in two places. A 403 is a rate limit only when GitHub says so,
with `retry-after` or a rate-limit message. Otherwise it is a missing permission, and retrying it costs two
minutes before the same failure. GraphQL reports `RATE_LIMITED` with HTTP 200, so an exception carrying
`errors` and no status is not a network failure: its type decides. The tests run the script extracted from the
workflow against a stubbed client, and they fail against the previous copy.
