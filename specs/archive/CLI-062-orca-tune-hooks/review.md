---
spec: "CLI-062-orca-tune-hooks"
verdict: "PASS WITH GAPS"
reviewed_sha: "b1ae0c9822072429308b4c208b4f95cc974ba443"
reviewer: "nan/mimo-v2.5"
date: "2026-09-25"
---

## Adversarial review

**Scope**: CLI-062-orca-tune-hooks (`git diff 3f554ad73aa8253a09b5dc926abb26599de408c3...HEAD`, 3 commits, 17 files changed, +893/-566)
**Sources**: `specs/CLI-062-orca-tune-hooks/{proposal,tasks,verification,features}.md`, full diff, `go test`, `gocyclo`, `golangci-lint`, `bats`

### Spec and task alignment

All five acceptance criteria are mapped to specific tests and verified:

- **AC1** (repair both files with backups; idempotent): `TestTuneHooks_RepairsBothFilesWithBackups`, `TestTuneHooks_SecondRunChangesNothing` — both green. The byte-for-byte block pinning (`TestTuneScript_WritesTheRetiredScriptsBlockByteForByte`) catches the exact failure mode the `Decisions made during implementation` section documents (Go's `$req` template expansion eating PowerShell variables). **Met.**
- **AC2** (`--check`, nothing to do, generous timeout, unrecognised POST): Four tests cover each sub-case. The `--check` exit-1-on-drift path is exercised both in the orca package tests and in `TestRunOrcaTuneHooksUnrecognisedIsNotInSync` (cmd package). **Met.**
- **AC3** (doctor `--fix` through the package; remedy lines name the command): `TestCheckOrcaHook` table (8 sub-cases), `TestCheckOrcaHook_Fix`, `TestCheckOrcaHook_FixTunesTheScript`, `TestCheckOrcaHook_FixLeavesAnUnrecognisedPostAndFails`. Grep verified both FAIL lines contain `dotf orca tune-hooks`. **Met.**
- **AC4** (setup calls the command, retires the script; four files deleted; audit amended): `bats tests/setup-windows.bats -f 'CLI-062'` passes 1/1. `git ls-files` confirms the four deleted files are absent. `audit-007` row says "ported". `setup-windows.ps1` lists `orca-hook-tune.ps1` in `$retiredScripts` and removed from `$deployedScripts`. **Met.**
- **AC5** (Windows box, manual): Transcript in `verification.md` covers real files (`--check` exit 0), scratch copies with DX-006 defects (`--check` exit 1, then `tune` fixes + backups, then `--check` exit 0). Flagged as platform-gated. **Met.**

All tasks in `tasks.md` are ticked and each maps to verified diff evidence. No unchecked boxes without implementation.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | quality | `checkOrcaHook` (CC=15), `runOrcaTuneHooks` (CC=14), `TuneHooks` (CC=12) all exceed the project's <10 cyclomatic complexity threshold. `checkOrcaHook` was already CC=13 before this change (+2 from the new script-repair path). The complexity comes from necessary branching on file existence, drift, fix mode, and unrecognised lines — not from accidental nesting. | `gocyclo -over 10` on the three files; pre-change baseline of `checkOrcaHook` measured at CC=13 via `git show 3f554ad7^:...` | `TestCheckOrcaHook` (8 sub-cases), `TestRunOrcaTuneHooksUnrecognisedIsNotInSync`, `TestTuneHooks_*` (7 tests) | code (refactor opportunity, not blocking) |
| Minor | THEORETICAL | resilience | `writeTuned` uses a predictable temp-file path (`path + ".tmp"`). Two concurrent calls to `writeTuned` with the same target path would race on the temp file. Low probability in practice (desktop CLI, single invocation), but `os.CreateTemp` in the same directory would eliminate it with no API change. | `hooks.go:231` — `tmp := path + ".tmp"` | UNTESTED | code |
| Minor | THEORETICAL | quality | `ScriptUsesInvokeWebRequest` compiles a fresh `regexp.MustCompile` on every call. The other two regexes in the file (`timeoutRe`, `invokeWebRequestRe`) are package-level vars. This function is called in the `TuneHooks` → `TuneScriptFile` → `ScriptUsesInvokeWebRequest` path, so it may run twice per invocation. | `hooks.go:82` — `regexp.MustCompile("Invoke-WebRequest")` inside function body | UNTESTED | code |
| Minor | THEORETICAL | consistency | `doctor --fix` for the JSON half writes directly (`os.WriteFile`) without backup, while the script half goes through `TuneScriptFile` which backs up via `writeTuned`. A `doctor --fix` that repairs the JSON leaves no rollback point. Pre-existing asymmetry (old doctor code also didn't back up JSON), and `dotf orca tune-hooks` (the primary entry point) backs up both — so this is not a regression, but the asymmetry is worth noting. | `checks_orca.go:53` — direct `os.WriteFile(orcaJSON, tuned, 0o644)` vs `checks_orca.go:76` — `orca.TuneScriptFile` | UNTESTED (doctor JSON backup) | code |

No Blocker or Major findings.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A  | All five ACs verified with named tests covering happy, edge (generous timeout, unrecognised POST, nothing-to-do), and idempotency paths. |
| Verification       | A  | Every AC has specific test names + commit hashes in `verification.md`; AC5 has a box transcript; all tests green in this session. |
| Scope              | A  | Diff matches proposal exactly: new `orca/hooks.go` package, cmd wiring, doctor refactoring, setup-windows integration, four file deletions, audit-007 amendment. No unrelated changes. |
| Reliability        | B  | Atomic writes via temp+rename, backup before write, unrecognised lines left untouched. Predictable temp path is the only gap (THEORETICAL, Minor). |
| Maintainability    | B  | Clear naming, good doc comments explaining the WHY (DX-006, lesson 111, the Go template expansion trap). Three functions exceed CC<10 but all are straightforward switch-style branching. |
| Handoff-readiness  | A  | `features.json` filled, `verification.md` complete with evidence, `tasks.md` fully ticked, audit-007 amended. No promotion candidates needed. |

### Verdict

**PASS WITH GAPS**

All four findings are Minor and THEORETICAL. No Blocker or Major findings. The rubric has no C or D grades. The change is well-implemented, thoroughly tested, and correctly scoped.

### Recommended next steps

1. **Code (follow-up, not blocking archive):** Extract `ScriptUsesInvokeWebRequest`'s regex to a package-level `var` alongside the other two. (Minor, THEORETICAL quality.)
2. **Code (follow-up, not blocking archive):** Replace `path + ".tmp"` in `writeTuned` with `os.CreateTemp` in the same directory to eliminate the theoretical race. (Minor, THEORETICAL resilience.)
3. **Code (follow-up, not blocking archive):** Consider refactoring `checkOrcaHook` to reduce CC below 10 — the two `pathExists` blocks could be extracted into helpers. This was pre-existing debt (CC=13) that grew by 2; worth addressing but not introduced by this change alone. (Minor, THEORETICAL quality.)
4. **Spec (no action needed):** The asymmetry between doctor's JSON fix (no backup) and script fix (backup) is pre-existing and acceptable — `dotf orca tune-hooks` is the primary entry point and backs up both. No spec change needed.

**Is `dotf spec archive` advisable?** Yes. All ACs verified, no blockers, no majors, all tasks complete, `features.json` filled. The four Minor findings are tracked above and do not gate archive.
