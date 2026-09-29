---
spec: "SEC-006-vault-curation"
verdict: "PASS WITH GAPS"
reviewed_sha: "604a501081635e0ff30c593e779b1a4a744d8436"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-09-28"
---

## Adversarial review

**Scope**: SEC-006-vault-curation
**Sources**: `specs/SEC-006-vault-curation/{proposal,tasks,verification}.md` + `git diff c114094003855544fe1d399d4dd996064a2ae52b...HEAD`

### Spec and task alignment
- AC1-AC6 are fully implemented and verified with tests.
- Blocking guards operate as specified and fail closed.
- Passkey refusal works at the mutation boundary, applying to all operations (except safe ones like folder rename).
- The digest mechanism guarantees reviewed state matches apply state.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major    | THEORETICAL | merging | `compare` returns "unreadable" when an item has no login block, causing `mergeMismatch` to block merging two Secure Notes with "passwords unreadable". | `fieldFromItem` in `bw.go` returns `ErrBWFieldNotFound` if `it.Login == nil`, causing `compare` to see `err != nil`. | UNTESTED | code |
| Minor    | THEORETICAL | planning | `planMerge` evaluates `a == nil` before `t == nil`. If both the keeper and duplicate are absent, the row blocks ("keeper absent") instead of converging ("done(absent)"). | `switch { case a == nil: return blocked... case t == nil: return done... }` in `planMerge`. | UNTESTED | code |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Criteria met on happy path; minor negative-path gaps around merging secure notes vs logins. |
| Verification       | A | Evidence proves each criterion with reproducible commands, outputs, and exhaustive negative tests. |
| Scope              | A | Diff matches proposal exactly; other changes in diff range are merged PRs explicitly documented. |
| Reliability        | A | Error paths handled safely; TOCTOU races mitigated by digest; operations are idempotent. |
| Maintainability    | A | Code split logically, bounds checked, complexity below limits (gocyclo passed). |
| Handoff-readiness  | A | Lessons captured (lesson-312), spec closed with detailed records of review round 1. |

### Verdict
PASS WITH GAPS

### Recommended next steps
- **Code**: Fix `planMerge` to evaluate `t == nil` before `a == nil` so a merge row on two deleted items converges to `done`.
- **Code**: Modify `compare` or `mergeMismatch` so that merging two Secure Notes (items without a login block) does not erroneously block on "passwords unreadable".
