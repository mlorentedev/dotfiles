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

## Review dispositions

First review (nan/glm5.3-flash, FAIL) at 64da07dc:

1. Major, rune-boundary property untested: applied. `TestHeadAndTailCutsOnRuneBoundaries` puts both cut points inside a rune and refuses U+FFFD in the quote; a byte-cutting `headAndTail` fails it (mutation checked).
2. Major, a missing second signature quoted nothing: applied. The not-found branch for the second signer goes through the same hint; `TestArchiveGateQuotesADeadSecondSigner`, which fails with the wrap removed (mutation checked).
3. Minor, a relaunch that dies before tmux starts its command quotes the previous attempt's stderr: declined. That failure is the tmux command's own, and the launcher reports it directly at launch; archive is not where it surfaces.
4. Minor, the whole transcript was read to keep 800 bytes: applied. `readEnds` reads a small file whole and a large one by its first and last 4 KiB; `TestReviewerLastWordsQuotesBothEndsOfALongFile`.
5. Minor, invalid UTF-8 in the input reaches the quote: applied. `headAndTail` passes its result through `strings.ToValidUTF8`; covered in `TestHeadAndTailCutsOnRuneBoundaries`.
6. Minor, quoted output can contain a line that looks like the truncation notice: declined. The quote reproduces what the reviewer wrote; escaping it would change that.

None of these touches `proposal.md`, `tasks.md` or `features.json`.

Second review (nan/glm5.3-flash, PASS) at a450087f, three Minors:

1. An unreadable reviewer pool skips the quote: declined. The archive gate already fails closed on a missing or unreadable tracked pool (`TestArchiveFailsClosedWhenATrackedPoolGoesMissing`), so that refusal is the one a user sees.
2. `readEnds` does not re-stat between `Stat` and `ReadAt`: declined. Transcripts are append-only, and the reviewer marked it surface-only.
3. The round-one decline of finding 6 stands, as the review records.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: an error message change; #1511 holds the measurement.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: no decision between components.
- [x] New pattern candidate for `00_meta/patterns/`? no: one tool, one occurrence.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved to `specs/archive/`
- [ ] Bitácora ticket updated (Refs #1511; the issue stays open for options A and B)
