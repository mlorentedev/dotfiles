---
tags: [spec, verification, templates]
created: "2026-11-12"
id: "BUG-170-review-not-on-head"
type: spec-verification
status: active
owner: manu
---

# Verification - BUG-170-review-not-on-head

## Evidence

- [x] AC1: `TestReviewOnHeadHoldsAQueueTheCommentsLeaveClear` runs eight readings, each both repo-wide (`fetchWith`) and with `--pr` (`fetchPRWith`). Failed, skipped, still running, absent, the reusable workflow's name failed, and a success followed by a failure on one head are all held, with a reason naming `0123456` and the reading. A cancelled run followed by a success, and a look-alike `preview` check, leave it clear. `TestReviewOnHeadFailsWhenTheChecksCannotBeRead` makes a failed read, a full page, and an unknown head commit errors rather than a clear queue. `TestPrTriageQueue_AHeadReviewRowPrintsNoZeroTime` checks the printed row.
- [x] AC2: `TestReviewOnHeadIgnoresADraft` shows a draft with a skipped review is not held and its check runs are never read. `TestReviewOnHeadLetsAnExemptDiffThrough` shows a release-signature diff with a skipped review stays quiet, while a non-exempt diff with the same reading is held.
- [x] AC3: `TestDecide_ReviewDidNotRunOnTheHead` covers skipped under both names, absent, and only a look-alike, which all refuse with `review did not run on abc123`; `review / review` passing lands. `TestIsReviewCheck` pins the matcher. `TestPrLandOptions_UntriagedSkipsAHeadNoReviewRanOn` shows `pr land` does not also report such a PR as untriaged output. `manualMerge` uses the shared matcher.

## Test status

- `go test ./...` in `cli/`: 59 packages ok. `go vet` and `golangci-lint run` report 0 issues, natively and with `GOOS=windows`, after `golangci-lint cache clean`.
- Mutation check: with `withHeadReview` disconnected from `pendingOf`, 14 subtests fail. With `withoutExempt` restored to re-evaluating comments, `TestReviewOnHeadLetsAnExemptDiffThrough` fails.
- Smoke test against live dotfiles: `go run -C cli ./cmd/dotf pr triage-queue` held #3410 with `no review covers the head: the review check on a2c3c76 reads in_progress`, and listed the other four for their reviewer output. The drafts (#3403, #3399 and #3387 read `review` success) and release #3383 (`skipped`, exempt signature) were not held for their head review.

## Decisions made during implementation

- Both check names are in use, measured on live heads: `review` in ts-bridge (#722, and the #687 incident) and `review / review` in kubelab and hive. One matcher, `prtriage.IsReviewCheck`, serves both commands.
- The head review is read inside the existing comment fan-out, so the session-start probe pays no extra serial round trip. Drafts are not read at all.
- `withoutExempt` now drops a PR only when its diff is exempt, instead of re-evaluating the comments with the files attached. Those were equivalent before this change. Now that a PR can be pending for its head alone, a re-evaluation would wrongly release it.
- `pr land` reads its untriaged list through the `triageFetch` seam and skips PRs held only for their head review (`Status.HeadUnreviewed`). `Decide` refuses those in its own words, so "reviewer output awaits triage" is not printed for output that does not exist.
- The printed row omits the time when the status has none. A head-review hold has no reviewer output to date, and printed `0000-12-31` in the first smoke run.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-592-nothing-to-triage-is-not-a-review-that-found-nothing.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: it applies the existing pr-stewardship rule ("a notice that no review ran leaves the PR unreviewed") to the queue, with the owner's decision recorded in proposal.md
- [x] New pattern candidate for `00_meta/patterns/`? no: the cross-project form is already the pr-stewardship rule

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-170-review-not-on-head/` -> `specs/archive/BUG-170-review-not-on-head/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
