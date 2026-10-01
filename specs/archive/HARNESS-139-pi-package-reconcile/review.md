---
spec: "HARNESS-139-pi-package-reconcile"
verdict: "PASS WITH GAPS"
reviewed_sha: "33b96955f592ac0bf4dfd62bb76e62a2bdca7356"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-09-30"
---
## Adversarial review

**Scope**: `d9ecf7a6416cc74e6609c33fdeb77a1436d8285d...HEAD`
**Sources**: `specs/HARNESS-139-pi-package-reconcile/{proposal,tasks,verification}.md`, `features.json`

### Spec and task alignment
- AC1-AC8 are fully implemented and traced to named tests or explicit manual verification artifacts.
- The `retire` logic successfully archives old tool data (AC5) without destruction.
- The `check` and `apply` commands follow the `forge protection` standard (GUARD-017) and gracefully degrade when `pi` or `npm` are absent.
- Spec deviation: The first live `apply` ran from `setup-linux.sh` without the pre-announcement to peers mandated by the "Risks" section. This was acknowledged by the author in `verification.md` but is recorded below as a process gap.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major    | THEORETICAL | Correctness | `Identity(source)` strips versions using `@`, but ignores `#` (used for git/npm branches like `npm:github:owner/repo#v1`). A version bump on such a package changes its identity, causing an unnecessary remove/install cycle and bypassing duplicate validation. | Code read of `Identity()` in `packages.go` which only splits on `@`. | UNTESTED | code |
| Minor    | SPECULATIVE | Resilience | `retire()` has a TOCTOU race when determining the archive destination path. Two concurrent applies retiring the same package could collide during `os.Rename`. | Code read of `retire()` loop in `apply.go`. | UNTESTED | code |
| Minor    | SPECULATIVE | Resilience | `pi install` is executed synchronously via `exec.Command` without a timeout context. A hanging `npm install` could stall the reconcile indefinitely (though bounded by CI timeouts). | Code read of `ExecRunner()` in `apply.go`. | UNTESTED | code |
| Minor    | REAL    | Spec compliance | The first live `apply` was executed silently as part of `setup-linux.sh`, bypassing the "deploy-class" pre-announcement requirement stated in the spec. | Acknowledged in `verification.md` ("Decisions made"). | N/A | verification |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Logic handles all standard npm paths flawlessly; misses edge-case git versioning (`#`). |
| Verification       | A | Test traceability is complete; output capturing handles verbosity well. |
| Scope              | A | Diff matches proposal exactly; changes split logically across PRs without creep. |
| Reliability        | B | Graceful degradation on missing tools; minor missing timeouts and TOCTOU in archive. |
| Maintainability    | A | Code is clean, functions are short (CC < 10), replaces messy bash loops with structured Go. |
| Handoff-readiness  | A | Specs updated, `retire: memory` declared, `features.json` acts as an enforceable contract. |

### Verdict
PASS WITH GAPS

### Recommended next steps
- Update `Identity(source)` to handle `#`-based version suffixes for git/npm packages to future-proof the duplicate validation and prevent churn.
- (Optional) Use `context.WithTimeout` for the `exec.Command` in `ExecRunner` to prevent infinite hangs if the upstream package registry is unresponsive.
- `dotf spec archive` is advisable in the current state, as the gaps are tracked and the core implementation covers all active use cases perfectly.
