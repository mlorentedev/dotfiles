---
id: "lesson-362-an-uncomputed-merge-state-is-not-a-pending-check"
type: lesson
status: active
title: "An uncomputed merge state is not a pending check"
created: "2026-10-07"
---

# An uncomputed merge state is not a pending check

## Context
`dotf pr land --wait` waits while a PR's facts can still change: a check still pending, or a merge
state GitHub has not finished computing (`mergeStateStatus: UNKNOWN`, `mergeable: null` over REST).
One budget covered both, six rounds of a 30-second pause, sized for checks that register a few
seconds after a push.

## The Trap
The two waits have different tails. A check finishes or fails within its job's timeout. A merge
state has no deadline. On 2026-10-07, #2111 had every one of its 22 checks green and
`mergeable` still null 27 minutes after the push, with every GitHub system reported operational.
During an incident the same day, #2105 stayed null for more than ten minutes and no `pull_request`
workflow ran on that head at all. Each time the lander gave up after about three minutes and
reported `merge state is UNKNOWN`. That refusal was safe but useless: it read like a blocked PR,
and it gave no hint that waiting, or a new head, would end it.

## The Solution
Give the uncomputed state its own budget (`--unknown-wait`, 20 minutes by default), counted only
while every check is green and the state is UNKNOWN. BLOCKED keeps the short budget, because a PR
can be blocked for good with every check green. When the budget runs out, the refusal says how long
it waited and names the remedy that worked both times: a new head. Re-create the base merge with
the same tree and `--force-with-lease`. A merge commit does not count toward PR-Agent's push gate.

A wait loop is sized by the slowest thing it waits for. When it waits for two things, give each its
own bound, and make the refusal say which bound ran out (#2118).
