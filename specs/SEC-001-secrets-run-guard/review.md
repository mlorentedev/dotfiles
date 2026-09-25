---
spec: "SEC-001-secrets-run-guard"
verdict: "FAIL"
reviewed_sha: "746683376d9f9af8d31e816bcc6292fe9f914298"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-09-24"
---

## Adversarial review

**Scope**: SEC-001-secrets-run-guard
**Sources**: specs/SEC-001-secrets-run-guard/{proposal,tasks,verification}.md + git diff 0105662daf0c04c77f43adc9b02f588604b148c6...HEAD

### Spec and task alignment
- The implementation fulfills the updated acceptance criteria but leaves critical gaps in the option parsing logic for `sh -c` wrappers, directly violating AC3 and the core ADR-028 doctrine by allowing two deterministic bypasses to dump secrets.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker  | REAL    | parsing | Zsh option bundling bypasses the guard. `shellCommandString` assumes `-o` always consumes the next `argv` element. However, `zsh` allows bundling the argument directly (e.g. `-ovi`). This causes the guard to incorrectly skip the actual `-c` option that follows, treating it as the option argument. `cFlag` remains false, and `env` is executed uninspected. | `zsh -ovi -c env` successfully dumps the environment | UNTESTED | code + tests |
| Blocker  | REAL    | parsing | Bash empty `+` option bypasses the guard. The parser handles `-` and `--` as end-of-options, but forgets `+`. Bash parses `+` as a valid (but empty) option block that turns off no flags. Because `len("+") == 1`, `shellCommandString` falls to the `default` case and returns `+` as the command string. Bash skips `+` and takes the next argument `env` as the actual `-c` snippet, executing it uninspected. | `bash -c + env` successfully dumps the environment | UNTESTED | code + tests |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | D | Defects present: two REAL bypasses allow dumping the environment. |
| Verification       | B | Evidence covers criteria and tests are reproducible, but misses these shell edge cases. |
| Scope              | B | Diff matches proposal exactly; no scope creep. |
| Reliability        | B | Most error paths handled; guard fails closed where intended. |
| Maintainability    | B | Clear naming, small functions, and well-commented parser rules. |
| Handoff-readiness  | B | Spec updates included and previous lessons captured. |

### Verdict
FAIL

### Recommended next steps
- Fix the `+` option bypass in `shellCommandString` to treat `+` as an end-of-options or empty option, just like `-` and `--`.
- Fix the `zsh` `-o` bundling parsing. Since different shells handle option arguments differently, relying on a fixed `i += 1` for `-o` is fragile. Consider failing closed on unparseable/ambiguous option clusters or accurately simulating zsh bundling rules.
- Add named regression tests for `zsh -ovi -c env` and `bash -c + env` to `TestAssertSafeChildCommand`.
- Archiving is **not advisable** until these Blockers are addressed in the code and a new review round PASSES.
