---
id: lesson-307
type: lesson
status: active
created: "2026-09-25"
owner: manu
tags: [lesson, ci, guard, pr-agent, review]
---

# 307 — A new silent path must be taught to the guard that demands output

## What happened

TOOL-023 set out to stop pr-agent re-reviewing every push. The obvious change was to set PR-Agent's own incremental thresholds and let it skip a push below them.

The workflow already has a guard (#1107) that fails the job when no review comment was published. It exists because PR-Agent turns a failed inference into a clean exit.

A threshold skip is also a clean exit with no comment. So every push held back would have turned the job red, reading as "NaN concurrency exhaustion". Every correct skip would have looked like the failure the guard was built to catch.

The fix moved the decision in front of the tool. A gate step decides from the PR's own comments and commits. When it says no, neither PR-Agent nor the guard runs. The gate mirrors PR-Agent's rule for "the previous review", so it never starts a run that PR-Agent would decline.

## The rule

When you add a path on which a producer legitimately produces nothing, find every consumer that reads "nothing" as failure, and teach it the new path in the same change. The safest place for the decision is before the producer, where one visible step makes it, not inside the producer where only its log knows. A guard that cannot tell a correct silence from a broken one has to be told which one it is looking at.

Refs: TOOL-023 (#1756), ADR-040, #1107.
