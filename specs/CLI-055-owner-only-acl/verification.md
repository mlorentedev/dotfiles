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
- [x] AC5 (service account) -> the SID comes from the process token (`fsmode_windows.go`), asserted by f5's grep; box run under a domain account

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
- **User + SYSTEM, protected, nothing else.** The user from the process token (CI's service account, a domain account, a local one — all the same code path), SYSTEM because backup and Defender already read every profile and would start failing on a file we hardened; Administrators deliberately absent — they hold `SeTakeOwnershipPrivilege` regardless, and an entry would be a claim the OS does not enforce.
- **"Applied" means "protected".** `Needs` on Windows reads the DACL's `SE_DACL_PROTECTED` flag rather than walking ACEs: protection is the invariant that matters (nothing inherited applies), and the ACL struct's entries are not exported by `x/sys/windows`. Perm comparison on Windows is the one bit `os.Chmod` can express — the owner write bit — so the other POSIX bits never read as drift.
- **Verified by the administrator's tool.** The Windows tests parse `icacls`, not our own reading of the security descriptor, so they check the consequence an operator would check.

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
