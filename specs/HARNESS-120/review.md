---
spec: "HARNESS-120"
verdict: "FAIL"
reviewed_sha: "7b105e1fb798e16611a79039aab9c786fd0602c6"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-09-06"
---

## Adversarial review

**Scope**: HARNESS-120
**Sources**: `specs/HARNESS-120/{proposal,tasks,verification}.md`, `git diff 39754ec80efecd772570d9c2c2262dd738a3b7d2...HEAD`

### Spec and task alignment
- `tasks.md` accurately matches the implemented diff, separating pure harness functions from command wiring.
- `verification.md` meticulously covers AC1–AC6 with reproducible CLI outputs and explicitly names the tests that exercise each path.
- **AC7 is explicitly not met**. The implementation lacks proof of end-to-end dispatch against a live model because the local machine lacks an identity to pass the fail-closed gate.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker  | REAL    | Verification | AC7 (end-to-end real dispatch) is unverified. The contract guarantees the persona reaches a live model, but the gate blocked it and no real run was supplied. Archiving with an unmet AC violates the SDD contract. | `verification.md` section AC7 explicitly declares it "NOT SATISFIED. Blocked, and stated rather than substituted for." | UNTESTED | `verification.md` (add evidence) or `proposal.md` (waive AC7) |
| Minor    | SPECULATIVE | UX / Error reporting | Dictated `--tier` validation bypasses the helpful `ResolveTierForPersona` error. If an invalid tier is supplied, it skips the check that lists valid map tiers, instead failing later in `ResolveChain` with a potentially less descriptive error. | Code reading: `agent_auto.go:492` returns the dictated tier directly without validating it against `declaredChainTiers(m)`. | UNTESTED | code |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Pure functions and logic are highly correct, but lack of E2E verification leaves a gap in confidence. |
| Verification       | C | AC1-AC6 have strong, reproducible evidence, but explicitly skipping AC7 (the E2E proof) is a major gap. |
| Scope              | A | Diff exactly matches the proposed architecture; cleanly extracted shared dispatch setup without scope creep. |
| Reliability        | A | Employs fail-closed design robustly; `splitRecord` safely handles CRLF and BOM edge cases. |
| Maintainability    | A | Logic is cleanly separated into pure functions, well-documented, and heavily tested. |
| Handoff-readiness  | B | Spec and task files are updated, but blocked from archive due to the explicit AC7 gap. |

### Verdict
FAIL

### Recommended next steps
- Run the provided `HARNESS-120-f7` feature command on a machine with a valid `machine.json` identity to satisfy AC7, and add the output to `verification.md`.
- Alternatively, if testing against a live pool is impossible or deferred by design, update `proposal.md` to explicitly declare `review: waived` for AC7 or modify the AC.
- (Optional) Consider adding validation in `resolvePersonaForTask` when `--tier` is dictated so it fails with the same descriptive error as `ResolveTierForPersona`.

`dotf spec archive` / `/spec archive` is **NOT advisable** in the current state until the Blocker is addressed via either a contract edit or adding the missing verification evidence, followed by a re-review.
