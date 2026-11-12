---
id: "lesson-592-nothing-to-triage-is-not-a-review-that-found-nothing"
type: lesson
status: active
title: "Nothing to triage is not a review that found nothing"
created: "2026-11-12"
---

# Nothing to triage is not a review that found nothing

## Context

`dotf pr triage-queue` decided whether a pull request was clear by comparing its newest reviewer
output with its newest `## Review triage` comment. In ts-bridge#687 the `review` check failed on
the head and published nothing. The queue had nothing to compare, read clear, and a landing script
that trusted its exit 0 squash-merged an unreviewed head (#3243).

## Problem

A gate that only reads what was produced cannot see what was never produced. Absence of reviewer
output has two causes, and they need opposite answers: a review that ran and found nothing, or a
review that never ran. The queue gave both the same answer. Measuring the fix showed a second trap.
The same job reports as `review` when a repository runs it in its own workflow (ts-bridge#722), and
as `review / review` when it calls the reusable workflow (kubelab, hive). A rule keyed on the literal
`review` would have read every reusable-workflow repository as "no review check".

## Solution

Ask the head commit directly. Read its check runs, take the newest review run, so a cancelled run
superseded on the same head does not count, and hold the PR unless that run is `success`. Skipped,
failed, running and absent are all "no review covers the head". Drafts and exempt diffs are left out,
because they are not asking to land. Recognise the check by one shared matcher, `review` or
`<caller> / review`, which `pr land` uses too, so the two commands cannot disagree about which check
covers the head. Before encoding any check name, read it off live heads in more than one repository.

## Related

- #3243, BUG-170 (`specs/archive/BUG-170-review-not-on-head/`), ts-bridge#687
- `cli/internal/prtriage/headreview.go`, `cli/internal/prland/prland.go` (`reviewReasons`)
