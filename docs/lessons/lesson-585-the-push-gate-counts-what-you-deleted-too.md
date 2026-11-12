---
id: "lesson-585-the-push-gate-counts-what-you-deleted-too"
type: lesson
status: active
title: "The push gate counts what you deleted too"
created: "2026-11-11"
---

# The push gate counts what you deleted too

## Context

#3263 asked sweep to say why it did not park a lane. The change was sized before writing it as
"under 50 production lines, no spec needed", and the diff was measured the same way afterwards:
33 added, non-blank, non-comment lines in `sweep.go` and `cmd/worktree.go`.

## Problem

The pre-push SDD gate refused the push with "Production diff: 66 LOC (>= threshold 50)". The gate
counts the diffstat of production files, insertions plus deletions, and the refactor that split
`laneLanded` out of `parkSingleNamedLane` deleted as many lines as it added. By then the fix was
written, tested and committed, and the remedy the gate names, `dotf spec init`, refused too: ten
specs were active, the ADR-043 limit. Finished work was parked (#3263) for a number that could have
been read before the first line was written.

## Solution

Size a change the way the gate will: `git diff --stat <base>...HEAD -- <production files>` (the
merge base, so a base that moved on does not count) and read the total, or run the gate's own
check, `dotf gate spec --explain` with its `--base` and `--head`, on the committed changes. When the total will cross the
threshold, check the active-spec count on the base branch first; at the limit, the spec the item
needs opens only with `dotf spec init --over-wip-limit "<reason>"`, which records the reason in
its frontmatter, so the item is not ready to start until that reason exists. The autopilot
skill's third eligibility veto now says so.

## Related

- #3263, the parked change; #3310, which pinned the behaviour it completes
- ADR-043 (the WIP limit), `pattern-spec-driven-development` (the threshold)
