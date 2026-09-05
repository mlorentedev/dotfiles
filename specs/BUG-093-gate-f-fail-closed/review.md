---
spec: "BUG-093-gate-f-fail-closed"
verdict: "FAIL"
reviewed_sha: "2d9f41149b2be940282c19fa7a34e3bdc1910c58"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-09-05"
---
## Adversarial review

**Scope**: BUG-093-gate-f-fail-closed
**Sources**: specs/BUG-093-gate-f-fail-closed/{proposal,tasks,verification}.md + git diff 9e6ee7d33844c43ab567694bde09f3c5326dcec8^ 9e6ee7d33844c43ab567694bde09f3c5326dcec8

### Spec and task alignment
- AC1, AC3, AC4, AC5, AC6, AC8 are met.
- **AC2 violation**: The spec explicitly requires "every failure path answers `true`, not `false`." While `os.ReadDir("/proc")` correctly answers `true`, `os.Readlink("/proc/<pid>/cwd")` answers `false` on error. This is a deliberate but undocumented fail-open path that contradicts the spec.
- **Missing implementation tests**: The `verification.md` claims tests cover the fix, but the test suite only exercises the `hostProcessInside` seam. The actual `sweep_proc_linux.go` logic (`/proc` traversal and symlink resolution) has zero test coverage.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker | REAL | path-resolution | `isHostProcessInside` fails open on symlinked worktrees. It uses `filepath.Abs` which does not resolve symlinks, but `/proc/<pid>/cwd` returns the resolved physical path. Comparing the two fails, returning `false` (no process inside) and allowing an active worktree to be reaped. | Code inspection and shell validation (`/proc/PID/cwd` resolves symlinks natively). | UNTESTED | code (`filepath.EvalSymlinks`) + tests |
| Major | REAL | concurrency / permissions | The spec demands fail-closed on every failure path, but `processCwdInside` returns `false` (fail-open) on `os.Readlink` errors (e.g., Permission Denied). A root process inside a worktree will be ignored, and the worktree reaped. If changed to `true`, the tool becomes inert on Linux. This fundamental limitation must be documented. | Code inspection shows `return false` on `os.Readlink` error; `ls -l /proc/1/cwd` yields Permission Denied. | UNTESTED | spec + code (add comment) |
| Major | REAL | tests | The Linux `/proc` traversal implementation (`sweep_proc_linux.go`) is entirely untested. Existing tests only mock `hostProcessInside` to verify caller behavior, leaving the complex parsing logic unexercised. | `sweep_proc_test.go` uses `withHostProcessInside` seam; no `sweep_proc_linux_test.go` exists. | UNTESTED | tests |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | D | AC2 is violated (fail-open on `os.Readlink` errors) and symlinked worktrees cause silent fail-open defects. |
| Verification       | C | Evidence relies entirely on seam tests; the actual Linux `/proc` logic has zero test coverage. |
| Scope              | A | Diff matches proposal exactly; no unrelated changes or feature creep. |
| Reliability        | D | Silent fail-open on symlinked paths or unreadable processes allows active worktrees to be deleted. |
| Maintainability    | A | Clear structure, functions are short, and platform boundary is respected cleanly. |
| Handoff-readiness  | B | Spec updates are present, but the unreadable `/proc/<pid>/cwd` limitation was not documented. |

### Verdict
FAIL

### Recommended next steps (before archive)
- Fix the symlink resolution bug in `isHostProcessInside` by using `filepath.EvalSymlinks` on the target path.
- Add `sweep_proc_linux_test.go` with synthetic tests for `/proc` parsing, including symlinks.
- Update `proposal.md` and code comments to explicitly acknowledge the fail-open trade-off on unreadable `cwd` paths.
