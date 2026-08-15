---
spec: "CLI-024-secrets-file-migrate"
verdict: "PASS"
reviewed_sha: "7ff616fb5373c837a734b37d21850f4615185d5e"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-15"
---

## Adversarial review (round 3, wording-fix confirmation)

**Scope**: CLI-024-secrets-file-migrate / PR #965, commit `7ff616f`
**Sources**: specs/CLI-024-secrets-file-migrate/{proposal,tasks,verification,review}.md, features.json, PR #965 diff (commit `7ff616f`)
**Previous review**: `eecdc69`, verdict PASS (3 Minor spec-hygiene findings + 1 Minor review-artifact finding + 1 Operational PR-conflict finding)
**Fix claim**: commit `7ff616f` closes all 4 Minor findings; no code touched, wording only

### Round-2 findings — resolution status

| # | Finding (severity) | Claimed fix | Verified? | Evidence |
|---|--------------------|-------------|-----------|----------|
| 1 | `verification.md` AC3 evidence line still described the fixture as having "NO trailing newline" — factually wrong after the fixture fix (Minor/REAL) | Reworded to "multi-line fixture WITH a trailing newline, like `age -d` appends … the trailing newline is what makes the `isFile` branching regression-detectable" | **RESOLVED** | verification.md AC3 line read on disk; matches the actual fixture (secrets_migrate_test.go:105 `"...kind: Config\n" // trailing newline, like age -d`) |
| 2 | `tasks.md` AC3 task description still said "multi-line value that has NO trailing newline" (Minor/REAL) | Reworded to "HAS a trailing newline (like `age -d` appends — required so the `isFile` no-trim branch is regression-detectable, not a no-op)" | **RESOLVED** | tasks.md AC3 line read on disk; consistent with verification.md and the fixture |
| 3 | `features.json` f2 "still fails with the pre-existing #962 item-ambiguity error" and f4 "ZOHO_RECOVERY_CODE unaffected" — the same overstatement class flagged for AC2 in round 1 (Minor/REAL) | f2: "now surfaces the pre-existing #962 zoho item-ambiguity error, previously masked by the blanket file-secret guard this change removes"; f4: "still correctly resolving via age (blocked on #962, not a regression)" | **RESOLVED** | features.json f2/f4 read on disk; neither "still fails" nor "unaffected" remains; both state the surfaced #962 blocker accurately |
| 4 | Review artifact: old review.md used `verdict: "PASS WITH GAPS"` (spaces) vs the gate enum `PASS-WITH-GAPS`; also the review file was committed by the author rather than the reviewer (Minor/REAL) | review.md replaced with the round-2 review (`verdict: "PASS"`, valid enum) | **RESOLVED (by overwrite)** | review.md at HEAD carries `verdict: "PASS"`; superseded again by this round-3 file |

### New findings — this pass

| Severity | Reality | Area | Finding | Evidence | Fix location |
|----------|---------|------|---------|----------|--------------|
| Minor | REAL | Spec hygiene (verification.md) | The manual-smoke-test bullet (verification.md:23) still reads "confirmed to still fail with the pre-existing, unchanged error" — the exact overstatement construction the round-1 review flagged for AC2 ("unchanged error" overstates the before-state: pre-PR, `migrate ZOHO_RECOVERY_CODE` failed at `migrateGuard`'s blanket file-secret rejection; post-PR it fails with the #962 zoho ambiguity). Pre-existing from eecdc69, missed by `7ff616f` because the fix only touched the AC2 evidence line. Not blocking — the AC2 evidence line is correct and the bullet is a defensible short-hand — but it is the same stale-wording family this spec has now corrected four times elsewhere. | verification.md:23 (diff eecdc69..7ff616f touches only the AC3 line) | spec: reword the smoke-test bullet to "confirmed to surface the pre-existing #962 zoho ambiguity error" (one line) |
| Operational | REAL | PR mergeability | PR #965 is still `CONFLICTING` with `main` (`mergeStateStatus: DIRTY`, re-checked this session). Unchanged from round 2; blocks the merge, and archive is gated on merge. Not a code defect — the spec's code is correct at HEAD. | `gh pr view 965 --json mergeable,mergeStateStatus`: `{"mergeable":"CONFLICTING","mergeStateStatus":"DIRTY","state":"OPEN"}` | Operational: rebase `feat/secrets-file-migration` onto `origin/main` |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A  | Code untouched since round 2, which verified every `isFile` call site sound and both mutations failing. No new code to review. |
| Verification       | A  | AC3 fixture trailing-newline wording now matches the actual fixture (byte-exact test, parity abort, idempotency all regression-detectable). Live evidence captured. |
| Scope              | A  | `7ff616f` touches 4 spec files, wording only — exactly the claimed fix. No scope creep. |
| Reliability        | A  | Parity gate + mutation-tested isFile branching unchanged. |
| Maintainability    | B  | Spec text is now internally consistent (AC2 evidence, AC3 evidence, tasks, features.json all agree); one residual smoke-test bullet repeats the already-corrected "still fail … unchanged error" construction. |
| Handoff-readiness  | B  | Spec content ready for archive; PR merge conflict is the only gate left. |

### Verdict

**PASS**

All 4 round-2 Minor findings are **closed** — verified on disk, not taken on faith: verification.md and tasks.md AC3 now describe the trailing-newline fixture accurately (and match the actual fixture at `secrets_migrate_test.go:105`), and features.json f2/f4 now describe the surfaced #962 ambiguity instead of the "still fails"/"unaffected" wording. features.json parses as valid JSON (`python3 json.load`, 4 features). The commit diff (`--stat`: features.json, review.md, tasks.md, verification.md) is wording-only, no code touched — nothing else in it to flag.

One new Minor (the smoke-test bullet at verification.md:23, same "still fail … unchanged error" construction this spec already corrected four times elsewhere) and the round-2 Operational finding (PR #965 merge conflict) remain. Neither affects the code's correctness or the regression sensitivity of the tests.

### Recommended next steps (before archive)

1. **Resolve the PR merge conflict first** — `feat/secrets-file-migration` must be rebased onto `origin/main` and merged; archive is gated on merge. This is the only hard blocker.
2. **Optionally** reword the verification.md:23 smoke-test bullet ("confirmed to still fail with the pre-existing, unchanged error" → "confirmed to surface the pre-existing #962 zoho ambiguity error") in the rebase PR so the file no longer carries the stale construction. One line; not blocking.
3. Then archive: `dotf spec archive CLI-024-secrets-file-migrate` — the review is fresh (this file), the verdict is PASS, features.json is valid, and all acceptance criteria carry non-vacuous verification commands. **Advisable now only if the merge conflict is resolved first; `dotf spec archive` will otherwise precede the merge, which violates the merge-then-archive sequencing.**
