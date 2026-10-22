---
id: "lesson-510-a-skipped-job-still-cancels-through-its-concurrency-group"
type: lesson
status: active
title: "A skipped job still cancels through its concurrency group"
created: "2026-10-21"
---

# A skipped job still cancels through its concurrency group

## Context

`pr-agent` runs on every PR comment and decides in the job's `if:` whether the
comment is a slash command it should act on. The workflow-level concurrency
group, with `cancel-in-progress: true`, was keyed on the PR number and the event
name, so every comment on a PR shared one group.

## Problem

GitHub evaluates workflow-level `concurrency` before any job condition. A
comment the job is going to skip still enters the group, and it cancels the run
already there. Measured on #2855 (#2867): the owner's `/review` (39619480456)
was 70 seconds into its review when CodeRabbit's review comment started run
39621772110. That run skipped at the job `if:`, and it cancelled the review on
its way past. The workflow's own comment claimed "a comment no longer cancels a
review", which was true only for a review started by a push.

## Solution

The group key is computed from the same predicate the job gate uses. A comment
the job will run, a slash command from a trusted association, shares the group,
so a newer `/review` still supersedes an older one. Any other comment gets a
group of its own, keyed on `github.event.comment.id`, and cancels nothing. A
bats guard holds the group's association list equal to the job gate's.

Tightening the key also emptied an older guard: it grepped the `group:` line
for `github.event_name`, and the new condition mentions that name, so the guard
kept passing with the discriminator deleted. It now asserts the name is an
argument of the key, and the mutation that drops it is observed red.

## Takeaway

A job `if:` is not a filter on concurrency. Whatever a skipped run must not do
to its neighbours has to be expressed in the group key itself. And a guard that
greps for a token is only as strong as that token is unique: when a change adds
a second mention, re-run the guard's mutation before trusting it.
