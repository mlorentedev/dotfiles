---
id: spec-TOOL-037-finding-anchor-drift
type: spec
status: archived
created: "2026-11-13"
owner: manu
issue: "mlorentedev/dotfiles#3455"
---
# TOOL-037: a triage row matches a finding whose anchor moved

## Intent

`dotf pr triage-queue` holds a PR until every REAL finding has a triage row with a reproduction, and it finds the row by the finding's `path:line`. It read GitHub's `line` only, which is the line as of the current head: it shifts when a later push adds lines above the finding, and it goes null once the finding leaves the diff. Measured on #3421 (2026-11-13): a REAL finding on `verification.md:39` went `line: null` after two pushes, the queue read it as line 0, `rowFor` returns no row for line 0, and the PR was held with "the triage record has no row for it" whatever the author wrote. Four more findings on `lease.go` sat at `line`/`original_line` 114/105, 95/89, 101/96 and 113/107 while the reviewer's comments still printed the originals. The review loop exists to produce new heads, so the gate failed closed on exactly the PRs doing it right, and the failure read as an author who never triaged.

## Scope

- `Comment` carries both anchors: `Line` (current, zero once outdated) and `OriginalLine` (the reviewer's, never moves). `fetchInline` reads `original_line` alongside `line`.
- `rowFor` matches a row on the reviewer's anchor, `Comment.Anchors()`: `original_line`, or `line` only when GitHub reports no original. The current line is not a key, because a push can move one finding onto the line another was anchored to and one row would clear both (REAL finding on #3457, reproduced before the fix). The path rules (bounded name, directory tail, shared base names) are unchanged.
- `triage-queue` prints a finding as the reviewer did, `Comment.Where()`: the reviewer's line once the current one is gone, the path alone for a file-level comment.
- Out of scope: a file-level comment (`line` and `original_line` both null) parsed as a REAL finding still has no anchor a row can name. No configured reviewer posts REAL findings that way today; noted on #3455 for a path-only match if one starts to.

## Checklist
- [x] `restInline.OriginalLine`, carried into `Comment.OriginalLine`
- [x] `Comment.Anchors()` and `rowFor` trying each anchor (`rowForLine` keeps the matching rule)
- [x] `Comment.Where()` used by the two places `dotf pr` prints a finding's location
- [x] `TestFetchKeepsTheOriginalLineOfAFindingThatLeftTheDiff` (null `line`, row under the original)
- [x] `TestFetchMatchesARowOnTheReviewersLineOfAShiftedFinding` (`line` 9 / `original_line` 4: a row keyed by 4 clears, one keyed by 9 holds)
- [x] `TestFetchDoesNotClearAFindingOnAnotherFindingsRow` (A 4/9 and B null/4 with one row `:4`: A stays held)
- [x] Lesson 598

## Evidence

```
cd cli && go build ./... && go vet ./internal/prtriage/ ./internal/cmd/   -> ok
cd cli && go test ./internal/prtriage/ ./internal/cmd/                      -> ok
golangci-lint run ./internal/prtriage/... ./internal/cmd/...                -> 0 issues
probe: original_line fallback removed -> TestFetchKeepsTheOriginalLineOfAFindingThatLeftTheDiff fails
       ("the triage record has no row for it")
go run ./cmd/dotf pr triage-queue --pr 3421 from this branch                -> [OK] no reviewer output is awaiting a disposition, exit 0
the same command built from main on the same PR                             -> exit 1, "REAL ...verification.md:0: the triage record has no row for it"
(the pinned release, 0.80.0, predates the stricter row match and does not hold it)
```

## Review dispositions

Independent review (`review.md`, `nan/mimo-v2.6-flash`, reviewed_sha `250c2f58`): PASS-WITH-GAPS, no Blocker, no REAL Major. This section was written after the review; the reviewed `spec.md` is the one `review-request.json` digests.

- Major, THEORETICAL, the print sites have no test asserting the reviewer's line: **applied**. `TestPrTriageQueue_PrintsTheReviewersLineOfAFindingThatLeftTheDiff` (`cli/internal/cmd/pr_triage_scoped_test.go`) feeds a `Line: 0, OriginalLine: 39` finding and asserts `verification.md:39` is printed and `:0` never is, the incident's headline symptom.
- Minor, REAL, a file-level REAL finding stays unmatchable: **deferred by declaration**, out of scope above; the trigger (a reviewer posting REAL findings file-level) is recorded on #3455.
- Minor, THEORETICAL, rows keyed by a shifted current line stop matching once this lands: **applied in #3457**, whose `## Migration` section records the repo-wide measurement (the fixed queue held the same open PRs main held, none newly held).
- Minor, REAL, the checklist says "trying each anchor" and the commit message names `restInline.anchor()`: **declined as a text edit**, the final `Anchors()` returns one anchor and the commit message is history; the Scope bullet is the accurate statement.

## Promotion candidates
- [x] Lesson? `yes: docs/lessons/lesson-598-a-review-anchor-moves-when-you-push.md`
- [x] ADR? `no: the queue keeps keying rows by the finding's anchor; it only reads both anchors GitHub reports`
- [x] Pattern? `no: one tool's reader of one API field`
