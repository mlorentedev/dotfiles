---
spec: "CLI-055-owner-only-acl"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "1da0db0c73b54fe98204ddc0760dd1bf643ff959"
reviewer: "nan/glm5.3-flash"
date: "2026-09-24"
---

## Adversarial review

**Scope**: `CLI-055-owner-only-acl`, the whole change: `git diff dab595e34a72d6c304fb3335061ecda0a26297fa...1da0db0` (`3f554ad` the implementation, `8a110d7` + `1da0db0` the round-1/round-2 contract corrections). Round 3. The implementation was reviewed as it stands at HEAD, not as a delta since round 2; round 2's findings (F1–F8) were re-checked against the files they touched.

**Sources**: `specs/CLI-055-owner-only-acl/{proposal,tasks,verification}.md`, `features.json`, `cli/internal/{fsmode,deploy,secrets,cmd}`, `.github/workflows/{cli,ci}.yml`, PR #1380.

### Spec and task alignment

- AC1↔`TestApply_OwnerOnlyModeSetsAProtectedOwnerOnlyDACL` + `TestApply_IsChmodOnEveryOS` + `TestOwnerOnly`; AC2↔`TestApply_SharedModeKeepsTheInheritedACL` + `TestNeeds_SeesAnInheritedDACLAsMissingOwnerOnly`; AC3↔`TestDeploy_InSyncContentStillGetsItsDeclaredMode` + `TestNeeds_MirrorsApply` + the f3 grep; AC4↔box transcript; AC5↔process-token resolution + CI. Every AC maps to a named test or a recorded box artifact. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain.
- Verified by running, this session, on Linux: `go build ./...`, `go vet` (host, `GOOS=linux`, `GOOS=windows`), `go test ./...` (one failure — finding 2, pre-existing at base), `golangci-lint run` (pinned 2.12.2, 0 issues), all five `features.json` verifiers (f1/f2/f3/f5 exit 0; f4 exits 1 off-Windows with the documented message), and the round-2 mutation claim — renaming `TestApply_SharedModeKeepsTheInheritedACL`, `TestApply_OwnerOnlyModeSetsAProtectedOwnerOnlyDACL` each makes f2/f1/f5 exit 1. All mutations reverted; working tree clean apart from this review.
- AC5's CI premise verified structurally and by outcome: `.github/workflows/cli.yml` runs `go test ./...` on `windows-latest` for PRs touching `cli/**`, and PR #1380's check rollup shows `test (windows-latest)` → SUCCESS. The fsmode Windows tests carry `//go:build windows` and no `t.Skip`, so they ran for real under the hosted runner's account.
- `grep -rn "os.Chmod("` across `cli/`: only the fsmode wrapper itself, plus `internal/harness/mirror.go` (0644) and `internal/doctor/checks_memshape.go` (0644 / preserves existing). Both remaining sites are shared modes where fsmode would be a no-op beyond `os.Chmod` on Windows — no owner-only writer was missed; AC3's "the three writers" claim holds.
- Round 1 and round 2 dispositions were re-checked and hold: f5's verifier greps for `OpenProcessToken(windows.CurrentProcess()` / `GetTokenUser()` and exits 0 (run this session); every Windows-only-test verifier now requires existence and compilation off-Windows (mutation-verified).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | Windows ACL | `ownerOnlyApplied` keys only on `SE_DACL_PROTECTED`, so a protected-but-permissive DACL (e.g. someone ran `icacls /grant` on a hardened file) reads as owner-only: `Needs` misses the drift and the in-sync path never repairs it. Unchanged since round 1; ticketed twice. | Code read of `cli/internal/fsmode/fsmode_windows.go` (`ownerOnlyApplied`); no reproduction attempted. | UNTESTED — no test reprotects a DACL with broader entries and asserts `Needs` sees it. | code + tests — already ticketed as #1663 (CLI-084). Tracked, not blocking per the reality rule. |
| Major | REAL — **out of this change's scope** | harness (pre-existing) | `TestMergeAgainstTheRealDeployedSettings` fails on this box: `MergeHooks` drops a foreign hook (17→16) on the real deployed `~/.claude/settings.json`. Ran identically at HEAD and at base `dab595e` in a throwaway worktree; nothing in this diff touches `internal/harness`. It does **not** gate this spec's archive — but it is a live defect in the suite and must not be lost. | Session output: `FAIL: TestMergeAgainstTheRealDeployedSettings ... foreign hooks 17 -> 16 on the REAL file`, reproduced at base. | The named test itself (it found the bug); the fix needs its own regression test. | code + tests — ticket on the bitácora with the base-commit evidence. Disposition in `verification.md`. |
| Minor | REAL | spec verifier (contract set) | f4's command is weaker than its behavior text: it asserts no `(I)`, `NT AUTHORITY\SYSTEM:(F)`, and a second-run `in sync`, but never the calling user's ACE, the "exactly two ACEs" property, or the 0644 neighbour — it would exit 0 on a file with an extra ACE or a missing user entry. Bounded: AC1's Windows unit test (`TestApply_OwnerOnlyModeSetsAProtectedOwnerOnlyDACL`, run on the box via f1) asserts exactly two ACEs incl. the token user. | Read of `features.json` f4 vs. its `behavior` field. | UNTESTED as a verifier; covered indirectly by the AC1 unit test on Windows. | spec (`features.json`) — contract set, so NOT editable alongside this verdict; disposition in `verification.md` (apply post-archive or ticket to #1664's successor). |
| Minor | REAL | tests | The rendered in-sync path's `ensureMode` call site is untested: `Deploy` has two (`c.Render` false branch and the rendered compare branch); `TestDeploy_InSyncContentStillGetsItsDeclaredMode` exercises only the non-rendered one (`Render: false`, nil renderer). Both share the helper, so the exposure is wiring, not logic. | Code read of `cli/internal/deploy/deploy.go` (two `return ensureMode(...)` sites). | UNTESTED for the rendered path. | tests |
| Minor | REAL | hygiene | `cli/internal/fsmode/fsmode.go` is still not gofmt-clean (trailing blank line at EOF) — round-1 finding #7, ticketed and still open, confirmed by `gofmt -l`. | `gofmt -l cli/internal/fsmode/` → `fsmode.go`; `gofmt -d` shows one blank-line removal. | UNTESTED (gofmt is the check). | code (trivial) — already ticketed under #1664. |
| Minor | THEORETICAL | symlink handling | The new in-sync mode fix follows a symlinked destination (`os.Stat` in `Needs`, `os.Chmod` and `SetNamedSecurityInfo` in `Apply` all follow), where the old in-sync path did nothing and the out-of-sync path renamed over the link. Impact is bounded by same-user rights (chmod/SetACL as non-owner fails and the error surfaces), so it is surprising behavior, not a privilege boundary break. | Code read; no repro. | UNTESTED. | code (an `os.Lstat` guard in `ensureMode`) or accept with a comment. |
| Question / assumption | — | POSIX semantics | On POSIX, `Needs` compares full permission bits, so a deploy silently converges a file the operator tightened back UP to the declared mode and reports `mode fixed` (declared 0644, operator set 0600 → widened). That is the manifest's declared-wins doctrine working, but it is silent widening on the primary platform. | Code read of `fsmode_other.go` `permDiffers`. | UNTESTED. | Already ticketed as #1664 (round-2 F3). Needs a human "this is intended" or a doctor row — confirm, don't leave implicit. |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All five ACs verified against named tests and reproduced commands; gaps are the f4 verifier strength and one untested wiring path, not observed defects. |
| Verification       | B | Verifiers, tests, lint, vet and the mutation claim all reproduced this session; AC4 rests on a box transcript and the one suite failure is pre-existing and out of scope. |
| Scope              | A | The diff matches the proposal exactly; the only post-landing commits are the review rounds' own contract corrections. |
| Reliability        | B | Errors are surfaced with path context, the fix is idempotent (second run in sync, proven by test); POSIX convergence-both-ways is ticketed, not hidden. |
| Maintainability    | B | One small, single-purpose package, short functions, comments explain WHY; the outstanding gofmt nit is ticketed. |
| Handoff-readiness  | A | Two review rounds' dispositions recorded in `verification.md`, promotion candidates assessed with reasons, round-3 task pending in `tasks.md`. |

