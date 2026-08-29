---
spec: "CLI-058-env-persist"
verdict: "PASS"
reviewed_sha: "389aaf50f3d0246b56b1e27871094694dbf44701"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-29"
---

## Adversarial review

**Scope**: CLI-058-env-persist (HEAD `389aaf5`, "feat(env): dotf env persist writes the contract variables where a profile-less process reads them (#1362)")
**Sources**: `specs/CLI-058-env-persist/{proposal,tasks,verification,features}.json` + `git show HEAD` (17 files, +749/-1) covering `cli/internal/env/persist{,_windows,_other,,_test}.go`, `cli/internal/cmd/env_persist.go`, `cli/internal/doctor/checks_env_persist{,_test}.go`, `setup-windows.ps1`, `tests/setup-windows.bats`, `cli/go.mod`.

### Spec and task alignment

- All six acceptance criteria are present, testable, and ticked in `tasks.md`. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain — the archive gate's own precondition holds.
- The implementation matches the proposal precisely: `env.ResolveVars` is the shared resolver behind both `generate` (`Resolve`) and `persist` (`ResolveVars` → `Resolve`), so "one resolver, two sinks" is real, not claimed. Confirmed: both load the contract + machine.json and produce the same `ResolvedVar` list.
- **Non-goal checked**: `Resolve` skips vars with no per-OS default (HOME / USERPROFILE are `required_on` only, no `default` key → `raw==""` → skipped), so `persist` never writes the OS-provided home vars. Confirmed from `env-contract.json` and `resolve.go`.
- **Critical negative check passed**: none of the 11 persisted variables is `PATH` or any other system/ExpandableString var — the list is DOTFILES_DIR, DOTFILES_REPO_DIR, CLAUDE_CONFIG_DIR, SCRIPTS_DIR, AGY_HOME, COPILOT_HOME, OPENCODE_HOME, VAULT_PATH, HIVE_VAULT_PATH, AGE_KEY_PATH, SOPS_AGE_KEY_FILE (all REG_SZ paths). There is no PATH-clobbering risk from writing a partial absolute PATH to `HKCU\Environment`.
- The box evidence is internally consistent: first `--check` reported 10 drifted (one var already present), `persist` wrote 10 / left 1 unchanged, second `--check` reported all 11 ok, second `persist` 0 changed — matching `TestPersist_TouchesOnlyWhatDiffers`'s idempotence contract.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | AC3 no-op (POSIX) | The `errors.Is(err, env.ErrUserEnvUnsupported) → print + return nil` branch in `env_persist.go` is only **compile**-verified (`GOOS=linux go vet`, CI ubuntu-latest compiling both files). No test executes `RunE` with the error injected to assert the exit-0 "nothing to persist" contract. `verification.md` overstates: compiling is not exercising. Consequence is currently low (no POSIX setup script calls `persist` — `setup-linux.sh` does not), but a future refactor could flip the exit code and break setup symmetry silently. | code read of `env_persist.go:36-43` + `persist_other.go`; no `userEnvStore`/`ErrUserEnvUnsupported` reference exists in `cli/internal/cmd/*_test.go` | UNTESTED | tests (add a `newEnvPersistCmd`/Execute case with `userEnvStore` returning `ErrUserEnvUnsupported`, assert exit 0 + message) |
| Minor | THEORETICAL | cascade inversion / stale registry | A persisted User-scope value is non-empty, so it satisfies cascade rule #1 (`if (-not $env:VAR)` in paths.ps1 and `${VAR:-…}` in paths.sh both skip), making the registry authoritative over the rc files for these vars. If `machine.json` is later changed without re-running `dotf env persist`, the stale registry value silently overrides what `generate` renders. Mitigated by design (setup re-persists each run; the new doctor section WARNs naming the stale vars + the one-command remedy; `persist` re-reads machine.json fresh) and documented in the proposal's Risks. Flagged so the coupling is on record, not because it should gate. | code read of `Generate`/`Render` (`if (-not $env:VAR)`, `${VAR:-…}`) vs `persist` unconditional `Set`; proposal Risks section | `TestCheckPersistedEnv_ByStatus` "one different → WARN naming it" proves doctor's stale detection | spec (already documented); consider a doctor hint, not required |
| Minor | SPECULATIVE | perf | `registryUserEnv.Set` broadcasts `WM_SETTINGCHANGE` after **each** of the N changed writes (N broadcasts when all N differ), instead of once after `Persist` completes. Wasteful; final broadcast gives the correct end state, so no correctness impact. | code read of `persist_windows.go:Set` | UNTESTED (inherently OS-only, not unit-tested) | code (hoist broadcast to a single call after the write loop) — optional |
| Minor | Maintainability | declared seam unused | `var userEnvStore = env.NewUserEnvStore` in `cmd/env_persist.go` is a test seam nothing tests; the persistence logic is covered at the `env` and `doctor` package layers but not at the cobra-command layer (`--check` flag wiring, exit codes, stdout text). Box evidence covers it; a future command wiring regression would be caught only by hand. | `cli/internal/cmd/` has no `env_persist_test.go`; the seam is unused | UNTESTED | tests (optional) |

Reality tags: no REAL failures found. The strongest candidates (PATH clobbering, secret exposure via AGE_KEY_PATH/SOPS_AGE_KEY_FILE) were **disproven** by reading the contract (no PATH var; the persisted values are paths to the key file, never the key — the proposal documents this, and `Resolve` persists paths, not secrets). All findings are Minor and none changes the verdict.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | AC1/2/4/5 have named tests + documented mutations; AC6 box-verified with a concrete `-UseNewEnvironment` repro; AC3 no-op is compile-only (Minor gap). |
| Verification       | B | `verification.md` is reproducible for the Go suite and the bats case; AC6 is box-specific as appropriate; AC3 claim slightly overstates (compiling ≠ exercising). |
| Scope              | A | Diff matches the proposal exactly; 17 in-scope files; the new direct dependency `golang.org/x/sys` is declared and justified. No creep. |
| Reliability        | B | Store errors wrapped with the variable name, partial-failure returns what it did, unreadable → WARN, DR broadcast best-effort, idempotent by construction. |
| Maintainability    | B | Functions well under 40 lines, CC ≤10, `UserEnvStore` interface + build-tag split keep Windows out of the logic; one unused command seam. |
| Handoff-readiness  | A | Proposal `status: verifying`, promotion candidates declared (no lesson/ADR needed), archive checklist present; only the pooled review was pending. |

### Verdict
PASS

A single Minor, THEORETICAL, UNTESTED finding (AC3's POSIX no-op branch) and three Minor notes — none rises to Major or Blocker, and no finding is REAL. Per the aggregation rule (all B or above → PASS) the change is solid and the acceptance criteria are met.

### Recommended next steps (before archive)
- `dotf spec archive CLI-058-env-persist` is **advisable** now: the review gate is freshly satisfied, no draft tags remain, and no blocker/major blocks it.
- Optional, non-gating: add a `cmd`-layer test that points the `userEnvStore` seam at `ErrUserEnvUnsupported` and asserts exit 0 / the no-op message, closing the AC3 runtime gap; hoist the `WM_SETTINGCHANGE` broadcast to once per run (perf nicety). Ticket either if not done in-scope so it is not lost.
- On merge, close bitácora #1324 with the PR link (per the archive checklist).
