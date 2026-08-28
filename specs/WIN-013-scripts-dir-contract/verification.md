---
tags: [spec, verification]
created: "2026-08-28"
---

# Verification - WIN-013-scripts-dir-contract

## Evidence

Run on the Windows work box, 2026-08-28, worktree `dotfiles-wt-setup-cluster`,
branch `fix/scripts-dir-contract`.

- [x] **AC1** → `tests/env-contract.bats` "SCRIPTS_DIR: the Windows default,
  required_path_entries and setup-windows.ps1 name the same directory".
- [x] **AC2** → `tests/setup-windows.bats` "removes every retired script and the
  legacy ~\scripts copies of the live ones": seven names present in the removal
  list, `$LegacyScriptsDir` declared, both-dirs sweep, no `SetEnvironmentVariable("PATH"…LegacyScriptsDir`.
- [x] **AC3** → `.github/scripts/doctor-gate-known-failures.txt` carries no
  entry; `tests/doctor-gate.Tests.ps1` 10/10 (the example-match guard now asserts
  the parser's count, 0 included); `Test-DoctorGate` with `@()` patterns and no
  FAIL lines → 0 unexpected, 0 stale.
- [x] **AC4** → the MEM-002, CLI-019 and CLI-018 guards refute `Copy-Item.*name`
  and `&.*name` instead of any mention.
- [x] **AC5** → CI `test-windows` on `1076ed0` (run 33201808112): `doctor gate: 0 known runner-only FAIL(s), 0 unexpected, 0 stale`. The first run of the PR failed `integration` on the second `setup-linux.sh` run with no output; the re-run with the same setup code passed, and the test now prints the run's last 40 lines so a repeat names itself.

## Test status

```
bats tests/env-contract.bats tests/setup-windows.bats tests/ci-windows-doctor-gate.bats tests/profile-heal-ps1.bats   -> all ok
Invoke-Pester tests/doctor-gate.Tests.ps1   -> 10/10
Invoke-ScriptAnalyzer setup-windows.ps1 (repo settings)   -> 0 findings (same as main)
Parser::ParseFile setup-windows.ps1   -> 0 parse errors
non-ASCII characters in setup-windows.ps1   -> 11 (unchanged from main)
```

- No regressions in the existing suite: yes.
- `setup-windows.ps1` was NOT run on this box from the branch (the main
  session owns machine state); the real migration runs after merge and its
  outcome is recorded on #1310.

## Decisions made during implementation

- **Align Windows to the contract, not the contract to Windows.** ADR-025 makes
  the contract the SSOT and `dotf harness mirror` (#1305) already put the deploy
  tree under `~\.dotfiles`; scripts follow.
- **Sweep both locations every run.** Idempotent removal of a fixed list is
  cheaper and more honest than a one-time migration flag that nothing verifies.
- **Leave the legacy PATH entry.** Rewriting a User PATH from a script is the
  2048-char hazard #148 named; the new entry is prepended, so resolution order is
  correct without pruning.
- **Guards measure deployment, not mention.** Three existing guards failed
  because the removal list names the retired scripts — the lesson-235 shape; they
  now refute `Copy-Item`/invocation.

## Promotion candidates

- [ ] Lesson: no — the mention-vs-invariant lesson already exists (235).
- [ ] ADR-worthy decision: no.

## Archive checklist

- [ ] `dotf spec review WIN-013-scripts-dir-contract` PASS
- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/WIN-013-scripts-dir-contract/`
- [ ] Bitácora #1310 closed with the PR link
