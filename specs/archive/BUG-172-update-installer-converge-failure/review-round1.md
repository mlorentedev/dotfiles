---
spec: "BUG-172-update-installer-converge-failure"
verdict: "FAIL"
reviewed_sha: "b911c61a66ba2c99789c53e0e8a5f3ea6940f1b9"
reviewer: "nan/qwen3.8-flash"
date: "2026-11-12"
---

## Adversarial review

**Scope**: `BUG-172-update-installer-converge-failure` (issue mlorentedev/dotfiles#3395)
**Sources**: `specs/BUG-172-update-installer-converge-failure/{proposal,tasks,verification,features}.md/json`;
diff `git diff 437e4556474e4c539c1a2191a8fdf0ed6b72c37f...HEAD` (= one commit, `b911c61a`, confirmed with
`git log --first-parent --no-merges <base>..HEAD`: no other work merged into the range); `install.sh`,
`install.ps1`, `cli/internal/converge/{report,converge}.go`, `cli/internal/cmd/{update,converge}.go`,
`cli/internal/update/{update,record,pin_test,update_test}.go`, `docs/runbooks/guide-self-deploy-timer.md`.
Contract digests in `review-request.json` match `dotf spec digest BUG-172-update-installer-converge-failure`
(features `b59496a2…`, proposal `86976baf…`, tasks `4e033e63…`) — this review is about the current contract.

### Spec and task alignment

- **Premise verified at the installer, not just asserted.** `install.sh` ends with
  `install_dotf || exit 1` then `exec "$_dotf_bin" converge "$@"` (install.sh:292-293); a failed
  download/verify exits before any converge. `install.ps1` mirrors it: `throw 'install.ps1: dotf could not be
  installed, so nothing was converged'` before `& $script:DotfBin converge @args` (install.ps1:256-258).
  So "a failed report newer than the installer's start is the installer's own converge" holds on both
  platforms the pin path runs on. This is the load-bearing assumption of the whole spec and it is true.
- **Report provenance verified.** `last.json` is written only by `saveConvergeReport(..., full)` with
  `full = len(only) == 0` (converge.go:189-212), so `--only` runs (the quiet-day `system-upgrade,cache-gc`
  path) and `--plan` runs cannot leave a fresh report. `WriteReport` stamps
  `time.Now().UTC().Format(time.RFC3339)` and sets `result:"failed"` exactly when the run errored, with
  `error` non-empty — so the AC3 `"the run ()"` degenerate case needs a hand-written report. The installer's
  converge inherits `os.Environ()` from `runInstaller` (`cmd.Env = append(os.Environ(), …)`), including the
  `XDG_STATE_HOME` that `--state-home`/`updateHomes.apply()` set, so reader and writer resolve the same path
  through `env.StateDir()`.
- **Status vocabulary is not new.** `converge-failed` already flows into the update record
  (`recordUpdate` → `update.NewRecord`, `Record.Failed()` keys on `Error != ""`), and
  `doctor/checks_self_deploy_test.go:61` already asserts a `converge-failed` record FAILs with its error.
  No status whitelist anywhere enumerates update statuses (`grep -rn "install-failed\|converge-failed"` over
  the repo hits only Go tests + `update.go`), so the re-tag cannot strand a consumer.
- **Every `[x]` in `tasks.md` maps to diff evidence.** The two named tests exist
  (`pin_test.go:TestRunPin_ConvergeFailsInsideInstaller`, `update_installer_test.go:TestFailedConvergeSince`),
  the seam and the reader exist, and the tooling box is true — re-run below. The one unticked box is the
  closing one (verification/review/archive), correct for this window.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in the spec folder.
- Nit (finding F6): the proposal's own AC boxes are still `[ ]` while `tasks.md` ticks everything; the
  template's two check-boxes disagree about the same three criteria.

### Commands run in this review (fresh)

| Claim under review | Command | Result |
|---|---|---|
| tests pass | `go test ./internal/update/ ./internal/cmd/ -timeout 8m` | `ok update 0.131s`, `ok cmd 44.060s`, exit 0 |
| vet clean | `go vet ./...`; `GOOS=windows go vet ./...` | exit 0, exit 0 |
| lint clean | `golangci-lint run ./internal/cmd/... ./internal/update/...` (+ `GOOS=windows`, after `cache clean`) | `0 issues.` both |
| formatting | `gofmt -l` on the four touched Go files | no output |
| AC1/AC2/AC3 mutation claims in `verification.md` | `.Truncate(time.Second)` removed → `--- FAIL: TestFailedConvergeSince/finished_in_the_installer's_start_second`; `len(failed)==0` fallback removed → `--- FAIL: …/a_failed_run_with_no_failed_step_names_the_run`; `installerConvergeFailed(d)` stubbed to nil → `--- FAIL: TestRunPin_ConvergeFailsInsideInstaller/{installer's converge failed a step,two steps failed}` | every claim reproduced; the suite is not vacuous on the units |
| report-input robustness | probe in a `/tmp` copy of the module: naive timestamp (no offset), non-UTC offset landing before the start, missing `finished_at`, `status:"opaque"` entry | `[]`, `[]`, `[]`, `["the run (x)"]` — fails closed, and the offset case is compared as an instant, not a wall clock |
| complexity ceiling | `min-complexity: 15` → `9` in the copy, `golangci-lint run ./internal/cmd/...` | neither new function is reported (only the pre-existing `TestUpdate_ConvergesAfterAFastForward` at 15), so `failedConvergeSince` and the `Run` branch are ≤ 9, inside the repo's stated < 10 |

All mutation work ran in a `rsync` copy under `/tmp/rev172` (removed afterwards); the reviewed worktree holds
only `specs/BUG-172-…/review.md` (`git status --porcelain` at the end of the run).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| **Major** | **REAL** | wiring / tests | The safety of the whole fix rests on one unguarded line in a package no test exercises: `installStart = time.Now()` inside the `Install` closure (`cmd/update.go:79-82`). `failedConvergeSince` has **no guard for the zero time** — `since.Truncate(time.Second)` is year 1, so *any* failed `last.json` ever written is accepted as "the installer's own converge". Mutation proof: deleting the assignment leaves `installStart` read-but-never-written (legal Go) and **the full `internal/update` + `internal/cmd` suites stay green** (`ok update (cached)`, `ok cmd 43.754s`). Probe proof: with `since = time.Time{}` a report stamped `2020-01-01T00:00:00Z` returned `["system-upgrade"]`; with a 2026 start the same report returned `[]`. The resulting user-visible lie is the *inverse* of #3395: a download that never landed is reported as `converge-failed` / "dotf X -> Y (the DOTF_VERSION pin) **installed**; its converge failed at …" — sending the operator to the wrong step again, now with a false positive about the binary. Same class: moving the assignment after `runInstaller` also survives (silently restoring the original bug). The premise is only enforced by call order in a *different package* from the reader. | surviving mutant (M1) + reproduced zero-time misread, this session | **UNTESTED** — no test drives `newUpdateCmd` through the pin path: `grep -rn "newUpdateCmd\|runInstaller" --include=*_test.go` hits only `installerInvocation` (`update_pin_test.go:153`) and the direct `failedConvergeSince(start)` call (`update_installer_test.go:46`); `TestUpdate_ConvergesAfterAFastForward` pins `DOTF_VERSION=1` but the running version is a source build, so `pinToInstall` returns `""` and the branch is never entered | code (`if since.IsZero() { return nil }` in `failedConvergeSince`) + tests (one table row: `{name: "no installer start recorded", start: time.Time{}, want: nil}` in `TestFailedConvergeSince`; and/or a command-level test of the pin path with a fake installer seam) |
| Minor | THEORETICAL | quality | `failedConvergeSince` matches the entry status against the literal `"failed"` while ~60 lines above the same file compares the typed constant (`e.Status == converge.StatusSkipped` in `systemUpgradeSkip`). `RunEntry.Status` is the persisted string of `converge.Status`, whose vocabulary is defined by a positional array literal (`[...]string{"ok","change","skipped","failed","opaque"}`), so a rename/drop shifts the wording of every report to `the run (…)`, or to nil, without failing anything. `converge.StatusFailed.String()` is exported and free. | `converge.go:77-79`, `cmd/update.go:224-228` vs `cmd/update.go:120-126` | UNTESTED — `TestFailedConvergeSince` hardcodes its own JSON strings, so a vocabulary change cannot turn it red | code |
| Minor | SPECULATIVE | perf/behaviour (accepted) | The same-second truncation accepts a failed report that finished in the installer's *start* second, which a foreign converge could also have produced. The direction is the right one (a converge can legitimately fail in <1s and must not be discarded), and the proposal's Risks section names this window and resolves it, so this is a recorded trade-off, not an oversight — the residual case it does not cover is a *human* running `dotf converge` while the timer's installer runs, which the risk note calls the only other converge on the machine. | proposal.md "Risks / open questions"; `update_installer_test.go` same-second case | `TestFailedConvergeSince/finished_in_the_installer's_start_second` | spec (already covered; no action needed beyond acknowledging the manual-converge variant) |
| Minor | REAL | handoff | All three `features.json` entries still carry `state: "pending"` and `evidence: ""` although `verification.md` records the named tests for each. These fields are normalised out of the staleness digest (so they do not invalidate a verdict), but a reader of the feature list sees three unverified behaviours. | `features.json` vs `verification.md` "Evidence" | n/a | tests/spec-adjacent file (`features.json` state/evidence — outside the contract set for staleness purposes, but it *is* listed as a contract file in the review request, so update it after the re-review, not alongside a passing verdict) |
| Minor | REAL | docs | `docs/runbooks/guide-self-deploy-timer.md` describes the installer branch ("When the installed `dotf` is not the pin, it runs the installer instead, which installs the pin and converges", :162-163) and lists the record statuses an operator can meet, but no doc anywhere distinguishes the two installer outcomes this change produces. The operator-facing consequence of the fix (which line to believe in `<state>/update/last.json`) is undocumented outside the spec folder. | runbook :25-45, :157-163; `grep -rn "install-failed"` repo-wide hits only Go | n/a | code-adjacent docs (`docs/runbooks/`, and/or `docs/troubleshooting/` per the repo's knowledge-placement rule) |
| Question | — | spec artifacts | `proposal.md` AC1-AC3 boxes are unticked while `tasks.md` ticks every implementation task including AC1-AC3. Which is the intended record of acceptance? Tick-at-archive, or a missed edit? (Contract set — do not edit under a passing verdict; say so here.) | proposal.md vs tasks.md | n/a | spec (answer in `verification.md` "Review dispositions") |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| Correctness | B | All three ACs hold in the shipped code and I could not produce a wrong report through any reachable path; the gap is the unguarded precondition, not a present defect. |
| Verification | B | Every evidence line in `verification.md` reproduced, including its mutation claim (M2/M3 killed) — but no evidence touches the command-level wiring (F1). |
| Scope | A | Diff is 2 code files, 2 test files, the spec folder: 303 insertions, no creep, and both stated out-of-scope items honoured (no installer change, no new tag). |
| Reliability | B | Fails closed on missing, unreadable, malformed, passing, stale, unparseable-timestamp and foreign-offset reports (probed); the zero time is the one input that fails open. |
| Maintainability | B | `gofmt` clean, `golangci-lint` 0 issues natively and `GOOS=windows`, both new functions ≤ 9 by measured gocyclo, comments say WHY; loses A on the string-literal status comparison (F2) inconsistent with its own file. |
| Handoff-readiness | B | Spec triad filled with named tests and decisions, promotion candidates explicitly answered; loses A on `features.json` still pending (F4) and the runbook split (F5). |

Aggregation: no D, no C → the rubric alone would read PASS. The severity path escalates independently
(F1 is a REAL, UNTESTED Major), and the more severe of the two wins.

### Verdict

**FAIL** — one REAL Major (F1): the one line that makes the "the pin is installed" claim true is
neither guarded against the zero time nor covered by a named test, and a mutation that deletes it leaves the
whole suite green while the tool tells the operator the pin installed when nothing landed. Per the
test-traceability gate, an UNTESTED Major is not resolved by the implementer's claim.

`dotf spec archive` is **not advisable** in this state.

### Recommended next steps

Minimum set to flip this to PASS (all of it outside the contract set — `proposal.md`, `tasks.md` and
`features.json` need no edit, so the digests hold and the re-review is one round of tests):

1. **F1, code**: `failedConvergeSince` — return nil when `since.IsZero()`, with a one-line comment naming the
   invariant it defends (a start that was never recorded proves nothing about the installer).
2. **F1, tests**: table row in `TestFailedConvergeSince` — zero start + a failed report → `nil`. That row
   kills the M1 mutant; a command-level test of the pin path (a `runInstaller` seam) would kill the
   "assignment moved after the call" variant too, and is the stronger of the two if time allows.
3. Re-run `dotf spec review BUG-172-update-installer-converge-failure`; the reviewer should need only
   `go test ./internal/update/ ./internal/cmd/` plus the two mutants.
4. **F2** (code, cheap, no re-review dependency): use `converge.StatusFailed.String()`.
5. **F4/F5**: set `features.json` state/evidence for f1-f3 to the named tests, and add the
   `install-failed` vs `converge-failed` distinction to `docs/runbooks/guide-self-deploy-timer.md`
   (or a `docs/troubleshooting/` entry) — #3395's whole cost was an operator sent to the wrong step, and the
   runbook is where the next one looks.
6. Answer the Question row (proposal AC boxes) in `verification.md` → "Review dispositions".

Strengths worth keeping, because they are what the gap is bounded by: the premise was checked at the
installer rather than assumed, the chosen tag required no consumer change, and every fallback in the reader
is the conservative one (`install-failed`), which is why F1 is the only path that can produce a false claim.
