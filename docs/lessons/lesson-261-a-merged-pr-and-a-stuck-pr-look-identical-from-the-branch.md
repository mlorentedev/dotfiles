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

**The PR had merged.** `merged=true`, `merged_at=2026-09-03T03:13:46Z`.

Two ordinary facts produced every symptom between them, and neither needed a GitHub bug:

1. **The merge.** A merged PR pins its head, stops reporting `mergeable`, and never re-runs a check.
2. **The merge commit I could not account for was the "Update branch" button.** `git log -1
   --format='%an / %cn'` on it reads `Manu Lorente / GitHub` — authored by the repo owner in the web
   UI, committed by GitHub. It exists on the remote branch and in no local checkout, which is exactly
   the shape I read as corruption. The same thing happened again on the follow-up PR #1462 a few
   hours later; the second time it took one command to identify.

I had reached for an exotic explanation while two mundane ones were each a single query away.

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

And when a commit on the remote branch is not in your checkout, ask git who wrote it before assuming
the ref is damaged:

```bash
git log -1 --format='%an / %cn / %s' <sha>   # "… / GitHub / Merge branch 'main' into …"
```

A `%cn` of `GitHub` means a human pressed a button in the web UI. It is a normal commit that your
local clone has simply not fetched yet, and `git fetch && git merge --ff-only` ends the mystery.

Three corollaries:

- **`git push` succeeding tells you nothing about the PR.** A branch behind a merged PR still accepts
  pushes; they simply go nowhere anyone is looking.
- **A stale red check is not a stuck check.** It was true when it ran. If the tree it ran against is
  gone, the question is what replaced it, not why it has not re-run.
- **Pending is not failing.** "Update branch" restarts the whole suite, so a PR that was green
  minutes ago reads `mergeable_state=blocked` purely because the new run has not finished.

## Same family as lesson 256

This is *"probe the target, do not read the block's own comment"* applied to a PR: I inferred the
PR's state from symptoms rather than asking the PR. It is also the class GUARD-009 (#1448) tracks —
a claim (`the PR object is stuck`) that outlived any check against its referent. Recorded here
because the ticket for that class was filed in the same session that produced this instance.
