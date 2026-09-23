---
spec: "CLI-080-secrets-reconcile"
verdict: "FAIL"
reviewed_sha: "f16fb2e768e723ddd62a53310823479adf24bf9c"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-09-22"
---

## Adversarial review

**Scope**: CLI-080-secrets-reconcile
**Sources**: `specs/CLI-080-secrets-reconcile/proposal.md`, `specs/CLI-080-secrets-reconcile/tasks.md`, `specs/CLI-080-secrets-reconcile/features.json`, `specs/CLI-080-secrets-reconcile/verification.md`, and git diff 717756bcc83b6b7c167795538be1cf6491a5763e...HEAD

### Spec and task alignment
- `reconcile` properly syncs, applies operations in dependent order, and successfully splits `GitHub` legacy items into per-purpose PATs.
- `from` validation appropriately blocks ambiguous sources or missing items/fields.
- The two-phase apply correctly protects against failing mid-way due to bad source values.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker  | REAL    | planner / cmd | First `--apply` fails its own convergence check if a new item is created and its source is flagged with `retire: true`. The planner correctly waits for the destination to exist before planning `OpRetireSource`, but the verification re-plan at the end of the first `--apply` then spots `OpRetireSource` and fails the command with "store did not converge". | verified by writing `TestReconcileBug_RetireInFirstRun` | UNTESTED | code + tests |
| Minor    | THEORETICAL | seam | `w.ResolveFolder` creates folders via `POST /object/folder` but does not invoke `syncAfterWrite`. While local caching masks this for the current run, the omission creates a small window where daemon cache could be stale for subsequent external reads. | code read | `TestBWServeWriter_ResolveFolder` | code |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | Convergence check fails on first run if `retire: true` is present for an absent destination. |
| Verification       | A | Excellent test coverage, mutation testing, and evidence from live vault runs. |
| Scope              | A | Diff matches the proposal perfectly, operations are strictly additive. |
| Reliability        | A | Safe two-phase apply, constant time compares, correctly aborts on ambiguous sources. |
| Maintainability    | A | Logic is well isolated, clear operations, and tests verify pure core logic. |
| Handoff-readiness  | A | Spec is fully updated, clear documentation of decisions. |

### Verdict
FAIL

### Recommended next steps
- Update `secrets_reconcile.go` or `reconcile.go` to handle `retire: true` gracefully on the first run, so it doesn't fail the convergence check (e.g., auto-applying remaining retires or avoiding an error return).
- Add a named regression test for the case where a new item is declared with `from: { retire: true }` and `--apply` is run, ensuring it succeeds without raising a convergence error.
- (Minor) Consider calling `syncAfterWrite` after `POST /object/folder` in `ResolveFolder` to ensure daemon cache consistency.
