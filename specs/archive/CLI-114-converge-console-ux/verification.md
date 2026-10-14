---
tags: [spec, verification, templates]
created: "2026-10-14"
---

# Verification - CLI-114-converge-console-ux

## Evidence

PR 1 (macOS arm64, 2026-10-14):

- [x] AC1 -> `TestColorEnabled` (7 cases: a terminal, no TERM, a pipe, NO_COLOR, NO_COLOR=false, an empty NO_COLOR, TERM=dumb), `TestIsTerminal_NeverForANonFile`, `TestPainter`, `TestDuration`. All failed to compile before `internal/console` existed
- [x] AC2 -> `TestReport_Golden` (plain, plain-verbose-clean, color, color-verbose-clean). Its golden files were committed (`01dcfa7a`) before doctor moved onto `console` (`6df8d647`) and stayed byte-identical after it
- [x] AC3 -> `TestConvergeConsole_Colour`. The plain golden was captured from the unchanged `printConvergeReport` (`de3bbeae`) and is byte-identical after colour. The coloured output, with its SGR codes removed, equals the plain output. `TestConvergeConsole_ARunOnATerminalIsColoured` covers a plan; `TestConvergeConsole_AnAppliedRunColoursTheTerminalNotTheTranscript` covers an apply: the terminal gets the codes and the transcript none
- Mutations, each caught:
  - `runConverge` printing with a zero `Painter` -> `ARunOnATerminalIsColoured` FAIL
  - colour decided on the transcript tee, `convergePainter(out)` -> `AnAppliedRunColoursTheTerminalNotTheTranscript` FAIL
  - the tag printed unpainted -> `TestConvergeConsole_Colour` FAIL
- [x] AC4 -> `TestRun_RecordsEachStepsDuration` (an injected clock, `now`, steps 1.5s; the step never reached holds 0), `TestWriteReport_HoldsEachStepsDurationInMilliseconds` (reads `duration_ms` back from the file), and the console goldens (`converge-{apply,plan}-{plain,color}.golden`: `2m05s`, `850ms`, `12.3s`, `40ms`, blank for "not reached"). Mutations, each caught:
  - `runOne` not setting `Duration` -> `TestRun_RecordsEachStepsDuration` FAIL
  - `duration_ms` written as 0 -> `TestWriteReport_HoldsEachStepsDurationInMilliseconds` FAIL
  - a zero duration printed as `0ms` instead of blank -> golden FAIL
  - the name column fixed at 16 -> `TestConvergeConsole_ALongNameWidensTheColumn` FAIL
  - a step's duration recorded unrounded or under 1ms (a 300µs step as 0, which the JSON reads as "not reached"; a 1.6ms step as "2ms" beside `duration_ms` 1) -> `TestRun_RecordsADurationAtMillisecondResolution` FAIL
  - the long-name test passing with neither detail printed -> `TestConvergeConsole_ALongNameWidensTheColumn` FAIL (it asserts the index is found)
- [x] AC6 -> `TestConvergeConsole_FailureSummary`: two failed steps (one a revisit, `after tools: …`) are listed again under `failed:` after the counts; the same run with no failures prints no summary. `TestConvergeConsole_AFailedRunSummarisesBeforeTheLog` runs a failing `dotf converge` and asserts `failed:` comes before `log:`. Mutation: the summary never printed -> both FAIL
- [x] AC5 -> `TestRunObserved_ReportsEachStepItRuns` (start then done per step, in run order, a revisit's final "after tools:" entry, a step the OS does not support done without a start, the unreached step never done), `TestRunObserved_AGateSkipsWithoutRunning`, and `TestConvergeConsole_Streams` (goldens `converge-stream-{terminal,pipe}.golden`: on a terminal `→ tools` precedes the step's own output and the result line follows it; a pipe gets no `→` lines; the unreached step printed exactly once). Mutations, each caught:
  - `→` printed on a pipe too -> `converge-stream-pipe.golden` FAIL
  - `finish` dropping the entries Done never saw -> the once-only count FAIL
  - `finish` reprinting every entry -> the once-only count FAIL
  - `revisit` not calling `Done` -> `TestRunObserved_ReportsEachStepItRuns` FAIL
  - `Done` not counting what it printed -> golden FAIL
- [x] AC7 -> `converge.Run` keeps its signature and delegates to `RunObserved` with a no-op observer; no existing test in `internal/converge` was edited, and `go test ./...` passes on every PR

## Test status

