---
tags: [spec, verification, templates]
created: "2026-08-29"
---

# Verification - CLI-055-owner-only-acl

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (0600 → protected DACL, user + SYSTEM; POSIX → chmod) -> commit `a04c95f` / `TestApply_OwnerOnlyModeSetsAProtectedOwnerOnlyDACL` (Windows), `TestApply_IsChmodOnEveryOS`, `TestOwnerOnly`
- [x] AC2 (0644 → inherited ACL untouched) -> `TestApply_SharedModeKeepsTheInheritedACL`, `TestNeeds_SeesAnInheritedDACLAsMissingOwnerOnly`
- [x] AC3 (writers use `fsmode.Apply`; in-sync content still gets its mode) -> `TestDeploy_InSyncContentStillGetsItsDeclaredMode`, `TestNeeds_MirrorsApply`; `grep os.Chmod(` in the writers → 0
- [x] AC4 (box) -> transcript below, Windows work box, 2026-08-29
- [x] AC5 (any account: an administrator on CI's hosted runner, a domain account on the box) -> the SID comes from the process token (`fsmode_windows.go`), asserted by f5's grep; box run under a domain account

## Test status

- Test suite: `cd cli && go test ./... -count=1` -> every package `ok`, `FAIL_COUNT=0` (on the Windows box, so the Windows-only tests ran for real); `go vet` clean under `GOOS=windows` and `GOOS=linux`; `golangci-lint run` (pinned 2.12.2) `0 issues`
- Manual smoke test (AC4), binary built from this branch, `DOTFILES_REPO_DIR` at the worktree:

  ```text
  --- dry run ---
  would fix mode pi    C:\Users\<user>\.pi\agent\models.json
  --- run 1 ---
  mode fixed pi        C:\Users\<user>\.pi\agent\models.json
  --- icacls 0600 ---
  C:\Users\<user>\.pi\agent\models.json TDY\<user>:(F)
  NT AUTHORITY\SYSTEM:(F)
  --- run 2 ---
  in sync   pi         C:\Users\<user>\.pi\agent\models.json
  --- 0644 neighbour keeps (I) ---
  inherited entries: 3
  --- pi still reads the file? ---
  2903
  ```

  Before this branch's `Needs`, the very first run printed `in sync` and `icacls` showed the
  three inherited `(I)` entries untouched — the finding that produced task 3.
- No regressions in existing test suite: yes

## Decisions made during implementation

- **The mode is part of "deployed".** The in-sync path compared bytes only, so a 0600 file deployed by a binary that could not express owner-only stayed inherited forever. `Deploy` now asks `fsmode.Needs` on the in-sync path and applies the mode without a content rewrite, reported as `mode fixed` (`would fix mode` on `--dry-run`) — a distinct word, because "deployed" would claim a rewrite that did not happen.
- **User + SYSTEM, protected, nothing else.** The user from the process token (an administrator on CI's hosted runner, a domain account, a local one — all the same code path), SYSTEM because backup and Defender already read every profile and would start failing on a file we hardened; Administrators deliberately absent — they hold `SeTakeOwnershipPrivilege` regardless, and an entry would be a claim the OS does not enforce.
- **"Applied" means "protected".** `Needs` on Windows reads the DACL's `SE_DACL_PROTECTED` flag rather than walking ACEs: protection is the invariant that matters (nothing inherited applies), and the ACL struct's entries are not exported by `x/sys/windows`. Perm comparison on Windows is the one bit `os.Chmod` can express — the owner write bit — so the other POSIX bits never read as drift.
- **Verified by the administrator's tool.** The Windows tests parse `icacls`, not our own reading of the security descriptor, so they check the consequence an operator would check.

## Review round 1 — dispositions

`nan/deepseek-v4-flash`, verdict FAIL, reviewed at the landing commit `3f554ad` (retroactive review, #1626). The implementation was judged sound; the FAIL was the contract set.

| # | Finding | Disposition |
|---|---|---|
| 1 | Major, REAL: f5 greps for `OpenCurrentProcessToken`, which the code does not call, so AC5's proof could not pass | **Applied**. f5 greps for `OpenProcessToken(windows.CurrentProcess()` and `GetTokenUser()`, and exits 0. Its `evidence` named the wrong call too, and is corrected. The code keeps the explicit form, which the pinned `golang.org/x/sys` recommends over the deprecated shorthand. |
| 2 | Major, THEORETICAL: `ownerOnlyApplied` reads only `SE_DACL_PROTECTED`, so a protected but permissive DACL reads as owner-only | **Ticketed: #1663** (CLI-084). It is a Windows-only regression test and a fix in `fsmode_windows.go`, testable on CI's `windows-latest` leg. |
| 3 | Minor: the proposal's What said "nothing else changes shape", but `Deploy`'s in-sync path changed | **Applied** in the proposal. |
| 4 | Minor: doctor and deploy disagree on a mode-only drift | **Ticketed: #1664** (CLI-085). |
| 5 | Minor: the `Apply` calls in `commit()` and `stage()`, and the two new output lines, are unpinned | **Ticketed: #1664**. |
| 6 | Minor: three stale comments | **Ticketed: #1664**. |
| 7 | Minor: `fsmode.go` is not gofmt-clean | **Ticketed: #1664**. |
| 8 | Minor, SPECULATIVE: the `icacls` footer is filtered by an English literal | **Declined.** It can only cause a false failure, never a false pass, and every box this runs on is English. |

f4 exits 1 off Windows by design, and since round 2 it says so: its behaviour names the platform, and it prints "not this platform" before exiting.

## Review round 2 — dispositions

`nan/deepseek-v4-flash`, verdict FAIL, reviewed at `8a110d7` (round 1's contract fix). Round 1's fix held: f5 passes. This round found a different vacuous verifier.

| # | Finding | Disposition |
|---|---|---|
| F1 | Major, REAL: f2 runs zero tests off Windows and exits 0, because both tests it names are build-tagged `windows` | **Applied** to f1, f2 and f5. On Windows each runs its named tests with a guard on the number of `=== RUN` lines. Elsewhere it requires every named Windows test to exist in `fsmode_windows_test.go` and the package to compile with `GOOS=windows`. Measured on Linux: all three exit 0, and renaming any one test makes its verifier exit 1. The Windows branch runs only where `uname` says MINGW (the box); CI runs the tests themselves on `windows-latest`. |
| F2 | Major, THEORETICAL: `ownerOnlyApplied` reads only `SE_DACL_PROTECTED` | **Ticketed: #1663** (unchanged from round 1). |
| F3 | Minor, REAL: the in-sync mode convergence runs on POSIX too and converges both ways, so a file an operator tightened is silently widened back to the declared mode and reported as `mode fixed` | **Ticketed: #1664**, as a named defect there. It is behaviour on the primary platform, so it is not left as a note. |
| F4 | Minor, REAL: the two new output lines do not share the other lines' name column | **Ticketed: #1664.** |
| F5 | Minor, REAL: "the writers go through `fsmode.Apply`" is pinned only by a grep in f3, and nothing in CI runs `features.json` verifiers | **Ticketed: #1664** for the call-site pins. That no workflow executes a spec's verifiers is systemic, and it is tracked on the harness epic, not here. |
| F6 | Minor, REAL: `fsmode.go` is not gofmt-clean, and nothing in CI runs gofmt | **Ticketed: #1664.** |
| F7 | Minor, REAL: "CI's service account" is not what a hosted runner is | **Applied** in the proposal's risk note, AC5, f5's behaviour and evidence, and tasks.md. |
| F8 | Minor, SPECULATIVE: `GetNamedSecurityInfo` could return `(nil, nil)` | **Declined.** It is unreachable for a file with a DACL, which is every file this code touches. |

## Review round 3 — dispositions

`nan/glm5.3-flash`, verdict **PASS WITH GAPS**, reviewed at `1da0db0` (round 2's contract fix, on top of the landing commit `3f554ad`). The contract set is closed. Each gap is dispositioned here.

| # | Finding | Disposition |
|---|---|---|
| 1 | Major, THEORETICAL: `ownerOnlyApplied` keys only on `SE_DACL_PROTECTED` | **Ticketed: #1663**, unchanged since round 1. |
| 2 | Major, REAL, out of scope: `TestMergeAgainstTheRealDeployedSettings` failed at `3f554ad` on this machine (a foreign hook dropped, 17 to 16) | **Not reproducible on `main`.** At `08b8e19` it passes: "5 distinct foreign hook commands, none belonging to a third party lost". The reviewer ran it at the landing commit, against the code of that day. No ticket. |
| 3 | Minor, REAL: f4 asserts less than its behaviour states (not the user's ACE, not "exactly two", not the 0644 neighbour) | **Declined.** f4 is the box smoke check. The exactly-two-ACEs property is AC1's, pinned by `TestApply_OwnerOnlyModeSetsAProtectedOwnerOnlyDACL` on CI's Windows leg, and the 0644 neighbour is AC2's `TestApply_SharedModeKeepsTheInheritedACL`. The box transcript for all three is in the evidence above. |
| 4 | Minor, REAL: the rendered in-sync path's `ensureMode` call site is untested | **Ticketed: #1664.** |
| 5 | Minor, REAL: `fsmode.go` is not gofmt-clean | **Ticketed: #1664** (round 1, #7). |
| 6 | Minor, THEORETICAL: the in-sync mode fix follows a symlinked destination | **Ticketed: #1664**, alongside the other in-sync mode behaviours. |
| Q | On POSIX, the in-sync convergence widens a file an operator tightened | **Ticketed: #1664**, as its named defect 5. It is not left implicit. |

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons/`? no — the finding ("in sync by content is not in sync by mode") is recorded here and in the test's comment; it is a property of this deploy, not a cross-cutting class yet
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no
- [ ] New pattern candidate for `00_meta/patterns/`? no

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-055-owner-only-acl/` -> `specs/archive/CLI-055-owner-only-acl/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
