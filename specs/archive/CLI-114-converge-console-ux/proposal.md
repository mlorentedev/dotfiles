---
id: "CLI-114-converge-console-ux"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-10-14"
issue: "mlorentedev/dotfiles#2460"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, converge, doctor, console]
template_version: "1.0"
wip_override: "the owner put the converge logs first (2026-10-14: 'los logs antes, luego las integraciones'); #2446 shipped the transcript half and this is the console half of the same ask (12 active, limit 10, 2026-10-14)"
---

# CLI-114-converge-console-ux

## Why

<!-- from issue #2460: CLI-114: a readable converge console: live progress, severity colour, step timing, failure summary -->

`dotf converge` is the one entry point on every OS (PLAT-001b), and its console is still a plain table printed after the last step (`printConvergeReport`, `cli/internal/cmd/converge.go`). During a run nothing says which step is working or for how long. A failed step is one line among the oks, and nothing says which step was slow. #2446 made every run durable (a redacted transcript and a JSON report). This spec makes the run readable while it happens, and gives converge and doctor the same look on every OS.

## What

1. **One console package**, `cli/internal/console`, that converge and doctor both use. It holds:
   - the colour decision: a terminal, `NO_COLOR` unset, `TERM` not `dumb`;
   - the palette and the status-tag styler;
   - the duration format.

   The colour decision takes `getenv` and `isTerminal` as parameters (the seam `MiseShimsDir` and #2446's `identityTerminal` already use), so golden tests set nothing in the process environment. Each command keeps its own vocabulary: doctor's pass, fail, warn, skip, info and fix; converge's ok, change, skipped, failed and opaque. The shared unit is the look, not a status enum.
2. **Severity colour on a terminal.** Converge's tags (`[CHANGE]`, `[ OK ]`, `[FAIL]`, …) are coloured by status. The tags stay, so a pipe, a CI log and the transcript read the same text. The transcript already strips ANSI (`runlog.go`, `writeLine`), so one writer serves both. The colour decision is made on the raw writer, before `beginRunLog` wraps it in the transcript tee, because a tee is never a terminal.
3. **Per-step duration.** `converge.Entry` gains `Duration`, measured around `Reconcile` and `Probe` in `runOne`. The console shows it beside each step. `<stamp>.json` and `last.json` carry it as `duration_ms`, an additive field: its one programmatic reader, `from-zero.yml`, selects `.result` and `.entries[].status` with `jq`, and no Go code reads the file back (measured 2026-10-14, `git grep last.json`). The JSON stays the contract, and the console is a view of it.
4. **Streaming progress.** Each step prints its line when it finishes, not all lines at the end. On a terminal, a step also prints a `→ <step>` line when it starts, so a long step is visibly the one running.
   - `converge.Run` keeps its signature. `RunObserved` takes an observer (`Start(name)`, `Done(Entry)`), and `Run` calls it with a no-op.
   - Entries that `Run` appends without running a step ("not reached", a plan's gate) never reach `Done`. The renderer prints every entry it has not printed when the run ends, so no line is lost.
5. **A failure summary at the end.** After the counts, every failed step comes again with its detail and its error, then the `log:` path. A failure is never only one line inside the table.

## Out of scope

- **An elapsed-time ticker on the running step.** #2460 asks for "how long it has taken" while the step runs. A `\r`-rewritten line shares the terminal with the subprocesses' own output (installers, `mise`, `git`) and would garble both. This spec prints the start line and the duration at the end. A heartbeat line every N seconds, with no `\r`, is an owner question recorded below, not built here.
- **A remedy per failure.** `Reconciler` returns an error and a detail; no reconciler declares a remedy. A `Remedy` field changes the contract of every reconciler, so it gets its own spec when wanted. TOOL-026 (#2355, a structured error catalog) is its natural home.
- **Enabling VT processing on legacy Windows consoles.** Doctor already colours on Windows without it, and this spec inherits that behaviour unchanged.
- **Restyling doctor's layout.** Doctor moves onto the shared package with byte-identical output; only the code moves.

## Risks / open questions

- **Interleaving.** A step's subprocess output lands between its `→` line and its result line. That is wanted, because it shows which step printed what. The golden tests cover a step that writes to the console while it runs.
- **Owner question:** a heartbeat line (`… tools still running, 2m10s`) every 30 s on a terminal. It would serve the "how long it has taken" half without `\r`. Not decided here; it is CLI-115 (#2480), so the close of #2460 does not drop it.
- **Golden output.** Durations vary from run to run. The renderer takes a clock, so tests inject a fixed one and the golden files hold no real time.

## Acceptance criteria

- [ ] AC1: `cli/internal/console` decides colour from an injected `getenv` and `isTerminal`. It is off for a non-terminal, for a non-empty `NO_COLOR` (no-color.org; doctor reads it the same way today), and for `TERM=dumb`. Table-driven tests cover each case.
- [ ] AC2: doctor renders through `console`, and its output is byte-identical before and after the move, with colour on and off.
- [ ] AC3: converge's console colours each tag by status on a terminal and prints the same text, with no escape bytes, to a pipe. Golden files cover both.
- [ ] AC4: every `converge.Entry` carries `Duration`; the console prints it, and `<stamp>.json` and `last.json` hold `duration_ms`. A test reads the JSON back.
- [ ] AC5: on a terminal, each step's `→ <name>` line precedes its own output, and its result line follows it. Entries `Run` never ran ("not reached", gate skips) are still printed exactly once. A golden test covers a failing middle step.
- [ ] AC6: a run with failures ends with a summary listing each failed step's detail and error, then the `log:` path. A run without failures prints no summary.
- [ ] AC7: `converge.Run`'s signature and every existing caller are unchanged, and `go test ./...` passes without edits to tests that do not cover the new behaviour.

## References

- #2460 (this spec), #1098 (half (a), "the log"), #2446 (the transcript and JSON report this view reads)
- PLAT-001b (one entry point), TOOL-026 #2355 (the remedy catalog, out of scope here)
- `cli/internal/doctor/report.go` (today's colour policy), `cli/internal/converge/runlog.go` (ANSI stripping in the transcript)

<!-- archived 2026-10-14 — PR: https://github.com/mlorentedev/dotfiles/pull/2468 -->
