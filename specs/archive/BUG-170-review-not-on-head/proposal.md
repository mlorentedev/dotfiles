---
id: "BUG-170-review-not-on-head"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-11-12"
issue: "mlorentedev/dotfiles#3243"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# BUG-170: a review that did not run on the head is not a clear queue

## Why

<!-- from issue #3243: pr triage-queue: a failed review on the head reads as a clear queue -->

`dotf pr triage-queue` counts only reviewer output that exists. When the `review` check fails on the head without publishing anything, there is nothing to triage, and the queue reads clear. In mlorentedev/ts-bridge#687 a landing script trusted that exit 0 and squash-merged head `69690fe`, whose `review` check had failed. The last commit landed with no reviewer pass. `dotf pr land` refuses a failed `review`, but it accepts a skipped one or none at all.

## What

Both commands now ask whether a review ran on the head, as well as whether its output was triaged. The owner decided the rule on 2026-11-12: any `review` check on the head that is not `success`, skipped included, makes the queue unanswerable, and `pr land` refuses the same way.

- `triage-queue`, repo-wide and `--pr N`, lists an open, non-draft, non-exempt PR when the newest `review` check run on its head is not `success`. That covers failed, cancelled, skipped, still running, and absent. The reason names the head and the reading, so the command exits 1.
- `pr land` refuses a PR whose `review` check is skipped or absent, with the reason `review did not run on <sha>`. A failed check is already refused, and a pending one is already waited for.
- The review check is named `review` when the job lives in the repository's own workflow (ts-bridge#722), and `review / review` when it is called from the reusable workflow (kubelab, hive). One matcher recognises both, and both packages share it.

## Out of scope

- Draft PRs. `pr land` refuses them anyway. Measured on 2026-11-12, dotfiles' drafts read `review` success because the `reviewable` step skips inside a green job. Other repositories may differ, and a draft is not asking to land.
- A registry field naming the check. `harness/**` is owner-landed, so this would be a follow-up if a repository names the job anything else.
- The escape that `pr land` prints, `manualMerge`. It stays the human's way past a red reviewer check (#3108). Only its literal moves to the shared matcher.

## Risks / open questions

- Absent check: a repository with a reviewer registry but no reviewer job would list every PR. That is the intended fail-closed reading ("never claim a review you did not get"). Resolved.
- Cost: one REST call per open, non-draft PR that the queue would otherwise report clear. Bounded by the open-PR page. Resolved.
- A check read that fails returns an error, never an empty queue. That is the package's standing rule. Resolved.

## Acceptance criteria

- [ ] AC1: `triage-queue` lists a non-draft PR whose newest `review` check run on the head is not `success`, including absent and skipped, and the reason names the head SHA and the reading. That holds both repo-wide and with `--pr N`. A run superseded on the same head by a successful one does not list it.
- [ ] AC2: A PR whose diff is an exempt signature, or that is a draft, is not listed for its head review, so a release PR with a skipped `review` stays quiet.
- [ ] AC3: `pr land` refuses a PR whose `review` check is skipped or absent, with `review did not run on <sha>`. Both check names, `review` and `review / review`, are recognised by one shared matcher, which `manualMerge` uses too.

## References

- #3243, ts-bridge#687 (the incident), #3108 (manualMerge), #2549 (`--pr N`)
- `cli/internal/prtriage/fetch.go`, `cli/internal/prland/prland.go`
