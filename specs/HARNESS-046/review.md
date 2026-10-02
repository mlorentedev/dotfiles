---
spec: "HARNESS-046"
verdict: "FAIL"
reviewed_sha: "ee5eae5ef2bbaf8558006076a60e4d16781d6527"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-10-01"
---

## Adversarial review

**Scope**: `git diff d49e6f87357483cb5cc3b87ca5a0b400527ceb4e...HEAD`
**Sources**: `specs/HARNESS-046/{proposal,tasks,verification}.md`

### Spec and task alignment

- **AC1-AC3**: Render and `dotf doctor` checks pass (the generator uses the newer `dotf harness` implementation, resolving earlier text limits).
- **AC4**: The consistency guard was written but is not executed by any workflow.
- **AC5**: `hermes-nan` record correctly points to its state without duplication.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| Blocker | REAL | CI | `check-roster-consistency.py` is not wired into any CI workflow (bats or Actions). It is isolated in the spec directory, meaning "the next drift is caught by a check" remains unfulfilled because the check never runs automatically. | `grep -rn "check-roster-consistency.py" .github/ tests/ scripts/` returns empty. | UNTESTED | tests + code (move script and invoke in bats or CI) |
| Blocker | REAL | code | `check-roster-consistency.py` checks strict list equality (`roster[name] != skills`), causing false failures if skills are just reordered without semantic drift. | Fails on `HEAD` for `curator`: `['vault-doctor', 'crystallize', ...]` vs `['crystallize', 'handoff', ...]`. | UNTESTED | code (use set equality) |
| Major | REAL | system prompt | The `AGENTS.md` payload has grown to 25309 bytes, breaching Antigravity's payload limit. The engine is silently truncating the file, causing agents to lose doctrine. | The system prompt for this session ends with `<truncated 1469 bytes>`. | UNTESTED | code / vault (enforce budget or paginate) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| Correctness | D | Drift guard crashes on valid list reordering, falsely reporting failure. |
| Verification | C | AC4 claimed the drift check would catch future drift, but an unwired script verifies nothing automatically. |
| Scope | B | Diff matches proposal; no significant creep found. |
| Reliability | D | The drift guard is brittle (list ordering) and the payload size limit truncation silently breaks agent behavior. |
| Maintainability | B | Personas match `curator` shape; `dotf harness` delegation improved the architecture cleanly. |
| Handoff-readiness | A | Spec docs accurately track out-of-scope gaps and decisions. |

### Verdict

FAIL

### Recommended next steps

- Move `check-roster-consistency.py` to `tests/lib/` or `scripts/` and wire it into a `bats` test so CI enforces it.
- Change `roster[name] != skills` to `set(roster[name]) != set(skills)` so the check is robust to ordering.
- Address the `AGENTS.md` character limit overflow so rules are not truncated by the platform.
