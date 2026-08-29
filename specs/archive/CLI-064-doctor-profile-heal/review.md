---
spec: "CLI-064-doctor-profile-heal"
verdict: "PASS WITH GAPS"
reviewed_sha: "389aaf50f3d0246b56b1e27871094694dbf44701"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-28"
---

## Adversarial review

**Scope**: CLI-064-doctor-profile-heal / PR #1353 (commit 5078d490, merged into HEAD 389aaf50)
**Sources**: `specs/CLI-064-doctor-profile-heal/{proposal,tasks,verification,features}.json` + `cli/internal/doctor/checks_profile.go`, `checks_profile_heal_test.go`, `checks_profile_test.go`, `doctor.go`, `scripts/profile-heal.ps1`, `tests/profile-heal-ps1.bats`, `env-contract.json`

**Reviewer independence**: This run is the detached pooled review (`dotf spec review`). `review-request.json` pins `reviewed_sha 389aaf50...` (== `git rev-parse HEAD`) and `reviewer nan/deepseek-v4-flash` (== the runner's `PI_MODEL`), the pool's primary. The change was implemented in a separate worktree (`dotfiles-wt-doctor-cluster`), so this pass is independent of the implementer.

### Spec and task alignment

- Proposal, tasks, verification and features.json are mutually consistent; every task `[x]` maps to a named test; every AC (1–5) has a named, runnable verification command. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain in any spec artifact.
- Diff scope matches the proposal exactly: `checks_profile.go` (+detection/heal), `checks_profile_heal_test.go` (new), one-line `doctor.go` wiring of `opts.Fix` + `contract`, `profile-heal.ps1` threshold `>2`→`>1`, and the `.bats` grep kept in sync. No unrelated churn; the only other change to `checks_profile_test.go` is the mechanical `checkProfileFiles(sys, nil, rep, false)` signature update.
- **Evidence re-run in this session (Windows box, go1.26)**: `go test ./internal/doctor/ -run TestCheckProfileFiles -v` → all three new funcs + existing `TestCheckProfileFiles` PASS (incl. the OneDrive-redirected, missing-profile, and WinPS 5.1 rows); `go build ./...` rc=0; `go vet ./...` rc=0; `golangci-lint run ./internal/doctor/` → 0 issues. The verification.md claim and the mutation-check result are credible; I did not re-derive the mutation case but the two tests it would flip are green and structurally independent of `profileCorruption` returning non-nil.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | cross-boundary (detect/heal target) | doctor validates the profile found by its **heuristic** path enumeration, but the heal operates on PowerShell's real `$PROFILE`. `repairProfile(sys, rep, profile, heal)` re-reads `profile` (doctor's found path) after the run, while the script resolves its own `$profilePath = $PROFILE` internally. If `findPowerShellProfile`'s 4 hardcoded roots (`~/Documents/{PowerShell,WindowsPowerShell}`, `~/OneDrive/Documents/{...}`) do not equal the `$PROFILE` a redirected box loads, doctor can (a) report the real profile healthy while it is corrupt (the exact false-PASS this spec exists to kill), or (b) run the heal on `$PROFILE` but re-measure its own different `profile`, yielding a spurious "ran but still corrupted". The realistic corporate root `OneDrive - <Org>` (this box's own PATH carries `OneDrive - Teledyne Technologies Inc`) is neither enumerated nor tested — proposal's OneDrive note covers only the exact `OneDrive/Documents` literal. Not observed here (dev profile found and PASSes), so THEORETICAL, not a reproduction. | UNTESTED — no test asserts `findPowerShellProfile`(doctor) == `$PROFILE`(script); the `OneDrive - <Org>` + redirected root rows are absent | code (resolve via known-folder/`$PROFILE`, or pass the profile path to the heal) + tests (force doctor-path == script-target; `OneDrive - X` fixture) |
| Minor | THEORETICAL | detection parity (AC4) | Go counts markers with `strings.Count(string(raw), …)` on **raw bytes**; the script counts via `[regex]::Matches($raw,…)` on **decoded** text (`Get-Content -Raw`). A UTF-16LE profile (WinPS's legacy default `Out-File`) holds markers with interleaved NULs, so Go sees 0→healthy while the script sees duplicates→heal — the "doctor and script agree on the threshold" guarantee (AC4) does not hold across encodings. Only bites when duplicates live in a <1 MB UTF-16 file (size signal still catches the big case), so narrow; surfaced for honesty, not gating. | UNTESTED — no UTF-16/BOM fixture; no byte-parity test | code (decode profile with BOM/UTF-16 detection before scanning, or normalize) + tests (UTF-16 fixture) |
| Minor | THEORETICAL | coupling of AC4 | The `>1` threshold is enforced by two **unlinked** literals — the Go test (`>1`) and the `.bats` grep on `-gt 1` — so neither test catches the other drifting to `>2`. The agreement that AC4 certifies is not asserted by any single cross-check. Currently correct (verified: Go `if starts>1||ends>1`; PS `-gt 1`; size `>1<<20` == `>1MB` both 1048576), so the gap is a future-drift hazard, not a current defect. | UNTESTED — no test asserts doctor-threshold == script-threshold | tests (one cross-check, e.g. `.bats` greps the Go source's `> 1`) |
| Minor | SPECULATIVE | reliability | `repairProfile` shells out via unbounded `sys.CommandOutput("pwsh",…)`; other doctor checks use `CommandOutputBounded` for the same reason. A hung `pwsh` (pathological profile, SSOT over a slow/network mount) would hang `dotf doctor --fix` with no timeout. The script itself always exits and the work is local, so this is low-probability. | UNTESTED — no timeout in the seam test | code (bound the pwsh call) |
| Minor | THEORETICAL | data loss | `--fix` runs a heal that **replaces the whole profile** from the SSOT (START + SSOT + END), discarding any user content outside the dotfiles section — recoverable only from the timestamped `.corrupted-*.bak`. Target divergence (Finding 1) can silently aim this at the wrong file. Disclosure exists (the FAIL/Fix messages name the backup), and "pruning non-dotfiles content" is declared out of scope, so not a spec violation — but a naive `doctor --fix` on a corrupted profile does destroy user `$PROFILE` customizations by design. | UNTESTED — no test asserts non-dotfiles content survives or that the user is warned before wipe | code (stronger pre-fix warning / confirm) + spec (deliberate scope note) |
| Question / assumption | SPECULATIVE | scope | The parser-error signal (signal 3, still in the script) is deliberately not checked by doctor. So a small profile with one marker pair but parser errors reads as **healthy** under `--fix` and is never healed — the same "doctor blesses a broken profile" shape this spec set out to fix, just for the parser class. Declared out of scope, so not a violation; flagging so the residual is a tracked decision rather than a silent gap. | UNTESTED / no feature behavior | spec (follow-up issue or explicit ADR) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All 5 ACs met with named, re-run green tests incl. negative paths; cross-boundary (encoding, profile-target identity) negative cases untested |
| Verification       | A | Evidence reproducible: I re-ran build/vet/lint/tests; mutation check independently credible |
| Scope              | A | Diff matches proposal exactly; no unrelated changes |
| Reliability        | C | Guards (missing script, no pwsh, no-op heal, healthy) are excellent, but heal target can diverge from the validated path and the pwsh call is unbounded |
| Maintainability    | A | Clear naming, small functions, comments explain WHY, lint/vet clean, threshold logic centralized in four named consts |
| Handoff-readiness  | B | Spec complete with promotion + archive checklist; archive steps pending (correct for this phase) |

### Verdict
PASS WITH GAPS

### Recommended next steps (before archive)
- **Do not block archive on the THEORETICAL findings** — none is a reproduction, and each is scoped. The change is fit to land as-is; the gaps are hardening, not defects.
- Address Finding 1 (highest value) as a follow-up: resolve the profile via the known-folder/`$PROFILE` (or pass the doctor-found path to the heal so heal-and-re-read target the same file), add a `OneDrive - <Org>`/redirected fixture, and add a test asserting doctor-found path == script `$PROFILE` target. File it on bitácora before closing #531 if not done in this merge.
- Add the AC4 coupling test (Finding 3) and a UTF-16/BOM fixture (Finding 2) so the parity doctor/script agree on is actually enforced by one test.
- Resolve the Finding 5/6 question explicitly (parser-error profile remains undetectable by design) — a one-line scope note or a backlog issue so the residual is tracked, not silent.
- `dotf spec archive CLI-064-doctor-profile-heal` is **advisable** in the current state (no blocker, no D, minors+one THEORETICAL Major tracked). Suggested order: fix Finding 1 in PR #1353 first if the follow-up is cheap; otherwise archive now with the findings filed as bitácora issues, then close #531 with the PR link.
