---
spec: "GUARD-017-protection-as-code"
verdict: "PASS WITH GAPS"
reviewed_sha: "f9f56117f30562ea150cf51f9c7199f94f54ea80"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-10-22"
---

## Adversarial review

**Scope**: GUARD-017-protection-as-code
**Sources**: specs/GUARD-017-protection-as-code/{proposal,tasks,verification}.md + git diff 24130692a37e6a3054ee4e3f784034b3edfe32d0...HEAD

### Spec and task alignment

The spec covers branch protection as code across two PRs and a closing evidence phase:
- **PR-A (declare and detect)**: `forge/branch-protection.json` and its schema `forge/branch-protection.schema.json` declare protection per repository; `dotf forge protection check` diffs field by field; `dotf doctor` includes a `branch-protection` diagnostic check; D-1 policy block (`required_approving_review_count = 0`) is pinned by `TestProtectionApprovalsRationalePinned`.
- **PR-B (apply)**: `dotf forge protection apply` converges protection via whole-object PUT with re-read assertions and preflights newly required checks against the last 5 merged pull requests to prevent maintainer lockout.
- **Stale declaration refusal (Lesson 442)**: `CheckFresh` and `LandedDeclaration` prevent acting on a local checkout whose declaration differs from `origin` default branch.
- **AC6 live close**: Commit `f9f56117` documents live verification of `spec-gate` on `mlorentedev/dotfiles` `main`, clean `apply --dry-run` (`changed=0`), and registers `f5` and `f6` in `features.json`.

All acceptance criteria AC1 through AC6 are demonstrably satisfied. The implementation in `cli/internal/forge` is robust, well-tested, and adheres to all project quality rules.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| Minor | REAL | handoff | Promotion candidates in `verification.md` lines 85–87 retain unfilled `<...>` template placeholders. `CheckPromotions` (`cli/internal/spec/promotion.go`) checks these during `dotf spec archive` and will refuse until each candidate is answered with `yes: <path>` or `no: <reason>`. | `CheckPromotions` regex `placeholder = regexp.MustCompile("^<[^>]*>$")` rejects unfilled placeholders; `lesson-442-a-declaration-read-from-a-checkout-is-only-as-current-as-the-checkout.md` already exists to answer the lesson candidate. | `TestArchivePreflightCheckPromotions` in `cli/internal/spec/promotion_test.go` | verification (`specs/GUARD-017-protection-as-code/verification.md`) |
| Minor | REAL | spec-alignment | `proposal.md` lines 141–149 still display `- [ ]` for AC5 and AC6, whereas `tasks.md` and `verification.md` mark them complete with full passing evidence. | `proposal.md` lines 141, 145 show `- [ ]`, while `tasks.md` lines 33–35 and `verification.md` lines 54, 100 show completed verification. | Visual comparison / `dotf spec digest GUARD-017-protection-as-code` | spec (`proposal.md` / disposition in `verification.md`) |
| Minor | THEORETICAL | verification | Verification command for `features.json` entry `f6` relies on live network and authentication (`gh api repos/mlorentedev/dotfiles/branches/main/protection`), which cannot pass in offline or sandboxed CI environments. | `features.json` f6 command invokes `gh api ... && dotf forge protection apply --dry-run`. | `features.json` f6 verification command | spec (`features.json` / documented tradeoff: AC6 is explicitly defined as live owner-run) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All acceptance criteria AC1–AC6 verified live and in unit suites; negative paths and error responses handled. |
| Verification       | A | Comprehensive unit test suite with stateful fake forge, live CLI runs, and mutation tests killing altered behaviors. |
| Scope              | A | Implementation is cleanly bounded to declared branch protection and freshness checks; no unrelated modifications. |
| Reliability        | A | Bounded concurrency (6 workers), network timeouts, re-read after PUT, and unreported-context preflight avoid lockout. |
| Maintainability    | A | Functions are under 48 lines with CC <= 7, strict types, clear docstrings, and 0 `golangci-lint` issues. |
| Handoff-readiness  | B | Implementation and tests are complete; promotion candidates in `verification.md` need answers before archive preflight. |

### Verdict
PASS WITH GAPS

### Recommended next steps

The contract set (`proposal.md`, `tasks.md`, `features.json`) is closed and matches the recorded contract digests. Do not modify contract files to avoid invalidating this review. All recommendations are directed to `verification.md` (which is outside the staleness check):

1. **Answer promotion candidates in `verification.md`** (outside contract set):
   Update lines 85–87 of `specs/GUARD-017-protection-as-code/verification.md`:
   - Lesson: `yes: docs/lessons/lesson-442-a-declaration-read-from-a-checkout-is-only-as-current-as-the-checkout.md`
   - ADR: `no: decision D-1 and rationale are recorded in proposal.md and pinned by TestProtectionApprovalsRationalePinned`
   - Pattern: `no: repository-specific forge configuration; no cross-repository pattern needed`
2. **Disposition minor gaps in `verification.md`**:
   Record that AC5 and AC6 are complete per `tasks.md` and live evidence, and note that `features.json` f6 intentionally exercises live forge state.
3. **Archive readiness**:
   `dotf spec archive GUARD-017-protection-as-code` is **advisable** as soon as step 1 above is applied in `verification.md`.
