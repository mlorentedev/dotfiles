---
id: lesson-261
type: lesson
status: active
created: "2026-09-02"
owner: manu
tags: [lesson, git, github, pr, verification]
---

# A merged PR and a stuck PR are indistinguishable from the branch side

## What happened

While applying an independent review's findings to PR #1455, three pushes in a row produced the same
symptoms: the PR's `head.sha` never moved off a merge commit that was not in the branch's history,
`mergeable` read `null`, `mergeable_state` read `unknown`, and no new workflow runs fired. A red
`spec-gate` stayed red against a tree that no longer existed.

I diagnosed "the PR object is stuck in GitHub" and published that diagnosis in the triage comment,
along with a repair: merge the detached head back into the branch to restore continuity.

**The PR had merged.** `merged=true`, `merged_at=2026-09-03T03:13:46Z`. Every symptom follows from
that one fact, and none of them needed a GitHub bug to explain.

## Why it matters beyond the embarrassment

The merge landed *while the reviewer's findings were being applied*, so `main` took the feature
without the hook timeout, without a root-resolution fix, without three test-coverage repairs — and
with an active `specs/HARNESS-110/` for the issue the PR had just closed, which is exactly the state
the spec gate exists to prevent. The wrong diagnosis cost an hour; the merge timing cost a defect in
`main` that needed a second PR (#1462) to close.

## The rule

**Before diagnosing a PR's mechanics, ask whether it is still open.**

```bash
gh api repos/<owner>/<repo>/pulls/<N> --jq '"merged=\(.merged) merged_at=\(.merged_at) state=\(.state)"'
```

One field answers it. I asked after three failed pushes instead of before the first, because the
symptoms were exotic enough to feel like an infrastructure problem, and an exotic explanation
crowded out the ordinary one.

Two corollaries:

- **`git push` succeeding tells you nothing about the PR.** A branch behind a merged PR still accepts
  pushes; they simply go nowhere anyone is looking.
- **A stale red check is not a stuck check.** It was true when it ran. If the tree it ran against is
  gone, the question is what replaced it, not why it has not re-run.

## Same family as lesson 256

This is *"probe the target, do not read the block's own comment"* applied to a PR: I inferred the
PR's state from symptoms rather than asking the PR. It is also the class GUARD-009 (#1448) tracks —
a claim (`the PR object is stuck`) that outlived any check against its referent. Recorded here
because the ticket for that class was filed in the same session that produced this instance.
