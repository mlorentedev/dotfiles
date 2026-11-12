---
tags: [spec, verification, templates]
created: "2026-11-12"
id: "MEMORY-018-handoff-line-budget"
type: spec-verification
status: active
owner: manu
---

# Verification - MEMORY-018-handoff-line-budget

## Evidence

- [x] AC1 -> `TestCheckMemoryBudgetRefusesAWritePastTheLineCap`. The fixture is 201 lines and well under 25,000 bytes, which the test asserts, so only the line budget can refuse it. The refusal states "201 lines" and "the 200 a session start loads", and names `old` before `new`.
- [x] AC2 -> `TestCheckMemoryBudgetLetsAFilePastTheLineCapShrink` covers a lowered count, a count held with more bytes, and a refused growth. `TestCheckMemoryBudgetPassesAFileAtTheLineCap` covers exactly 200 lines passing and 200 plus an unterminated line refused.
- [x] AC3 -> `TestCheckMemoryBudgetStillRefusesPastTheByteBudget`. The existing `TestCheckMemoryBudget`, `TestTheBudgetRefusalSuggestsOnlyDatedThreadsAndHowToArchiveEach` and the cmd-level `TestHandoffWriteRefusesAWriteThatCrossesTheByteBudget` are unchanged and pass.

## Test status

- `go test ./...` in `cli/`: every package ok. `go vet` and `golangci-lint run` report 0 issues, natively and with `GOOS=windows`.
- Smoke test, on a copy of the real dotfiles MEMORY.md (217 lines after the 2026-11-12 archive) with its threads directory: `go run ./cmd/dotf mem handoff-write --agent claude` with a one-line body exits 1. The message is "this write takes MEMORY.md to 223 lines, past the 200 a session start loads". It names main@egw-len029, feat-root-commit-review and docs-3191-workspace-root-contract as the oldest dated threads. The copy is left at 217 lines.
- No regressions.

## Decisions made during implementation

- The byte check runs first. A write past both budgets gets the byte message, which is unchanged. Each budget refuses only a write that grows the file in its own measure, so a file over one budget can still be rewritten in place.
- `lineCount` counts a trailing unterminated line, because a 200-line read sees it.
- Once a release carries this, the dotfiles MEMORY.md (217 lines) refuses every handoff write that adds a line until threads are archived. The refusal names them. That is the intended pressure, the same as the byte cap.

## Review dispositions

The review, by nan/deepseek-v4-flash at `27e4d097`, gave PASS-WITH-GAPS.

1. Minor: `handoff-write --help` described only the byte budget. Applied: the help now names both budgets and the hold-or-shrink allowance.
2. Minor: ADR-056 described one refusal rule. Applied: Decision 8 records the line budget and the order the two are checked in, and the PR's Knowledge section names the ADR.
3. Minor: no test fixed which message a write past both budgets gets. Applied: `TestCheckMemoryBudgetNamesTheByteBudgetWhenBothAreCrossed`.
4. Minor: no `## Promotion candidates` section. Applied: added below.
5. Question: with MEMORY.md at 217 lines, every handoff write that adds a line fails until threads are archived. Accepted, as the proposal's Risks says. The owner is told in the PR body and the run report. Automatic archiving is #3393.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: the reason is in the constant's comment and ADR-056 Decision 8
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: docs/adr/adr-056-a-handoff-thread-is-stored-in-its-own-file.md
- [x] New pattern candidate for `00_meta/patterns/`? no: specific to this tool's MEMORY.md budget