- PR 1: `go build`, `go vet`, `GOOS=windows go vet`, `go test ./...` -> exit 0, every package ok; `golangci-lint run` 2.12.2 after `cache clean` -> 0 issues; bats over the 7 files that reference converge's output (`git grep -l -E 'dotf converge|converge on |converge plan' -- 'tests/*.bats'`) -> 1..252, 252 ok
- PR 2: `go build`, `go vet`, `GOOS=windows go vet`, `go test ./...` -> exit 0; `golangci-lint run` -> 0 issues; bats over the same 7 files -> 1..252, 252 ok
- PR 3: `go build`, `go vet`, `GOOS=windows go vet`, `go test ./...` -> exit 0; `golangci-lint run` -> 0 issues. The PR 1 and PR 2 golden files did not change: a finished report prints exactly as before
- PR 3 smoke test (`converge --plan` under `script -q /dev/null`, a pty): `→ checkout`, then its `[SKIP]` line, `→ records-mirror`, then its `[CHANGE]` line, and so on, each step announced before it runs
- PR 2 smoke test (`converge --plan` from the Mac's checkout, piped): every step carries its duration (`tools 5.7s`, `agent-extensions 4.1s`, `configs-deploy 431ms`), and the columns stay aligned past `workstation-target`, an 18-character name that overflowed the old fixed 16-column width
- Manual smoke test (the branch built into `$TMPDIR`, run from the Mac's checkout, `converge --plan`, a dry run): piped -> the tags bare, the same text as before; under `script -q /dev/null` (a pty) -> `\x1b[36m[SKIP]`, `\x1b[35m[CHANGE]`, `\x1b[32m[ OK ]` with the columns aligned; `NO_COLOR=1` under the pty -> bare again
- After all three PRs merged (main at `3cf4bcd1`, macOS arm64, 2026-10-14): each `features.json` verification command -> exit 0 (f1-f7); `go build`, `go vet`, `GOOS=windows go vet` -> exit 0; `golangci-lint run` -> 0 issues; bats over the 6 `.bats` files that reference `dotf converge` (`git grep -l 'dotf converge' -- tests`) -> 1..179, 0 not ok
- Smoke test on main (`converge --plan`, a dry run, from the Mac's checkout): piped to `cat` -> 0 escape bytes and no step announcement (the one `→` is inside records-mirror's own detail); under `script -q /dev/null` (a pty) -> 14 lines carry SGR codes, each of the 13 steps darwin supports announced with `→ name` before its result line. Both end `1 to change, 11 ok, 0 opaque, 2 skipped, 0 failed`

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- Colour is decided on the raw writer in `runConverge`, before the run log's tee, so the transcript on disk stays plain whatever the terminal gets.
- The `→ name` announcement prints on a terminal only. A pipe keeps the finished-report shape, so scripts and `from-zero.yml`, which read the output, see what they saw before.
- No `\r` ticker: it would garble the subprocesses' own output. The heartbeat alternative is the owner's question, CLI-115 (#2480).

## Review dispositions

Independent review (`review.md`, nan/qwen3.8-flash through pi, 2026-10-14): **PASS-WITH-GAPS**. One disposition per finding:

| Finding | Disposition |
|---|---|
| Major, THEORETICAL: a step's child output can land after its result line (pump vs direct write) | **Ticketed**, CLI-116 (#2486), with the design notes: a pump-acknowledged barrier, not routing the printer through the pipe (that would put SGR codes in the transcript), and a `runConverge` test with a real child. Cross-referenced on CLI-115 (#2480). |
| Minor, REAL: `printConvergeReport` has no production caller | **Ticketed**, REFACTOR-019 (#2487). Moving the five golden sites onto the production renderer can change golden bytes (the two derive the column width differently), so it is its own change. |
| Minor, REAL: two terminal probes, doctor's `console.IsTerminal` and converge's `convergeTerminal`, and a double probe in `runConverge` | **Ticketed**, same REFACTOR-019 (#2487): the same code area, one change. |
| Minor, THEORETICAL: a terminal run's transcript holds the `→ name` lines a piped run's does not | **Applied, policy decided:** the transcript records what the terminal showed, uncoloured. `TestConvergeConsole_AnAppliedRunColoursTheTerminalNotTheTranscript` now requires the whole line `→ checkout` in the transcript (a bare `→` matched records-mirror's own detail, so the first version of the assertion survived its mutation); removing the announcement fails it. `docs/runbooks/guide-new-machine.md` says the transcript keeps those lines. |
| Minor, REAL: the `from-zero.yml` measurement omits `.changed` | **Applied as a correction here.** The sentence is in `proposal.md` §3, which this verdict closes. `from-zero.yml:168` also reads `.changed` (`jq -e '.result == "ok" and .changed == 0'`). The conclusion stands: `duration_ms` is a new field and no reader selects it. |
| Question: `features.json` rows stay `pending` | **Existing ticket**, GUARD-010 (#1470): nothing in the harness writes `state` or `evidence`. Every row's verification command exits 0 above (f1-f7). |

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-424-a-golden-file-with-an-extension-escapes-the-extensionless-eol-guard.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: a console presentation change inside one command; no contract or dependency outlives it
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: converge's console is specific to dotf

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/CLI-114-converge-console-ux/` -> `specs/archive/CLI-114-converge-console-ux/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)
