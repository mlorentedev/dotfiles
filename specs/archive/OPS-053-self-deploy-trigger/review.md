---
spec: "OPS-053-self-deploy-trigger"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "12818876a31f4b4fe0287519f89b0f4897d6178e"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-10-17"
---

## Adversarial review

**Scope**: OPS-053-self-deploy-trigger, round 2 — `git diff 2888f9cc3bbadd012238baf8394819c64668d8e0...HEAD` (launcher-resolved base, the whole change including round 1's scope), reviewed at `12818876`.
**Sources**: `specs/OPS-053-self-deploy-trigger/{proposal,tasks,verification,features}.md/json`, `review-round1.md`, `review-request.json`; the spec's commits inside the range are `bc0185d0` (#2682, darwin), `ba750094` (#2693, linux/windows + twin deletion), `adb56ad0` (#2695, record + doctor), `12818876` (#2717, round-1 fix round), with `dc1e7d9b` (#2721, closes #2688) adjacent.

### Spec and task alignment

- The range is wider than the spec (landed work by other efforts: sshtarget, forge, casks, spec-gate, release 0.74.0, tool pins). That is the base the launcher stated, not creep; findings are scoped to OPS-053's commits plus `dc1e7d9b`, which verification.md itself leans on as the F2 blocker.
- **Gates re-run fresh in this session, at `12818876`**: `go build ./...` exit 0; `go vet ./...` exit 0; `GOOS=windows go vet ./...` exit 0; `go test ./...` exit 0, **0 `--- FAIL` lines**; `golangci-lint run ./...` (pinned 2.12.2) 0 issues, exit 0. AC6's own gate `git grep -n 'DOTFILES_AUTODEPLOY\|DotfilesSelfUpdate\|dotfiles-selfupdate' -- setup-linux.sh setup-windows.ps1` returns exit 1 (empty), verified myself. Every `verification` command in `features.json` (f1–f7) was executed verbatim: all exit 0, no FAIL lines.
- **Round-1 findings re-verified by mutation, red-first, then reverted** (worktree left clean apart from this review):
  - F1: forcing `unreachable` to always return the error (pre-fix behavior) turned `TestSelfDeploy_WithoutTheRoleNeverFailsOnAnUnreachableScheduler` red on both sub-cases, plan and apply, and the probe. The fix reads the role (`env.LoadMachineRoles`) before the scheduler is built or inspected; without the role an unreachable scheduler is a skip naming the error with no backend write, with the role it stays a failure — matching the proposal's restated AC4 and the `workstationTarget`/doctor ordering in this same diff.
  - F3: deleting the `percent.Replace` on `ExecStart` argv turned `TestSystemdUnits_DoublesPercentEverywhereTheJobReaches` red; the test covers argv (program and arg), the service `Description`, and the timer `Description`, and `Environment=` carries `%` in its quote replacer. The comment above the replacer is now true of the code.
  - F5: appending `DOTFILES_AUTODEPLOY=1` to a copy of `setup-linux.sh` turned `tests/guard-setup-no-self-deploy.bats` case 1 red (reproduced myself); case 4 pins the guard's pattern to `update.TriggerNames` so a trigger rename cannot silently orphan the guard.
- F4 restated in both contract files (proposal AC4 and the marker rule now say the unmarked case binds on linux/windows; darwin `Managed` is constant); features.json f4 matches. F6 is ticketed: `gh issue view 2715` → OPEN, "Two registrars for one kind of Windows Scheduled Task". F7a's code fix (`selfDeployWhen` reads `job.Daily`) is in place. F9's closing boxes are ticked except the independent-review box, which correctly stays open until this round.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in any spec file (grep exits 1; transcript files excluded).
- AC5 stands on round 1's independent byte-comparison (golden vs `2888f9cc:cli/internal/env/launchd.go`): `env/launchd.go` and the golden are unchanged since round 1 (`git diff 9f596c94..HEAD` on `schedule/launchd.go`, `doctor/checks_self_deploy.go`, `internal/update` is empty), and `TestLaunchAgentPlist_BytesArePinned` passes.
- Cleared on inspection this round: the schtasks principal is deliberately excluded from the digest, but `Inspect` still flags an upgrade (standard registration → S4U when the caller is elevated) and never a downgrade, so elevation change cannot strand or flap the task (`schtasks.go` `read`/`Inspect`). `xmlEscape` covers `&<>`, which suffices for the text-node placements used. `Trigger` carries `XDG_CONFIG_HOME`/`XDG_STATE_HOME` so the job reads the same machine.json and writes the record doctor reads (`TestTrigger_CarriesTheXDGHomesOfTheInstallingRun`). AC7's verdicts are table-tested (`TestCheckSelfDeploy_ByState`, `TestRecord_*`, `TestRecordUpdate_KeepsTheFailureAndNeverFailsTheRun`, `TestUpdate_ConvergesAfterAFastForward`). Host state is consistent with the recorded live evidence: `launchctl print gui/501/...` exits 113 and this machine's `machine.json` declares no roles.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | verification (AC2/AC3) | Linux and Windows `changed=0` is still proved by fake runners only. Round 1's blocked `dotf vm linux` run stayed blocked, and **the blocker was closed inside this very range** (`dc1e7d9b`, "Closes #2688", landed 14:13) without the run being repeated — and a repeat still cannot certify this tree today, because the guest converges the pinned release 0.74.0, which predates the PATH fix. AC2/AC3 as worded accept fake-runner unit tests, so this is an evidence gap, not a spec violation. | `verification.md` F2 disposition ("blocked by #2688") vs `git log` showing #2688 closed in-range; the guest transcript in `verification.md` shows it dies at `records-harness` before `self-deploy`; my fresh runs show all fakes green. | Fake path covered by `TestSystemd_InstallThenCurrentThenRemove` / `TestSchtasks_InstallThenCurrentThenRemove`; **live path UNVERIFIED** | tests (`dotf vm linux` after the next release) + `verification.md` disposition refresh |
| Minor | THEORETICAL | round-1 fix F7a | The `selfDeployWhen` fix (OK detail reads the job's slot) has no named test: nothing asserts the detail reflects `job.Daily`, so the hardcoded-"03:00" drift F7a found could return silently. | `grep -rn "daily at" cli/ --include=*_test.go` exits 1; `TestSelfDeploy_OptedInInstallsThenIsConverged` checks only the `applied:` prefix, not the OK detail. | UNTESTED | tests |
| Minor | REAL | `verification.md` (stale text) | The F2 disposition row still says "blocked by #2688" while this range's own commit `dc1e7d9b` closes #2688; the practical block is now the release pin (guest runs 0.74.0), which the row states but does not connect to the closure. The sentence is stale, not false in effect. | `git log -1 --format=%s dc1e7d9b` → "Closes #2688", vs the row's wording. | n/a (documentation) | `verification.md` (outside the contract set) |
| Minor | THEORETICAL | scope / attribution | `dc1e7d9b` (#2721: converge prepends the running dotf's directory to its children's PATH unless PATH already resolves dotf to the same file) sits inside the launcher's range but is not an item in the proposal's "What". It closes #2688, carries its own tests and a lesson, and verification.md references it — adjacent work, not spec divergence. The behavior change is cross-cutting (every converge run's children get the running binary's dir first), so it deserves to be named rather than absorbed silently. | diff of `dc1e7d9b`; proposal has no PATH item. | `TestEnsureSelfOnPath`, `TestRunConverge_PutsTheRunningDotfFirstOnPath` | none — attribution note; the lesson documents it |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | All seven ACs met with named tests at the reviewed sha; the one demonstrated defect from round 1 (F1 hard-failing non-opted-in hosts) is fixed and I reproduced its red-first test; residual: AC2/AC3 unexercised on live hosts (counted under Verification). |
| Verification | B | Every features.json command re-run green; F1/F3/F5 mutations all turned named tests red and were reverted; AC5 independent byte check stands (renderer unchanged since round 1); deductions: no live Linux/Windows evidence, F7a's detail fix untested. |
| Scope | B | The spec's four commits touch only the declared surfaces plus registry wiring; the wide base is the launcher's, and `dc1e7d9b` is issue-linked, tested and lessoned. |
| Reliability | B | Role-before-scheduler ordering now matches doctor and the `workstationTarget` prior art; backends fail closed (113-only launchd absence, bus errors never read as absence, managed/digest markers, atomic writes). |
| Maintainability | B | Small functions, one job declaration, injected runners, no dead code; deductions for the untested `selfDeployWhen` and the grep-based bats guard's brittleness (mitigated by its TriggerNames pin). |
| Handoff-readiness | B | ADR-019 amended, lesson 468 written, runbook rewritten around the role, round-1 dispositions table with tickets #2715/#2696 verified; deduction: the F2 row is stale against #2688's closure in this range. |

Aggregation: all dimensions B or above → PASS path; the findings axis has one open **THEORETICAL Major** → PASS WITH GAPS is the floor; the more severe path wins.

### Verdict

**PASS WITH GAPS**

Round 1's FAIL rested on one REAL Major — the step read the scheduler before the opt-in. That is fixed, the regression test is named and I made it red myself by restoring the old ordering; the `%`-escaping and guard-bats minors are likewise fixed and mutation-verified; the contract restatements (AC4/f4) landed with the code. I could not break the marker rule, the digest rule, the 113-only absence rule, or the age threshold, and all seven verification commands pass verbatim. What remains open is one THEORETICAL Major (no live Linux/Windows proof, now gated on a release rather than on #2688) and three Minors, each tracked below — enough for GAPS, not enough to block.

### Recommended next steps

**Routed to `verification.md` dispositions or follow-up tickets — the contract set (`proposal.md`, `tasks.md`, `features.json`) is closed under this verdict; do not edit it:**

1. After the next release ships `dc1e7d9b`, run `dotf vm linux` and paste the guest's two converge reports into `verification.md`; reword the F2 row to say #2688 is closed and the remaining block is the release pin. (Major, THEORETICAL; Windows live still needs the work-lab box — keep that boundary stated.)
2. Add one assertion that the OK detail reflects `job.Daily` (e.g. a job at a non-03:00 slot), closing the UNTESTED gap on F7a's fix. (Minor, tests.)
3. The implementer may tick `tasks.md`'s independent-review Closing box for round 2 — checkbox ticks are ignored by the staleness check, so this does not invalidate the review.

**`dotf spec archive` is advisable in the current state**: the verdict is passing, `reviewed_sha` is the current HEAD, and the contract set is untouched by this review.
