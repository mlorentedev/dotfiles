---
spec: "BUG-172-update-installer-converge-failure"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "2c82e9188b56b7e84fc101366c14ba90fa881d64"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-11-12"
---

## Adversarial review

**Scope**: `BUG-172-update-installer-converge-failure` (issue mlorentedev/dotfiles#3395, state OPEN, verified this session)
**Sources**: `specs/BUG-172-update-installer-converge-failure/{proposal,tasks,verification,features}.json/md`,
round-1 review (`review-round1.md`, FAIL by nan/qwen3.8-flash at `b911c61a`); diff
`git diff 437e4556474e4c539c1a2191a8fdf0ed6b72c37f...HEAD` (whole change, both commits — round 2 judges the late fix
against the early one it builds on). `git log --first-parent --no-merges <base>..HEAD` lists exactly the spec's two
commits (`b911c61a`, `2c82e918`) — no other work merged into the range. Files: `cli/internal/cmd/update.go`,
`cli/internal/cmd/update_installer_test.go` (new), `cli/internal/update/update.go`,
`cli/internal/update/pin_test.go`, `docs/runbooks/guide-self-deploy-timer.md`, spec folder.
Contract digests in `review-request.json` match `dotf spec digest BUG-172-update-installer-converge-failure`
exactly (features `b59496a2…`, proposal `86976baf…`, tasks `4e033e63…`) — this review is about the current contract.

### Spec and task alignment

- **Load-bearing premise holds at HEAD.** `install.sh:292-293`: `install_dotf || exit 1` then
  `exec "$_dotf_bin" converge "$@"` — the installer converges only after the binary is placed, and a failed
  download/verify exits before any converge. So a fresh failed report really does prove the pin installed, and its
  absence really does mean the install itself failed (AC1/AC2's split). `install.ps1` mirrors it (round 1 checked
  both; neither file changed in this range).
- **Report provenance holds.** The writer resolves `<state>/converge/last.json` through the same `env.StateDir()`
  as the reader (`cmd/converge.go:205` vs `cmd/update.go:213`), and only a full run updates it — `cmd/converge.go:75`
  ("a plan or an `--only` run leaves [last.json] alone"). The installer inherits `os.Environ()` including
  `XDG_STATE_HOME` (`runInstaller`), so writer and reader meet on one path.
- **Round-1 F1 (REAL Major → FAIL) is fully addressed, and I re-derived it rather than reading the disposition.**
  The start instant is now taken inside `update.Run` just before `Install` (`update/update.go:109`, call at `:111`);
  `failedConvergeSince` returns nil for a zero instant (`cmd/update.go:206-208`) with table row "no installer start
  recorded proves nothing"; `TestRunPin_ConvergeFailsInsideInstaller` asserts the instant is non-zero and no later
  than the install (`pin_test.go:202-204`). Mutants (in an rsync copy under `mktemp -d`, removed at run end):
  zero-guard removed → named row fails; `var start time.Time` → 3 subtests fail; start moved after `Install`
  → 2 subtests fail; `.Truncate(time.Second)` removed → same-second row fails; `len(failed)==0` fallback removed
  → AC3 row fails. Every mutant round 1 and `verification.md` claim is killed — reproduced this session.
- **Round-1 F2 applied** (`converge.StatusFailed.String()` at `cmd/update.go:227`); `rep.Result != "failed"` at
  `:218` is a report-result literal, matching `converge/report.go`'s own vocabulary (no typed constant exists for
  `RunReport.Result`, and `ParseRunReport` validates it) — consistent, not a leftover.
