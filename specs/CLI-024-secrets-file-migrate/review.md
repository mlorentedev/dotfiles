---
spec: "CLI-024-secrets-file-migrate"
verdict: "PASS"
reviewed_sha: "eecdc69e775c3174019b78d2eee8cc6f26a90d7b"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-15"
---

## Adversarial review (re-review)

**Scope**: CLI-024-secrets-file-migrate / PR #965, commit eecdc69
**Sources**: specs/CLI-024-secrets-file-migrate/{proposal,tasks,verification,review}.md, features.json, PR #965 diff (commit eecdc69), supporting source files
**Previous review**: `d284b83`, verdict PASS WITH GAPS (one Major, two Minor findings)
**Fix claim**: commit eecdc69 claims to resolve all three findings

### Previous findings — resolution status

| # | Original finding (severity) | Claimed fix | Verified? | Evidence |
|---|-----------------------------|-------------|-----------|----------|
| 1 | `TestSecretsMigrate_FileSecret_ByteExact` fixture had NO trailing newline, making `strings.TrimRight` a no-op regardless of `isFile` threading — a regression in either `ageValue` or parity-gate `isFile` branch would pass silently. (Major/REAL) | Fixture now carries a trailing newline (`...kind: Config\n`), matching what `age -d` actually appends. | **RESOLVED** | Fixture line 105: `"apiVersion: v1\n...\nkind: Config\n"` — trailing `\n` confirmed. Two independent mutation tests proved the test now fails on both `isFile`→false regressions: (a) hardcoding `ageValue(..., false)` → trimmed value written, final assertion `fw.created != kubeconfig` catches it; (b) hardcoding `normalizeValue(back, false)` → parity abort, `t.Fatal(err)` catches it. Both confirmed empirically in this review session. The full suite (`go build ./... && go vet ./... && go test ./... -count=1 && golangci-lint run`) passes clean. |
| 2 | `proposal.md` frontmatter `status: draft` (Minor/REAL) | `status: verifying` | **RESOLVED** | proposal.md line 4: `status: verifying` confirmed. |
| 3 | AC2 wording said ZOHO_RECOVERY_CODE "still fails with the pre-existing #962 error" — overstates the before-state. Pre-PR, `migrate ZOHO_RECOVERY_CODE` failed at `migrateGuard` with a blanket file-secret rejection, not the zoho ambiguity. The correct behavior is that removing the guard surfaces the real #962 blocker. (Minor/REAL) | Reworded in proposal.md, tasks.md, verification.md to say "now surfacing the pre-existing zoho ambiguity, previously masked by the blanket file-secret guard this PR removes". | **RESOLVED** | proposal.md AC2: "...now surfacing the pre-existing `zoho` item-name ambiguity from #962 (previously masked by the blanket file-secret guard this PR removes)". tasks.md: "...now surfaces the pre-existing `zoho` item-ambiguity error (#962), previously masked by the blanket file-secret guard this PR removes". verification.md: "...now reaches the `bw get item "zoho": More than one result was found` error (#962) — before this PR it failed earlier, at `migrateGuard`'s blanket file-secret rejection". All three accurate. |

### New findings — this pass

