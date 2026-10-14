---
spec: "CLI-114-converge-console-ux"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "f18a3d62cb6ded8560d74fa39f94d521d36b2324"
reviewer: "nan/qwen3.8-flash"
date: "2026-10-14"
---

## Adversarial review

**Scope**: `CLI-114-converge-console-ux` — a readable converge console: one shared `cli/internal/console` package, severity colour, per-step duration in console + JSON, streaming progress, failure summary.

**Sources**: `specs/CLI-114-converge-console-ux/{proposal.md,tasks.md,verification.md,features.json,review-request.json}`; `git diff 9fdc76ecb2b5c28f30a099b76cd8cf819466e573...HEAD` (base as stated by the launcher, `git merge-base` confirms it is an ancestor of HEAD).

**Diff-attribution note (not a finding against this change).** The launcher's base predates seven sibling PRs that landed on main while this spec was in flight, so `base...HEAD` is wider than CLI-114: `e7971467` (#2476 mise prune), `63c16e6e` (#2482 from-zero probes), `3683070a` (PI-002 archive), `6e91381f` (#2478 download retry), `41ec68f4` (#2473 claude deny), `8ba9e749` (#2449 herdr integrations), `ed901e6d` (#2474 reviewer pool). CLI-114 is `0dd87f49` (#2466 PR1), `12a24032` (#2467 PR2), `3cf4bcd1` (#2468 PR3) and `f18a3d62` (post-merge evidence); every file-level finding below was checked against those four commits. The only overlap with the siblings is `converge.Options.HerdrRun` / `extensions.go` (#2449), which the printer and the report writer do not touch. This is the launcher's base-resolution behaviour being honest about "the whole change", not scope creep by the implementer; it is recorded so the next round does not re-derive it.

### Spec and task alignment

All seven acceptance criteria are covered by named tests, and I re-ran each one fresh (`-count=1`) rather than trusting the cached suite:

| AC | Named test(s) I ran | Result |
|---|---|---|
| AC1 | `TestColorEnabled` (7 cases incl. `NO_COLOR=false`, empty `NO_COLOR`, `TERM=dumb`), `TestIsTerminal_NeverForANonFile` | ok |
| AC2 | `TestReport_Golden` (4 variants) + my own revert experiment (below) | ok, and byte-identity proven independently |
| AC3 | `TestConvergeConsole_Colour`, `…_ColourIsATerminals`, `…_ColourOnATerminal`, `…_ARunOnATerminalIsColoured`, `…_AnAppliedRunColoursTheTerminalNotTheTranscript` | ok |
| AC4 | `TestRun_RecordsEachStepsDuration`, `TestRun_RecordsADurationAtMillisecondResolution`, `TestWriteReport_HoldsEachStepsDurationInMilliseconds`, `TestDuration`, goldens | ok |
| AC5 | `TestRunObserved_ReportsEachStepItRuns`, `TestRunObserved_AGateSkipsWithoutRunning`, `TestConvergeConsole_Streams` (terminal + pipe goldens) | ok |
| AC6 | `TestConvergeConsole_FailureSummary`, `TestConvergeConsole_AFailedRunSummarisesBeforeTheLog` | ok |
| AC7 | `go test ./...` (41 packages, all ok/none failing); `internal/converge`'s edited test files (`catalog_test.go`, `tools_test.go`, `extensions_test.go`) belong to #2476/#2449, not to CLI-114 — the "no existing test edited" claim holds | ok |

Invariants I traced rather than sampled:

- **The tail-flush cannot double-print.** `Done` is called immediately before each `append` in both `RunObserved`'s loop and `revisit` (`converge.go:150-200`), so the entries `Done` prints are always a *prefix* of `rep.Entries`; the `not reached`/gate entries are appended only by `continue` at the top of the loop. `finish`'s `rep.Entries[min(c.printed, len(rep.Entries)):]` is therefore correct, and my M4 mutation (flush everything) breaks the goldens, so the guard is load-bearing and covered.
- **`→` is decided before the tee.** `runConverge:77` computes `terminal`/`paint` from the raw writer, `beginRunLog` wraps it at `:82`; `runlog.go:262` strips `\x1b…` on every transcript line, so one writer serves both, as the proposal claims.
- **The colour cannot leak into a non-tty.** `Painter.Paint` returns bare text when `On` is false, and padding is computed on the *bare* tag (`converge.go:302-305`), so alignment survives colour on — visible in `converge-apply-{plain,color}.golden`, where the detail column starts at the same byte in both.
- **The duration column cannot lie about a step that never ran.** `runOne` floors at 1 ms, so `duration_ms == 0` means "did not run" in both JSON files; `stepDuration` prints blank for `<= 0`. This directly mitigates the documented JSON-contract risk in proposal §3.

