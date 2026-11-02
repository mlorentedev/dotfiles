---
id: "lesson-532-a-skip-boolean-cannot-carry-a-declaration"
type: lesson
status: active
title: "A skip boolean cannot carry a declaration"
created: "2026-11-01"
---

# A skip boolean cannot carry a declaration

## Context

`dotf pr declare-unreviewed` ports two kubelab steps. One declares a PR when `steps.reviewable.outputs.any` is
`false`, the other clears the label when it is `true`. The obvious port feeds them `reviewable=true|false` from
`dotf pr reviewable`.

## Problem

The port's boolean answers a different question. kubelab's `any` says what the diff holds, and an `open` step
before it had already skipped drafts and closed PRs. `reviewable` says whether to run PR-Agent, so its `false` also
means a draft, a closed PR or a forfeited branch, and its `true` also means a read that failed, because a read
that fails keeps the review. Wired directly, a draft would get a rationale saying its diff is empty, and an API
outage would clear a label that was right. Both defaults are correct for the decision they were made for: the safe
direction for running a review is not the safe direction for writing a claim on the PR.

The same shape hid in the section check. kubelab's `grep -qxF` looks like the gate's match and is not: the gate
strips `\r` and refuses an empty section, so a CRLF body would gain a second copy and an empty heading would be
reported as declared while the gate stayed red.

## Solution

`dotf pr reviewable` also prints `diff=empty|reviewable|unread`, and `unread` covers everything that did not read
the answer, a filter source lost along the way included. The verb acts on `empty` and `reviewable` only, and takes
the value as an input, so a run lists the files once. `withSection` reads the body as the gate does.

## Lesson

When a port moves a value from one step to another, check what the value was filtered by upstream, not only its
name. When a fail-safe default feeds a write, ask whether it is still safe there: "unknown, so proceed" is a good
rule for doing work and a bad one for making a claim. Where two tools must agree on a text, read the one that
decides, not the one being ported.