| Severity | Reality | Area | Finding | Evidence | Fix location |
|----------|---------|------|---------|----------|-------------|
| Minor | REAL | Spec hygiene (verification.md) | `verification.md` AC3 evidence line (line 12) still says "multi-line fixture with NO trailing newline" — this is now **factually wrong**. The fixture was changed to carry a trailing newline. The same file's "Adversarial review" section (line 33) correctly says "the fixture now carries a trailing newline", creating an internal contradiction. The fix commit updated the test docstring but left the AC3 evidence line stale. | verification.md:12: `- [x] AC3 ... (multi-line fixture with NO trailing newline; asserts...)` vs verification.md:33: `the fixture now carries a trailing newline`. | spec: update verification.md AC3 evidence to say "fixture with trailing newline; asserts byte-exact write preserves it" |
| Minor | REAL | Spec hygiene (tasks.md) | `tasks.md` AC3 task description (line 22) says "multi-line value that has NO trailing newline". The task description describes a fixture shape that was superseded by the fix. | tasks.md:22: `Write failing test ... with a multi-line value that has NO trailing newline` | spec: update tasks.md AC3 description to reflect the current fixture shape |
| Minor | REAL | Spec hygiene (features.json) | `features.json` f2 says "still fails with the pre-existing #962 item-ambiguity error, not a new error introduced by this change" — the same overstatement the previous review flagged for AC2. The fix commit updated AC2 in proposal/tasks/verification but not features.json. f4 similarly says "ZOHO_RECOVERY_CODE unaffected". features.json is one of the four `contractFiles` the archive gate checks for staleness. | features.json entries f2, f4 behavior strings | spec: update features.json f2/f4 behavior to match the corrected AC2 wording |
| Minor | REAL | Review artifact | The old review.md (committed in eecdc69 by the author) uses `verdict: "PASS WITH GAPS"` (with spaces) but the archive gate's `ParseReview` enum is `PASS-WITH-GAPS` (hyphenated). The old file would have been rejected by `dotf spec archive` as an unrecognized verdict. This is now moot (my new review.md overwrites it with `verdict: "PASS"`), but it reveals a procedural gap: the review file was committed without gate validation. | `cli/internal/spec/review.go:30`: `VerdictPassWithGaps Verdict = "PASS-WITH-GAPS"` vs old review.md:3: `verdict: "PASS WITH GAPS"` | Resolved by overwrite in this review |
| Operational | REAL | PR mergeability | PR #965 is `CONFLICTING` with `main` (`mergeStateStatus: DIRTY`). Two commits landed on main after the PR's base (092cb80): `a42cc67` (cherry-picked #585, creating duplicate-base conflicts) and `a6a1458` (spec review command, adding `review_launch.go` etc.). The PR needs a rebase before merge. Not a code defect — the spec's code is correct at HEAD — but the merge path is blocked. | `gh pr view 965 --json mergeable,mergeStateStatus`: `{"mergeable":"CONFLICTING","mergeStateStatus":"DIRTY"}` | Operational: rebase `feat/secrets-file-migration` onto `origin/main` |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A  | Code is correct — every path (ageValue isFile branch, parity gate normalizeValue, assertMigratable relaxation, checkBwSources field validation, unique-age guard) is soundly wired. No code defect. Both mutations confirmed to fail. |
| Verification       | A  | AC3's test now exercises the isFile branching — trailing-newline fixture makes both ageValue and parity-gate isFile branches regression-detectable (proven by mutation tests). Parity-gate abort on mismatch confirmed. Idempotency path tested. |
| Scope              | A  | Single logical change: relax guard, thread isFile, fix comment, flip registry. No scope creep. Spec additions clean. |
| Reliability        | A  | Parity gate guards write correctness. Rollback documented (keep .secret.age files, flip backend back). Idempotent. Mutation-tested. |
| Maintainability    | B  | Minimal diff. Doc comments well-rewritten. Test docstring factually correct. Two spec-hygiene textual inconsistencies (AC3 evidence in verification.md and tasks.md still describe the old fixture). |
| Handoff-readiness  | B  | Three spec-hygiene texts stale (verification.md AC3, tasks.md AC3, features.json f2/f4). PR has a merge conflict with main. The review file itself is now fresh. |

### Verdict

**PASS**

The one Major gap (AC3 fixture not exercising isFile branching) is **closed** — the trailing-newline fixture makes both `isFile` call sites regression-detectable, confirmed empirically by mutation tests in this session. The two Minor hygiene findings (proposal.md status, AC2 wording) are also resolved.

Three new Minor findings surfaced — all spec-hygiene textual inconsistencies (verification.md AC3 evidence, tasks.md AC3 description, features.json f2/f4 wording) left stale by the fix commit, and one Operational finding (PR merge conflict). None affect the code's correctness or the test's regression sensitivity. The code is correct, the full suite passes, and the verification gap is closed.

### Recommended next steps (before archive)

1. **Fix the stale AC3 evidence** in `verification.md` (line 12): change "multi-line fixture with NO trailing newline" to describe the actual fixture — e.g. "multi-line fixture with trailing newline (like `age -d`); asserts the bw-written value is byte-exact, neither trimmed nor appended".
2. **Fix the stale AC3 task description** in `tasks.md` (line 22): update the description to reflect the current fixture shape (with trailing newline).
3. **Fix features.json f2/f4 behavior wording** to match the corrected AC2 description — "now surfaces the pre-existing #962 ambiguity, previously masked by the file-secret guard" rather than "still fails with".
4. **Rebase the PR** onto `origin/main` to resolve the merge conflict.
5. After the three spec-text fixes: **`dotf spec archive CLI-024-secrets-file-migrate` is advisable** — the gate checks (no unresolved tags, fresh PASS review, non-stale, reviewer in pool) will all pass. The archive command does not check PR mergeability, so the spec can be archived independently of the PR rebase.