- **F3 declined with the trade-off in the proposal's Risks** (accepted window); **F5 applied** (runbook now tells
  the two installer outcomes apart); **F6 answered** in `verification.md` dispositions; **F4 declined** with the
  claim that `state`/`evidence` are harness-owned — the code agrees (`contract_digest.go:50-53`: "the fields the
  harness writes after a review… progress, not contract, blanked before digesting"), see the Question row.
- Every `[x]` in `tasks.md` maps to diff evidence; the closing box stays unticked correctly for this window.
  Out of scope honoured: no `install.sh`/`install.ps1` change, no new status tag (`converge-failed` is pre-existing
  and `doctor/checks_self_deploy_test.go:61` already FAILs it), no console parsing. Status is recorded through
  `recordUpdate(c.ErrOrStderr(), out, err)` (`cmd/update.go:87`), so doctor reads the new error text too.
- No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` markers in any scanned spec file (the archive's tag scan skips
  `review.md`/`review-round*.md` by `IsReviewState`, so round 1's prose mention is not a hit).
- The one accepted residual: a *foreign* full converge (e.g. a human running `dotf converge`) finishing inside the
  installer window would be attributed to it. The proposal declares and resolves this ("the only other converge on
  the machine is this same scheduled update"); noted, not re-opened.

### Commands run in this review (fresh)

| Claim under review | Command | Result |
|---|---|---|
| tests pass (whole change) | `go test ./internal/update/ ./internal/cmd/ -timeout 8m` | `ok update 0.155s`, `ok cmd 60.739s`, exit 0 |
| features.json f1–f3 verifications | `go test ./internal/update/ -run 'TestRunPin_ConvergeFailsInsideInstaller' && go test ./internal/cmd/ -run 'TestFailedConvergeSince'` | both `ok`, exit 0 |
| vet clean | `go vet ./...` ; `GOOS=windows go vet ./...` | exit 0, exit 0 |
| lint clean | `golangci-lint run ./internal/cmd/... ./internal/update/...` (native, then `GOOS=windows`) | `0 issues.` both |
| formatting | `gofmt -l` on the four touched Go files | no output |
| contract fresh | `dotf spec digest BUG-172-…` vs `review-request.json` | identical |
| mutation: zero guard removed | `TestFailedConvergeSince` | `--- FAIL …/no_installer_start_recorded_proves_nothing` |
| mutation: `var start time.Time` | `TestRunPin_ConvergeFailsInsideInstaller` | `--- FAIL` 3 subtests (since = year 1) |
| mutation: start after `Install` returns | same test | `--- FAIL` 2 subtests (since after installedAt) |
| mutation: `.Truncate(time.Second)` removed | `TestFailedConvergeSince` | `--- FAIL …/finished_in_the_installer's_start_second` |
| mutation: `len(failed)==0` fallback removed | `TestFailedConvergeSince` | `--- FAIL …/a_failed_run_with_no_failed_step_names_the_run` |
| **mutation: binding line deleted** (`InstallerConverge: failedConvergeSince,` removed from `cmd/update.go:81`) | `go test ./internal/cmd/ ./internal/update/ -timeout 8m` | **exit 0 — suite stays green** (the survivor behind F1 below) |
| complexity at round 1's threshold | copy with `min-complexity: 15 → 9`, `golangci-lint run ./internal/cmd/... ./internal/update/...` | neither new function reported (`failedConvergeSince`, `installerConvergeFailed`, `Run` all ≤ 9); 50 hits are pre-existing |
| writer/reader path agreement | `grep last.json` over `cli/internal` | `cmd/converge.go:205` vs `cmd/update.go:213`, same `env.StateDir()` join |

All mutation work ran in an rsync copy under `mktemp -d` (removed at the end); the reviewed worktree holds only
`specs/BUG-172-…/review.md` (and the launcher's own `review-request.json` update) per `git status`.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| **Major** | **THEORETICAL** | tests / traceability | Nothing covers the **effect** of the command-layer binding `InstallerConverge: failedConvergeSince` (`cmd/update.go:81`). Deleting that one line leaves the **entire** `internal/cmd` + `internal/update` suite green (50.4s run, exit 0, this session): the reader is tested directly, the outcome is tested against a fake seam, and the one command-level test that executes the line (`TestUpdate_FailsWhenThePinCannotBeInstalled`) writes no converge report, so it cannot tell wired from unwired. A future regression that loses the binding silently restores #3395's misattribution while every test stays green. THEORETICAL because the shipped wiring is correct and the failure direction is conservative (`install-failed`, never a false "installed" claim) — the wrong behavior needs an edit nobody has made; the *coverage gap* itself is demonstrated by the surviving mutant above. Round 1 offered this same test as an optional strengthening ("if time allows"); the test-traceability gate makes it the one open Major: an UNTESTED Major needs a named regression test. | mutation M-A (this session, full suite green after deleting `cmd/update.go:81`) | **UNTESTED** — no test distinguishes wired from unwired at the command level (that absence *is* the finding) | tests: extend `TestUpdate_FailsWhenThePinCannotBeInstalled` (or a sibling in `update_pin_test.go`) with `FAKE_INSTALL_RC=7` + a fresh failed `<XDG_STATE_HOME>/dotfiles/converge/last.json` → assert `converge-failed` in the command output; one test kills the mutant and exercises binding + start-order + reader together |
| Minor | REAL | quality | `update.Run` is 49 lines against the repo's <40-line rule. Pre-existing debt — it was 43 at base — but this diff grows it by 6 while extracting `installerConvergeFailed` (the right instinct). gocyclo at threshold 9 does not report it (measured), so only the length rule is breached. | `update/update.go:84-132`; base `437e455` = 43 lines | n/a (static; measured line count + gocyclo ≤ 9) | code — optional follow-up: fold the pin branch into a helper alongside `installerConvergeFailed` |
| Minor | REAL | evidence accuracy | `verification.md` claims the two start mutants fail "4 subtest failures each". Measured: `var start time.Time` → 3 subtest failures (+ parent = 4 FAIL lines); start-after-`Install` → **2** subtest failures. Both mutants do fail the suite, so the substance of the claim ("the test catches it") holds; the count does not reproduce. | this session's mutation runs vs `verification.md` "Test status" | `TestRunPin_ConvergeFailsInsideInstaller` (the counting site) | verification.md — outside the contract set; correct the count or drop it |
| Question | — | spec artifacts | `proposal.md` AC1–AC3 boxes remain `[ ]` while `tasks.md` ticks the same criteria. Answered in `verification.md` → "Review dispositions" (tasks + verification are the record of acceptance; contract files are not edited after review). Recorded, not re-opened — and this verdict assumes that answer. | proposal.md vs tasks.md | n/a | spec (answer already recorded; no edit under this verdict) |
| Question | — | features.json | `state`/`evidence` still `pending`/empty. `contract_digest.go:50-53` says these are harness-owned and blanked from the digest, and no writer for them exists anywhere in `cli/` (grep, this session) — **who sets `passing`, and when, is UNVERIFIED**. Implementer's decline of round-1 F4 is consistent with the code comment; the risk is that nothing ever sets them and the features stay visually unverified forever. | `features.json` vs `contract_digest.go:52` | n/a | spec-adjacent (harness; do not edit under this verdict — it is a contract file) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| Correctness | B | AC1–AC3 hold in the shipped code (both halves unit-tested, every behavior mutant I could build killed); the gap is the untested conjunction, not a present defect. |
| Verification | B | Every `verification.md` claim re-run and reproduced (tests, vet, lint ×2 GOOS, 5 of 5 killable mutants, digests) — but no evidence covers the command binding (F1) and one failure count doesn't reproduce (F3). |
| Scope | A | 11 files, all named by the spec: 2 code, 2 tests, 1 runbook, spec folder; both out-of-scope items honoured; no foreign commits in the range. |
| Reliability | B | Fails closed on missing, unreadable, stale, passing, zero-start and malformed reports (8 named rows + zero guard); loses A only on the accepted cross-process attribution window and the untested binding. |
| Maintainability | B | gofmt clean, lint 0 issues native and `GOOS=windows`, new functions ≤ 9 gocyclo at a 9 threshold; loses A on `Run` at 49 lines (>40). |
| Handoff-readiness | B | Triad + runbook + all six round-1 dispositions recorded; loses A on `features.json` still pending with an UNVERIFIED writer. |

Aggregation: no D, no C → the rubric path reads PASS. The severity path is taken instead: the one open Major is
THEORETICAL (its failure needs a future edit; the shipped diff has no wrong behavior I could produce), so the
verdict is **PASS WITH GAPS**, each gap tracked above.

### Verdict

**PASS WITH GAPS** — no Blocker, no REAL Major, rubric all B or above. Round 1's REAL Major (F1) is fixed in a
stronger form than it asked for, and I reproduced every mutant kill it claimed. The one open Major is a test
traceability gap (command-level binding UNTESTED, surviving mutant demonstrated) tagged THEORETICAL because the
shipped code is correct and its failure direction is the conservative one.

`dotf spec archive` is **advisable** once the implementer records F1's disposition (applied / ticketed / declined
with a reason) in `verification.md` — the file is outside the contract set, so that edit does not stale this
verdict, and the contract set (`proposal.md`, `tasks.md`, `features.json`) needs no change.

### Recommended next steps

For `verification.md` dispositions or a follow-up ticket — **not** contract edits; the contract set stays closed
under this verdict:

1. **F1 (tests, apply now — does not stale the review):** the one command-level test above. It kills the binding
   mutant and closes the last UNTESTED link between the spec's AC1 and `dotf update` as an operator runs it.
2. **F2 (code, optional follow-up):** split `Run`'s pin branch into a helper to get back under the 40-line rule.
3. **F3 (verification.md, cosmetic):** correct the "4 subtest failures each" count (3 and 2 measured).
4. **F4/F5 (Questions, no action under this verdict):** proposal AC boxes — recorded answer stands;
   `features.json` — harness's move after archive; confirm something actually sets `passing`, since no writer was
   found in `cli/`.
5. Second opinion: the harness offered no stronger-model review mechanism inside this run — cross-vendor
   independence here comes from the pool draw (round 1 was nan/qwen3.8-flash, this one nan/mimo-v2.6-flash); no
   differing second opinion exists to report.
