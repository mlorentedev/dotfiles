---
id: "BUG-166-review-base-widen"
type: spec
status: verifying
created: "2026-11-11"
issue: "mlorentedev/dotfiles#3258"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# BUG-166: the review base widens to where the branch left main

## Why

<!-- from issue #3258: spec review base is the parent of the spec-folder commit, so a spec committed after the code hides the production diff -->

`dotf spec review` diffs from the parent of the commit that first added the spec folder. When the spec folder is committed after the code it describes (fix first, then tests and spec, as in #3258), that base sits after the fix, so the reviewer certifies a diff that holds the tests and the spec but not the production change. A verdict over a diff that excludes the code it judges is worse than no review: it arrives looking like coverage.

## What

After the spec-folder trace resolves a base, the resolver computes the fork point, `git merge-base HEAD <default branch>`, where the default branch is the first of `origin/HEAD`, `origin/main`, `origin/master` that exists. When the fork point is a strict ancestor of the traced base, the base widens to the fork point. Otherwise the traced base stands. The base only ever moves earlier, never later.

When the base widened, the commits between the fork point and the traced base also count as the spec's own commits (they are the work that preceded the spec on its branch), and the launch's `Scope:` line and the reviewer prompt say the base was widened and from which commit.

## Out of scope

- Recording `base_sha` at `dotf spec init` (#1551). It addresses a base that is too early, not too late, and would not fix #3258: an init after the code records a base after the code.
- A spec whose code merged to main in an earlier PR than its spec folder (ts-bridge#490). The fork point there is after the code too; no local rule recovers it.
- Changing what the archive gate checks. It reads the base recorded in the request and stays as it is.

## Risks / open questions

- A stale or absent `origin/*` ref. A stale ref gives an older fork point, which is still an ancestor, so the base can only widen further; that costs reviewer budget and hides nothing. No ref at all keeps today's base. Resolved: both are safe by the widen-only rule.
- A review on main after a squash: the fork point is HEAD itself, which is never an ancestor of the traced base, so nothing changes. Resolved.
- A branch that merged main after the spec landed (the HARNESS-142 shape): the fork point is newer than the traced base, so it is not its ancestor and the base stands. Resolved.

## Acceptance criteria

- [ ] AC1: a spec folder committed after a fix commit on a branch resolves to a base at or before the fix, and the fix commit (which names no issue) is listed among the spec's own commits.
- [ ] AC2: a base never moves later than the traced one: on main after a squash, after a merge of main into the branch, and without any `origin/*` ref, the resolved base equals the traced base.
- [ ] AC3: the refusals `BaseNotCommitted`, `BaseRootCommit`, `BaseUntraceable` and the BUG-108 rename trace behave as before.
- [ ] AC4: `dotf spec review --dry-run` on the #3258 shape names a base whose `git diff --stat <base>...HEAD` includes the production file, and its `Scope:` line says the base was widened.

## References

- Issue #3258; related #1551 (out of scope), HARNESS-158 (own commits), BUG-108 (rename trace), CLI-132 (initial review).
