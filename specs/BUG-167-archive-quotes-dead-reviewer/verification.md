---
tags: [spec, verification, templates]
created: "2026-11-11"
id: "BUG-167-archive-quotes-dead-reviewer"
type: spec-verification
status: active
owner: manu
---

# Verification - BUG-167-archive-quotes-dead-reviewer

## Evidence

- [x] AC1 -> `TestArchiveGateQuotesWhatADeadReviewerSaid`.
- [x] AC2 -> `TestReviewerLastWordsQuotesTheEnd` (tail of a long transcript; head of a long stderr with multi-byte runes, checked with `utf8.ValidString`).
- [x] AC3 -> `TestArchiveGateNamesATurnCapOnADetachedRun`, `TestArchiveGateIsSilentAboutACapForAFinishedRun`, `TestArchiveGateNamesTheCapOnlyForAMissingVerdict`, unchanged.

## Test status

- `cd cli && go test ./internal/spec ./internal/cmd`: ok, ok.
- `golangci-lint run ./...` and `GOOS=windows go vet ./... && GOOS=windows golangci-lint run ./...`: 0 issues.
- A live reproduction with a fake failing runner was not run: the session's permission classifier denied it. The issue records the code-read measurement.

## Decisions made during implementation

- The helper keeps both ends rather than one: PR-Agent on #3322 showed that quoting only the tail would hide a startup failure's cause, which `confirmLaunched` relies on.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: an error message change; #1511 holds the measurement.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: no decision between components.
- [x] New pattern candidate for `00_meta/patterns/`? no: one tool, one occurrence.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/`
- [ ] Bitácora ticket updated (Refs #1511; the issue stays open for options A and B)