Verification claims I checked from the outside, all true: the three pre-move/pre-colour commits cited in `verification.md` (`01dcfa7a`, `6df8d647`, `de3bbeae`) exist in this repo; `from-zero.yml` reads only `.result`, `.entries[].status` and `.changed` from `last.json` (lines 112, 168), so `duration_ms` is genuinely additive; `.gitattributes`'s new `*.golden text eol=lf` plus `tests/gitattributes-eol.bats` close the Windows-CRLF hole that the byte-for-byte goldens would otherwise have opened (PR1's supporting change, correctly captured as lesson 424). No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tag remains anywhere in the spec folder (archive gate's first pre-flight is clear).

### Evidence produced in this session

Fresh runs, this reviewer, on `f18a3d62`, macOS arm64, repo `/Users/manu/Projects/dotfiles-wt-cli114`:

- `go build ./...` → exit 0; `go vet ./...` → 0; `GOOS=windows go vet ./...` → 0.
- `go test ./...` → every package `ok`, 0 failures.
- Every `features.json` verification command (f1–f6) re-run with `-count=1` → ok; f7 is the full suite above.
- `golangci-lint cache clean && golangci-lint run` (2.12.2) → `0 issues.` (1.86 s wall / 11.7 s user, i.e. a real run, not a cache hit; my first run without cleaning printed a warning about stale cache entries pointing at the sibling worktree `dotfiles-wt-mise-prune`, so the cache-cleaned number is the one to quote).
- `bats tests/gitattributes-eol.bats` → `1..4`, 4 ok, including the new `every tracked golden file resolves eol=lf`.
- **AC2 proved by reverting, not by reading:** `git checkout 9fdc76ec -- cli/internal/doctor/report.go` (the pre-move implementation, with its own ANSI constants and its own `isColorEnabled`) then `go test ./internal/doctor -run TestReport` → `ok`. The goldens were produced by the *new* code, so the old code passing them is direct evidence of byte-identical output, on both the colour and the plain path. File restored; `git status` clean.
- **Mutations, each reverted:**
  - `Start` printing unconditionally → `TestConvergeConsole_Streams/converge-stream-pipe` FAIL (goldens diff shown). Matches `verification.md`.
  - `runOne` dropping the 1 ms floor → `TestRun_RecordsADurationAtMillisecondResolution` FAIL (`a 300µs step: Duration = 0s, want 1ms`). Matches.
  - `finish` flushing every entry instead of the unseen tail → `TestConvergeConsole_Streams` FAIL with the whole table reprinted. Matches the "printed exactly once" claim.
  - Working tree after all of the above: clean apart from the untracked `specs/CLI-114-converge-console-ux/review-request.json`, which was already there when this run started.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|---|---|---|---|---|---|---|
| Major | THEORETICAL | `cmd/converge.go` + `converge_runlog.go` — step↔output ordering | On an **applied run on a terminal**, a step's subprocess output reaches the console through the `runCapture` pump goroutine (`pump(rOut, c.out, rl.Stream())`, `converge_runlog.go:99-101`), while the printer writes `→ name` and the result line **directly** to the original fd 1 via the tee. Nothing drains the pump between a step's `Start`/`Done`, so the child's bytes can land *after* that step's result line (and after the next `→`), attributing output to the wrong step; the same race reorders lines inside the transcript, which is a durable artifact. Proposal §Risks asserts the output "lands between its `→` line and its result line" — the code guarantees only that it was *written* in that order, not *observed* in it. Before CLI-114 the whole table printed after the run, so no interleaving was possible; the streaming is what makes the ordering a promise. Not reproduced (a real apply run under a pty is destructive), hence THEORETICAL, not REAL. | code read `converge_runlog.go:87-127`, `converge.go:74-101`; `runCapture.stop()` runs only in the deferred `done()`, after `saveConvergeReport` | UNTESTED for the capture path. `TestConvergeConsole_Streams` simulates the step writing to a `bytes.Buffer` (`step.Reconcile` → `s.out.WriteString`), which is synchronous and therefore cannot exhibit the race; `TestConvergeConsole_AnAppliedRunColoursTheTerminalNotTheTranscript` exercises the tee but asserts only escape-byte absence, not line order | code + tests (a `runConverge`-level test with a child that writes to stdout, asserting order, or a drain/barrier before each result line; if the race is accepted as cosmetic, that acceptance belongs in `verification.md` — the proposal is contract set and closed under this verdict) |
| Minor | REAL | `cmd/converge.go:266` `printConvergeReport` | Dead production code. `runConverge` builds `newConvergePrinter` + `RunObserved` + `finish`, so `printConvergeReport` now has **zero** production callers (`grep -rn 'printConvergeReport(' cli/internal/cmd/*.go \| grep -v _test` → only the definition); it survives as a test fixture used by 5 sites in `converge_console_test.go`. AGENTS.md's quality table says "no dead code", and a renderer that only tests call can silently diverge from the one production uses — here they already differ: it derives `width` from `rep.Entries`, the production path from `reg` | `grep` output above; `verification.md` does not mention it | covered by the tests that call it (`TestConvergeConsole_Colour`, `…_FailureSummary`, `…_ALongNameWidensTheColumn`), which is the problem: those goldens are produced by the non-production renderer | code/tests (either make the pipe path use it, or fold it into the test file as a helper built on `newConvergePrinter`) |
| Minor | THEORETICAL | transcript content vs proposal §2 | The `→ name` announcement is written to `out`, which for an apply run **is** the tee, so a terminal apply run's transcript contains announcement lines a piped run's does not. Proposal §2 promises "a pipe, a CI log and the transcript read the same text": true of the tags and the table, but the transcript's *shape* is now environment-dependent, and the runbook sentence added in PR3 ("on a terminal, a `→ <step>` line also marks the step that is running") does not say the transcript inherits it. Two people debugging from two transcripts of the same command get different line counts | `converge.go:96` (`pr := newConvergePrinter(out, …)`), `converge.go:283-287` (`Start` writes to `c.w` = the tee); `TestConvergeConsole_AnAppliedRunColoursTheTerminalNotTheTranscript` asserts absence of `0x1b` only | UNTESTED — no test asserts whether `→` is present or absent in the transcript | tests + `verification.md` (decide which one is the contract, then assert it; spec wording belongs in the next spec, not this contract set) |
| Minor | REAL | colour decision duplicated | AC1's "one colour decision" is shared for the *policy* (`console.ColorEnabled`) but not for the *probe*: doctor uses `console.IsTerminal(w)` (`doctor/report.go:79`), converge uses `cmd.convergeTerminal(w)` (`converge.go:235-238`) with the package's `isTerminal` var seam. The reason is legitimate (`console.IsTerminal` has no seam, so converge's tests could not fake a tty), but it is nowhere stated, so the next reader will "fix" one and not the other, and the two can drift on e.g. whether a closed fd counts | both call sites read above; `secrets.go:678` defines the seam | partially: `TestIsTerminal_NeverForANonFile` (console) and `TestConvergeConsole_ColourIsATerminals` / `…_ColourOnATerminal` (cmd) each pin their own side, so a drift in *one* is caught — a drift in the *pair* is not | code (give `console.IsTerminal` a seam and delete `convergeTerminal`, or document why they differ) |
| Minor | REAL | `converge.go:77` + `:241` | Redundant, mismatch-prone terminal probe: `runConverge` computes `convergeTerminal(w)` **and** `convergePainter(w)`, which recomputes `convergeTerminal(w)` internally — two `isatty` ioctls on the same fd per run. Worse, the pair takes the same writer twice in different shapes, so a future caller passing the tee to one and the raw writer to the other silently produces a coloured transcript (exactly the failure PR1's mutation list says it guards) | `converge.go:77`, `:241-243` | `TestConvergeConsole_ColourOnATerminal` pins `convergePainter` alone; the *pairing* inside `runConverge` is UNTESTED (only the end-to-end goldens would catch it, and they do) | code (`convergePainter(terminal bool, getenv)` or a single call returning both) |
| Minor | REAL | `verification.md` accuracy | `verification.md` states `from-zero.yml` "selects `.result` and `.entries[].status` with `jq`". It also selects `.changed` (`.github/workflows/from-zero.yml:168`: `jq -e '.result == "ok" and .changed == 0'`). The conclusion (a new additive field breaks no reader) is unaffected, but the stated measurement is incomplete, and this spec's own AC depends on it | workflow lines 110-118 and 164-170, read in this session | `tests/from-zero-doctor-gate.bats` covers the doctor gate, not this jq claim; the claim itself is UNTESTED (it is a prose measurement) | `verification.md` (outside the contract set — safe to edit now) |
| Question | — | `features.json` | All seven rows are `state: "pending"` with empty `evidence`, although every verification command exits 0 in this session. `contract_digest.go:51-72` confirms `state`/`evidence` are harness-owned and normalised out of the staleness digest, so this is not a gate blocker — but the archived spec will carry seven pending rows unless the harness writes them. Needs the implementer's confirmation of which command fills them | `features.json` + `cli/internal/spec/contract_digest.go` | n/a | harness/`verification.md` (not the reviewer's to write: `features.json` is contract set) |

Nothing found in the security checklist for this diff: no injection surface (the printer only formats strings converge itself produced), no secrets (the transcript's redaction path is untouched and PR1's colour decision happens *after* `runLogSecrets`, not inside it), no unbounded buffer, no new dependency (`golang.org/x/term` was already a direct dep), no authz change. `dotf converge`'s exit status is unchanged by every one of these findings — the ordering finding is presentation-only.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| Correctness | B | All 7 ACs verified by named tests re-run fresh and 4 of the implementer's mutations reproduced; one ordering guarantee (Major/THEORETICAL) is asserted by the spec but only proven in-memory. |
| Verification | A | Every command in `verification.md`/`features.json` reproduces exit 0 on `f18a3d62`; the SHA citations resolve; AC2 byte-identity was re-proved by reverting `doctor/report.go`, not by accepting the claim. |
| Scope | B | CLI-114's four commits match the proposal; `.gitattributes` + the new bats guard + the runbook line are supporting work beyond the proposal's numbered list (documented in lesson 424); the wide `base...HEAD` window is the launcher's, not the author's. |
| Reliability | B | Fails safe: colour off for a non-tty, ANSI stripped at the transcript boundary, writes' errors ignored deliberately, 1 ms floor keeps `duration_ms == 0` unambiguous; the pump-vs-printer race is the gap. |
| Maintainability | B | Functions under 40 lines, low CC, `golangci-lint` 0 issues on a clean cache, but one dead production renderer and two duplicated terminal probes. |
| Handoff-readiness | A | Lesson 424 written and indexed, runbook updated in the same change, ADR/pattern promotion lines answered with reasons, `verification.md` refreshed post-merge (`f18a3d62`) before the review. |

Aggregation: no D, no C → rubric path says PASS; the open Major is THEORETICAL, so the verdict path says PASS WITH GAPS. The more severe of the two applies.

### Verdict

**PASS WITH GAPS**

Every open Major is THEORETICAL (none REAL), there is no Blocker, and the rubric has no C or D. The gaps are tracked, not fixed, and this verdict is what makes them non-blocking — under the archive gate the contract set (`proposal.md`, `tasks.md`, `features.json`) is therefore **closed** for this round: do not edit those three to answer the findings above, or the verdict that permits the archive is invalidated. `dotf spec archive` is advisable once the implementer has recorded a disposition for each row below in `verification.md` (applied / ticketed / declined with reason).

### Recommended next steps

All of these land **outside** the contract set (`verification.md`, code, tests, or a follow-up ticket). None blocks the archive; each needs a stated disposition.

1. **Ordering race (Major/THEORETICAL)** — disposition it in `verification.md`. Either add the missing proof (a `runConverge`-level test where a step spawns a real child writing to stdout, asserting the `→`/output/result order through the capture) or record the acceptance explicitly: "interleaving is cosmetic; a step's own output may follow its result line". If it is fixed in code, the fix is a barrier before each result line, not a `\r` ticker — the ticker was scoped out for the same garbling reason (CLI-115/#2480).
2. **Delete or reparent `printConvergeReport`** (Minor/REAL) — one grep keeps this honest: after the change, `grep -rn 'printConvergeReport(' cli/internal/cmd/*.go | grep -v _test` must return nothing at all, and the three goldens it owns must still pass by going through the production renderer.
3. **Collapse the terminal probe** (Minor/REAL × 2) — give `console.IsTerminal` the same seam `cmd.isTerminal` has, drop `convergeTerminal`, and have `runConverge` derive `terminal` once and pass it to the painter. Both duplications die in one commit; the existing tests (`TestIsTerminal_NeverForANonFile`, `TestConvergeConsole_ColourOnATerminal`) already pin both ends.
4. **Decide the transcript's arrow policy** (Minor/THEORETICAL) and assert it in `TestConvergeConsole_AnAppliedRunColoursTheTerminalNotTheTranscript` — one added line either way (`Contains("→ ")` or `NotContains`). Today no test knows what the transcript should contain besides escape bytes and `converge on `.
5. **Correct the `from-zero.yml` measurement** in `verification.md` to include `.changed` (Minor/REAL, outside the contract set, safe to edit now).
6. **Confirm the `features.json` `state`/`evidence` writer** — if nothing in the harness fills them, the archived audit trail reads "seven ACs, all pending" against a suite that passes; that is a question for the implementer, not the reviewer, since `features.json` is contract set.
7. **Carry to CLI-115 (#2480)** the heartbeat decision unchanged; the ordering finding (row 1) is the same physical problem seen from the other side — a step that is slow *and* whose output arrives late is what a reader will actually experience on a pty. Worth naming there.