### Verdict
PASS WITH GAPS

No Blocker. The one open Major against the change is THEORETICAL (SE_DACL_PROTECTED-only check, ticketed #1663) and cannot move the verdict below PASS WITH GAPS; the REAL Major is pre-existing at base `dab595e` and outside this change's scope, so it is tracked, not gating. Rubric has no C and no D.

### Recommended next steps

Routed by set — on a PASS WITH GAPS the contract set (`proposal.md`, `tasks.md`, `features.json`) is closed; these go to the implementer's round-3 dispositions in `verification.md` (applied / ticketed / declined with a reason), not into contract edits:

- **Ticket (new)** the `TestMergeAgainstTheRealDeployedSettings` failure: `MergeHooks` drops one foreign hook on a real deployed settings file; fails identically at base `dab595e`; include the reproduction commands. This is the only finding with live user impact.
- **Disposition** the f4 verifier gap: either strengthen f4 post-archive (assert the token user's ACE and the 0644 neighbour) under #1664's successor, or record a decline noting AC1's unit test covers the two-ACE property on Windows.
- **Disposition** the rendered-path `ensureMode` gap: a one-test addition to `deploy_mode_test.go` (render a file, leave content equal, force mode drift) — apply or ticket.
- **Confirm** the POSIX converge-both-ways semantics as intended (it is already ticketed as #1664/F3); silence there leaves a silent-widening behavior on the primary platform implicit.
- #1663 (SE_DACL_PROTECTED) and the gofmt nit under #1664 are already tracked; nothing new owed.

`dotf spec archive` is **advisable** in the current state: the review is fresh against `1da0db0`, the contract files are unmodified in the working tree, and every open gap is either ticketed or disposition-ready without a contract change.
