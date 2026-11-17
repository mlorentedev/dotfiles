---
id: "lesson-571-a-negated-closing-keyword-still-closes-the-issue"
type: lesson
status: active
title: "A negated closing keyword still closes the issue"
created: "2026-11-10"
---

# A negated closing keyword still closes the issue

## Context

#3279 implemented HARNESS-158 but had to leave its issue, #1727, open until the spec's independent review and
archive. Its body said `Part of #1727`, and then, to make that explicit, "this PR does not close #1727".

## Problem

Both readers that decide whether a PR closes an issue match the keyword and the number, not the sentence:

- GitHub listed #1727 under the PR's `closingIssuesReferences`, so the merge would have closed it;
- the pre-push `sdd-spec-gate` hook failed with an archive-on-merge violation, because a PR that closes the
  issue of an active spec must archive it in the same change.

The words "does not" changed nothing. `close #1727` was a closing reference.

## Solution

The body was reworded to "#1727 stays open with it", with no closing verb next to the number, and
`closingIssuesReferences` went empty. Re-check it after any body edit:

```bash
gh pr view <N> --json closingIssuesReferences --jq '.closingIssuesReferences|length'
```

## Takeaway

Never put `close`, `fix` or `resolve` (in any tense) directly before an issue number you mean to keep open,
not even in a negation. Say `Part of #N` or `Refs #N`, and describe what stays open without the verb.
