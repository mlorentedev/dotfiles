---
spec: "BUG-093-gate-f-fail-closed"
verdict: "FAIL"
reviewed_sha: "5101938aa0013cb2105acedb706f2d02754580cb"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-09-05"
---
## Adversarial review

**Scope**: BUG-093-gate-f-fail-closed
**Sources**: specs/BUG-093-gate-f-fail-closed/{proposal,tasks,verification}.md, git diff main...HEAD

### Spec and task alignment
- AC1-AC4: Addressed. The platform boundary is respected and well-tested in both directions.
- AC5b: TOCTOU check implemented, but its placement leaves a significant race window open (see finding 1).
- AC6-AC8: Addressed. Verification commands and mutation tests are robust and successfully run.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | concurrency | The TOCTOU re-check in `reapSingleWorktree` is placed before expensive external shell-outs, leaving a wide race window. | Code read: `hostProcessInside` is checked first, followed by `isDirty` and `isMerged` (which run slow `git` commands), before the actual deletion. A process arriving during the shell-outs will be missed. | UNTESTED (Mock runner hides the timing gap) | code |
| Minor | SPECULATIVE | error handling | `filepath.Abs` failure inside `isHostProcessInside` fails OPEN (`Inside: false`), violating AC2 ("every failure path answers true"). | Code read: `sweep_proc_linux.go` returns `Inside: false` on error. While unreachable via `gateF` (which passes an absolute path), the function is package-visible and should fail closed natively. | UNTESTED (Declared as surviving in `verification.md`) | code |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Criteria mostly met, but the TOCTOU window implementation leaves a significant gap and AC2 is technically violated on unreachable code. |
| Verification       | A | Exemplary mutation testing and clear, reproducible verification evidence. |
| Scope              | A | Diff matches proposal perfectly; no scope creep. |
| Reliability        | B | The TOCTOU window across `git` shell-outs reduces the effectiveness of the late lock check. |
| Maintainability    | A | Clear structure, excellent comments explaining "why", and cleanly isolated platform logic. |
| Handoff-readiness  | A | Spec is fully updated, lessons documented, and open issues (#1470, #1523, #1530) are cleanly filed. |

### Verdict
**FAIL**


`dotf spec archive` is **NOT advisable** in the current state due to the Major finding.


### Recommended next steps (before archive)
- Move the `hostProcessInside(absWT).Inside` check in `reapSingleWorktree` to immediately precede `executeWorktreeReap` to minimize the TOCTOU window against external git processes.
- Change the `filepath.Abs` error return in `isHostProcessInside` to `Inside: true` to fully satisfy AC2, even if currently unreachable.
