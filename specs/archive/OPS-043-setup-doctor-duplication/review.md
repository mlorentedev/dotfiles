---
spec: "OPS-043-setup-doctor-duplication"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "b55a5493ef1a7f5f263d7c05ddc8b776c2f6cf07"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-02"
---

## Adversarial review

**Scope**: OPS-043-setup-doctor-duplication
**Sources**: `specs/OPS-043-setup-doctor-duplication/{proposal,tasks,verification}.md`, `features.json`, git diff `745f320..HEAD` (merge-base→HEAD, 2 commits: `416233e` + `b55a549`).

### Spec and task alignment

- All six acceptance criteria (AC1–AC6) are functionally met and each has a named test: `TestCheckHomeDeployDrift` (AC1/AC2), `TestHomeDeployExemptionsAreReasoned` (AC2), `TestSetupShellParity` + `TestCheckDockerCompose` + `TestSetupParityTableIsComplete` (AC3), `tests/guard-setup-preexports-present.bats` (AC4), `TestHomeDeployMapCoversSetup` (AC5), and the `go build/vet/test` + `GOOS=windows go vet` + bats + shellcheck sweeps (AC6).
- The port-before-delete ordering (R3) holds: the map/checks land in `416233e` and the shell deletion bodies in the same commit; the parity table is a frozen historical record, and the setup deletion is guarded by `TestSetupShellParity` so a dropped item fails the suite.
- R1 (join guard) and R2 (exempt list measured, not inherited) are implemented and non-vacuous — see mutation evidence below. R4 (absence → SKIP) is honored and test-pinned. R5 (dead `sed -i` clause) is corrected (grep returns nothing).
- In-proposal scope: the deletions of `check_deployed`/`check_dependencies` bodies (`scripts/utils.sh`) and `Test-FileDrift` (`scripts/utils.ps1`) are covered by the tasks "Cleanup" item; no scope creep found.
- The Windows leg is deliberately out of scope and SKIPs, filed as OPS-046 (#1447) — a stated non-goal, not a hidden gap.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| Minor | THEORETICAL | tests (wiring) | The invocation of `checkHomeDeployDrift` + `checkDockerCompose` in `doctor.Run` is not backed by a committed regression test. The unit tests prove the functions in isolation and the parity/join tests prove the map&harr;script contract, but nothing asserts the checks actually run. | Reproduced by mutation: deleting the two `checkXxx(sys, ...)` lines from `doctor.go` leaves `go test ./internal/doctor/` green. Mitigated: I ran `dotf doctor` end-to-end against a temp HOME/DOTFILES_DIR with `.zsh/functions.sh` drifted → `[Deploy-dir↔$HOME drift] [FAIL] .zsh/functions.sh has drifted from …` and exit code 1, so the wiring is demonstrably live today. Only future-regression protection is missing. | UNTESTED — `TestCheckHomeDeployDrift` covers the function, not its invocation. | tests — add a `TestRun`-level test asserting the drift section yields exit 1 on a drifting fixture. |
| Minor | THEORETICAL | code/spec | `reportHomeDeployEntry` treats an absent deploy-dir source uniformly as "not provisioned → SKIP", including the 8 unconditionally-deployed files. A deploy-dir missing e.g. `.zsh/functions.sh` (deployed unconditionally at line 107) SKIPs rather than FAILs; `checkDeployDrift` also SKIPs on either-side-absent, so an incomplete deploy-dir would go unreported. | Code read of the `case !pathExists(src)` branch; consistent with R4 (conditional deploys) and with `checkDeployDrift`'s both-sides-present rule. By design and explicit, not a defect — but worth confirming the unconditional case merits SKIP. | `TestCheckHomeDeployDrift/"source absent from deploy dir → skip, not fail"` pins SKIP as the expected outcome. | spec/code — decide whether an unconditional source absent should FAIL (currently tested as SKIP). |
| Minor | THEORETICAL | tests | The join guard regex only matches the exact literal `deploy_file "$DOTFILES_DIR/<src>" "$HOME/<dst>"`. A future call using `${HOME}` or a variable base would evade it; the "changed shape" tripwire (`len(pairs)==0`) fires only if ALL calls change form. | Code read of `setupDeployFileCall`; all 11 current call sites match the literal form. | `TestHomeDeployMapCoversSetup` covers the literal-form calls (stales both directions). | code/tests — widen the regex or add a per-call-form tripwire. |
| Minor | REAL (measured, not a regression) | verification.md | `verification.md` states "1534 tests, exit 0"; in this environment the bats suite exits 1 with 7 `not ok` (install-dotf checksum/cross-compile cases 629–635 and skills-pipeline BUG-771 #1198). | Identical 7 failures reproduced at the merge-base `745f320` (verified via a temporary worktree), so they are pre-existing/environmental, not caused by OPS-043. | N/A (environmental) — the OPS-043-relevant bats all pass: `guard-setup-preexports-present.bats` 4/4, `iac-deploy.bats` 6/6, `setup-linux.bats` clean. | verification.md — re-state the bats result with the pre-existing caveat. |
| Question | THEORETICAL | spec | `ssh/config` is content-checked; a user editing `~/.ssh/config` directly (e.g. adding a Host alias) would turn doctor red as "drift" until they edit in repo + re-run setup. This is intended copy-deploy philosophy (ADR-012), but `ssh/config` is a plausible manual-edit target and the spec measured only `.gitconfig` as the observed drifter. | Code read of the map (`ssh/config` contentChecked) + `TestCheckHomeDeployDrift` covers ssh/config only in agree/absent cases, not a manual user edit. | No named test for an `ssh/config` user-drift scenario. | spec — confirm intent; consider exempting with a stated mechanism. |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | AC1–AC6 met and verified (unit + mutation + end-to-end exit-1); minor nuances: unconditional-source-absent SKIP, ssh/config manual-edit surface. |
| Verification       | B | Strong reproducible commands, mutation tests, CI on both OS, end-to-end smoke; gap: wiring proven by ad-hoc smoke not a committed test; bats exit-0 claim not reproducible here (pre-existing env failures). |
| Scope              | A | Diff matches proposal exactly; the `check_deployed`/`check_dependencies`/`Test-FileDrift` deletions are in-scope per tasks Cleanup; no creep. |
| Reliability        | A | Error paths branch correctly (unprovisioned/missing/symlink/drift/docker-absent); read-only and idempotent. |
| Maintainability    | A | Functions split ≤40 lines (22), WHY-comments, low cyclomatic complexity, no dead code. |
| Handoff-readiness  | A | proposal/tasks/verification complete; lesson 256 promoted; OPS-046 filed for the Windows leg; out-of-scope explicit. |

### Verdict

**PASS WITH GAPS** — no Blocker or Major findings; rubric is all A/B (no C, no D). Five Minor/Question items are tracked below. The change is genuinely solid: AC1–AC6 are verified by named tests, mutation testing proves the parity/ordering guarantees are non-vacuous, and an end-to-end run confirms the new `Deploy-dir↔$HOME drift` section FAILs on byte drift and exits 1.

### Recommended next steps (before archive)

1. **Add the wire-in regression test (Finding 1, the most substantive of the minors).** A `TestRun`-level test that sets up a drifted `homeDeployMap` fixture and asserts the doctor run emits `[Deploy-dir↔$HOME drift] [FAIL]` and exits 1. This converts the current ad-hoc smoke test into a committed, non-deletable guarantee — the class of gap the spec's own `verification.md` criticises ("a verification command that could not fail").
2. Optionally re-state the bats result in `verification.md` with the 7 pre-existing/environmental failures noted, so the claim matches a reproducible run.
3. Confirm (or exempt with a stated mechanism) the `ssh/config` content-check decision, and confirm the unconditional-source-absent SKIP is intended — both are design confirmations, not blocking.
4. `dotf spec archive` is **advisable** in the current state: no Blocker/Major, all criteria functionally verified, frontmatter valid, no `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the contract files. It may proceed with the minors tracked; the only item I'd strongly prefer addressed first is step 1.
