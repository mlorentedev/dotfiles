---
spec: "SEC-001-secrets-run-guard"
verdict: "FAIL"
reviewed_sha: "d8f8b115275e8a71ac6cc34f1d4fdaa98e5ea308"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-09-24"
---
## Adversarial review

**Scope**: SEC-001-secrets-run-guard
**Sources**: `specs/SEC-001-secrets-run-guard/{proposal,tasks,verification}.md`, `git diff 0105662daf0c04c77f43adc9b02f588604b148c6...HEAD`

### Spec and task alignment
- Acceptance criteria 1-10 are marked complete and align with the tests and implementation artifacts.
- Shell wrapper introspection (AC3) is implemented but misses alternative shell flag patterns (`+c`), standard builtins (`typeset`), and flag interleaving (`bash -c -- env`).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker  | REAL    | snippet introspection | Interleaving flags or `--` between `-c` and the snippet bypasses the guard. `isCFlag` assumes the snippet is always exactly `argv[i+1]`. `bash` allows `bash -c -- env` or `bash -c -i env`. The guard inspects `--` or `-i` as the snippet and ignores the actual snippet (`env`). | `dotf secrets run -- bash -c -- env` and `dotf secrets run -- bash -c -i env` dump the decrypted environment variables to stdout. | UNTESTED | code + tests |
| Blocker  | REAL    | snippet introspection | `sh +c env`, `bash +c env`, and `zsh +c env` execute the command string and dump the environment, bypassing the snippet guard entirely. `isCFlag` rigidly checks for a `-` prefix, so the `+c` argument is ignored and the inner command is never inspected. | `dotf secrets run -- bash +c env` dumps the decrypted environment variables to stdout. | UNTESTED | code + tests |
| Major    | REAL    | introspection commands | `typeset` is a standard builtin in `bash`, `zsh`, and `ksh` that prints environment variables (identically to `declare` or `set`), but it is missing from `introspectionWords`. | `dotf secrets run -- bash -c typeset` bypasses the guard and dumps the environment. | UNTESTED | code + tests |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | The guard catches `-c` but misses `+c`, flag interleaving, and `typeset`, leaving substantial negative-path gaps. |
| Verification       | A | Verification artifacts are complete and runnable, though test cases missed edge inputs. |
| Scope              | A | Diff closely matches proposal with no scope creep. |
| Reliability        | B | Handles error paths well, but fails open on unexpected shell invocation flags. |
| Maintainability    | A | Code is clean, modular, and CC is well within limits. |
| Handoff-readiness  | A | Spec is updated and new lessons (288, 289) are accurately captured. |

### Verdict
FAIL

### Recommended next steps
- Fix snippet extraction to properly emulate shell flag parsing: stop at `--` or the first non-option argument instead of blindly checking `i+1`.
- Fix `isCFlag` to handle `+c` prefixes across all inspected shells (e.g. `arg == "+c" || ((strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "+")) && strings.Contains(arg, "c"))`).
- Add `typeset` to `introspectionWords`.
- Add test rows for `--`, `-i`, `+c` flags, and `typeset` commands to `TestAssertSafeChildCommand` to prove they are now correctly refused.
- `dotf spec archive` / `/spec archive` is NOT advisable in the current state due to the FAIL verdict. Fix the code and tests, then request a re-review.
