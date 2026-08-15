---
spec: "HARNESS-071-reviewer-pool"
verdict: "PASS"
reviewed_sha: "38b70da221a741f392e930a0af7a333f60da02d1"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-08-14"
---

## Adversarial review

**Scope**: HARNESS-071-reviewer-pool
**Sources**: `specs/HARNESS-071-reviewer-pool/{proposal,tasks,verification}.md`, `cli/internal/spec/review.go`, `cli/internal/spec/reviewer_pool.go`, `cli/internal/spec/review_launch.go`

### Spec and task alignment
- `reviewer-pool.json` serves as a rigid allow-list for the `reviewer:` signature.
- Check behavior differentiates between a missing pool (untracked, allows skip) and a lost pool (tracked in HEAD but missing from the tree, fails closed), fixing earlier logic flaws.
- `dotf spec review` executes models explicitly rather than falling back to defaults.
- `gitStaleness` correctly traps uncommitted modifications (staged, unstaged, and untracked contract replacements) via `git status --porcelain`.
- Test suite and mutation assertions provide strong evidence for the acceptance criteria.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | SPECULATIVE | Launcher | If the prompt string began with a hyphen, CLI argument parsers (like pflag) could misinterpret `--print -value` as an invalid flag instead of the value. Since `ReviewPrompt` always starts with "Perform", this is speculative. | code read of `ReviewerCommand` | UNTESTED | code (using `--print=` format avoids this) |
| Minor | THEORETICAL | Git staleness | Untracked files with the exact name of a contract file (e.g. replacing a deleted contract file with an untracked one) correctly flag staleness due to porcelain output length > 0, though they might theoretically misclassify the returned reason string. | code read of `gitStaleness.Stale` | `TestGitStalenessDetectsUncommittedContractChange` | — (surface only; no action needed) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All ACs correctly mapped; strict id check and uncommitted changes handling are solid. |
| Verification       | A | Mutation testing proves the guards work; coverage is rigorous. |
| Scope              | A | Diff tightly matches the proposal; no scope creep observed. |
| Reliability        | A | Safe degradation for missing `tmux`, robust POSIX shell quoting. |
| Maintainability    | A | Extremely well commented, specifically around "why" decisions. |
| Handoff-readiness  | A | Spec is complete, lessons documented, and ready for archive. |

### Verdict
PASS

### Recommended next steps (before archive)
- You may proceed with `dotf spec archive`.
