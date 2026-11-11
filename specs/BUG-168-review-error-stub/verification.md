---
tags: [spec, verification, templates]
created: "2026-11-11"
id: "BUG-168-review-error-stub"
type: spec-verification
status: active
owner: manu
---

# Verification - BUG-168-review-error-stub

## Evidence

- [x] AC1 -> commit `c36eb7b5` / `TestDeadlineWritesAnErrorStubWhenTheRunSignedNothing` (exit 0, exit 3, deadline; stub reaches the caller's folder through the copy-back), `TestWriteErrorStubRecordsTheRunThatSignedNothing` (both slots), `TestWriteErrorStubWithoutOutputOrRequest`
- [x] AC2 -> commit `c36eb7b5` / `TestWriteErrorStubLeavesAnExistingReviewFile` (PENDING and PASS), `TestDeadlineKeepsTheReviewersOwnFile`
- [x] AC3 -> commit `c36eb7b5` / `TestParseReviewRefusesAnErrorVerdictByName`, `TestVerifyReviewProducedReportsAnErrorStub`
- [x] AC4 -> commit `c36eb7b5` / `TestStoppedRunRecordNamesAnErrorStub`; `TestSpecDeadlineStopsAStalledRunAndNamesTheLastCall` now expects the ERROR record

## Test status

- Test suite: `cd cli && go build ./... && go vet ./... && go test ./...` -> all ok; `golangci-lint run ./...` and `GOOS=windows golangci-lint run ./...` -> 0 issues; `GOOS=windows go vet ./...` clean
- Mutation: making `WriteErrorStub` write over an existing file fails `TestWriteErrorStubLeavesAnExistingReviewFile`; the deadline tests failed before the stub was wired
- No regressions in existing test suite: yes

## Decisions made during implementation

- The stub is written by `spec deadline`, which already sees how every run ends and does the copy-back, rather than by the transcript sink, which sees only stdout.
- The turn-cap pool lookup moved into `runnerOf`, shared by the archive hint and the stub, so pi's turn cap still names itself once a stub exists.
- `VerifyReviewProducedIn` reports the stub as the refusal it is, not as a malformed file the reviewer wrote.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: the rule (a detached launcher writes the failure where people look) is the spec itself, and #1511 records the measurement
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: a fourth verdict value following PENDING's precedent, not a new architecture
- [x] New pattern candidate for `00_meta/patterns/`? no: single-project

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-168-review-error-stub/` -> `specs/archive/BUG-168-review-error-stub/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
