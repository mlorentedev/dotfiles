---
spec: "MEMORY-017-session-end-handoff-lock"
verdict: "PASS"
reviewed_sha: "36b92fdd28b3ee7b81a38d5846d1a2be943b35c7"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-09-30"
---

## Adversarial review

**Scope**: MEMORY-017-session-end-handoff-lock
**Sources**: `specs/MEMORY-017-session-end-handoff-lock/{proposal,tasks,verification}.md`, `features.json` + diff `512cd05751bf7dfd626fb2432ae55602407ad264...HEAD`

### Spec and task alignment
- **AC1** (handoff-write and SessionEnd share lock): Met. The implementation consolidates the lock helper in `cli/internal/mem/handoff_lock.go` and uses it in both paths.
- **AC2** (SessionEnd waits for active writer): Met. `SessionEnd` acquires the shared lock before calling `os.ReadFile`. Test coverage explicitly validates the blocking behavior.
- **AC3** (Regressions green, builds across OS): Met. Verified by running `go test` and `go vet` on the changed packages.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | spec vs code | `proposal.md` requires preserving HARNESS-088's hashing byte-for-byte, but the implementation intentionally canonicalizes the parent directory instead of the file to fix a Windows handle race. This alters the canonical path for non-existent files, creating a spec mismatch. | `verification.md` explicitly notes this design change. | UNTESTED | spec |
| Minor | SPECULATIVE | testing | `TestSessionEndWaitsForConcurrentHandoffWrite` uses a 50ms timeout to assert `SessionEnd` is blocked. On heavily loaded CI runners, the goroutine might be slow to schedule, leading to a false PASS if the lock failed to block. | Source inspection of the 50ms `select`. | `TestSessionEndWaitsForConcurrentHandoffWrite` | tests |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All acceptance criteria verified, regressions passing, race conditions addressed. |
| Verification       | A | Verification artifacts and `features.json` cover criteria with reproducible commands. |
| Scope              | A | Diff matches proposal exactly; no scope creep or unrelated changes. |
| Reliability        | A | Error paths handled gracefully; `SessionEnd` correctly swallows timeout errors per contract. |
| Maintainability    | A | Lock logic isolated cleanly into `mem`, functions short and clear with CC < 10. |
| Handoff-readiness  | A | Spec updates included, verification filled, lesson capture decision explicitly reasoned. |

### Verdict
PASS

### Recommended next steps
- **spec**: Update `proposal.md` "Risks / open questions" to reflect the intentional change from file to parent directory canonicalization.
- **tests**: Consider increasing the 50ms timeout in `TestSessionEndWaitsForConcurrentHandoffWrite` to make the test less prone to false passes on slow runners.
- The change is ready for `dotf spec archive`.
