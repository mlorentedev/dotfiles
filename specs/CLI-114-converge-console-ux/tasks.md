---
tags: [spec, tasks]
created: "2026-10-14"
---

# Tasks - CLI-114-converge-console-ux

> TDD order. One task = one focused commit. Three PRs, each green and shippable alone.

## Setup

- [x] Branch created from main: `feat/converge-console-ux`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] The owner's heartbeat question is recorded on #2460 (Risks / open questions)

## PR 1: one console package, doctor on it, converge in colour

- [x] [AC1] Failing table test: `console.ColorEnabled(getenv, isTerminal)` off for a non-terminal, a non-empty `NO_COLOR` (per no-color.org), and `TERM=dumb`; on otherwise
- [x] [AC1] Implement `cli/internal/console`: the colour decision, the palette (doctor's ANSI constants move here) and a `Style(kind, text)` styler
- [x] [AC2] Capture doctor's output with colour on and off on a fixed set of checks as golden files, **before** the move
- [x] [AC2] Move `doctor/report.go` onto `console`; the golden files stay byte-identical
- [x] [AC3] Failing golden tests: converge's report to a terminal (colour) and to a pipe (no escape bytes, same text)
- [x] [AC3] Decide colour on the raw writer in `runConverge`, before `beginRunLog` builds the tee, and style the tags in `printConvergeReport`
- [x] Fix the stale comment at `cmd/converge.go` `saveConvergeReport`: "last.json, which doctor reads". Nothing in Go reads it; `from-zero.yml` does

## PR 2: durations and the failure summary

- [x] [AC4] Failing test: `runOne` sets `Entry.Duration` from an injected clock; `WriteReport` emits `duration_ms`, read back from the file
- [x] [AC4] Implement; the console prints the duration beside each step (`console.Duration`: `850ms`, `12.3s`, `2m05s`)
- [x] [AC6] Failing golden test: a run with two failed steps ends with a summary (detail, error), then `log:`; a clean run prints none
- [x] [AC6] Implement the summary in the converge renderer

## PR 3: streaming progress

- [x] [AC7] `converge.RunObserved(reconcilers, env, dryRun, obs)`; `Run` delegates with a no-op observer. Existing tests unchanged
- [x] [AC5] Failing golden test: a terminal run where the middle step writes its own output and fails. Expect `→ name` before that output, the result line after it, and the "not reached" entries printed once at the end
- [x] [AC5] Implement the observer in `cmd/converge.go`: `Start` prints `→ name` on a terminal only, `Done` prints the result line, and the end flushes the entries never seen
- [x] [AC5] `revisit` entries reach `Done` too (they go through `runOne`)

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] `features.json` lists each AC with a non-vacuous verification command
- [ ] `go build`, `go vet`, `GOOS=windows go vet`, `go test ./...`, `golangci-lint run` -> exit 0
- [ ] bats over the files that reference `converge` output (`git grep -l 'dotf converge' -- tests`)
- [ ] A real `dotf converge --dry-run` on the Mac (terminal) and piped to `cat`, recorded in `verification.md`
- [ ] `verification.md` filled in
- [ ] Independent adversarial review before archive (`dotf spec review CLI-114-converge-console-ux`)
