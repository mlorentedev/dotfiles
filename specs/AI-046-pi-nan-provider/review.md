---
spec: "AI-046-pi-nan-provider"
verdict: "FAIL"
reviewed_sha: "79c89b438a60e699ad9f34ac20ed39a3ec620fec"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-09-28"
---
## Adversarial review

**Scope**: AI-046-pi-nan-provider
**Sources**: specs/AI-046-pi-nan-provider/{proposal,tasks,verification}.md, specs/AI-046-pi-nan-provider/features.json, git diff 030ade1e1ca5539b1adf7d38eae20ea3524fd957...HEAD

### Spec and task alignment
- `packages.json` correctly declares the pinned package.
- `ai/pi/models.json` has removed the `nan` provider definition.
- `nan-provider.json` correctly specifies `mediaMcp: false`.
- `measure-ac6.sh` accurately tests the context window overflow reasoning guard.
- However, test deletion for the reviewer pool gate introduced a regression for non-NaN providers, the media MCP bridge defaults to ON if deployed out-of-order, and the harness verification for `f5` permanently fails.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker | REAL | tests/reviewer-pool | Test deletion regression: The `reviewer-pool.bats` test verifying every `pi` pool member is reasoning-class was completely deleted, but its replacement in `pi-nan-package.bats` explicitly filters for `.provider == "nan"`. Any non-NaN pi member (e.g. `openrouter`) added to the pool will now bypass the reasoning-class check entirely. | `tests/reviewer-pool.bats` lines 28-39 deleted; `tests/pi-nan-package.bats` lines 86-92 filters by `.runner == "pi" and .provider == "nan"`. | UNTESTED | tests |
| Major | REAL | deploy / pi-nan-provider | The media MCP bridge defaults to ON in the package, mitigated only by `nan-provider.json`. `dotf pi packages apply` does NOT deploy this file; it is deployed separately via `dotf deploy`. If `pi` is run before `deploy` completes, the unpinned `npx` bridge is enabled. | `pi-nan-package.bats` manually copies the config because `install` doesn't provide it; `ai/deploy.json` runs independently of `packages apply`. | UNTESTED | code |
| Minor | REAL | features.json | Feature `f5` uses `exit 1` in its verification command (`echo 'Recorded measurement...' && exit 1`). According to the SDD contract, the harness cannot mark this feature as `passing` if it exits non-zero, permanently blocking automated archive gates that check `features.json` state. | `specs/AI-046-pi-nan-provider/features.json` line 33. | UNTESTED | spec |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Core requirements met, but the unpinned media bridge could be active if invoked out of sequence. |
| Verification       | C | Good measurements, but the `reviewer-pool` test deletion gap silently dropped coverage for non-NaN models, and `f5` verification permanently fails. |
| Scope              | B | Diff matches proposal; side-changes (like CI parallelization) are due to git base but AI-046 changes are focused. |
| Reliability        | B | Handled network cuts correctly; no crashes observed. |
| Maintainability    | C | The deleted reviewer-pool test worsened maintainability by tying a generic check to a single provider. |
| Handoff-readiness  | B | Specs and lesson recorded. |

### Verdict
FAIL

### Recommended next steps
- tests: Restore the reasoning-class check for non-NaN pi providers in `tests/reviewer-pool.bats` (or modify the check to cover all).
- code: Address the out-of-order deploy race condition for the media MCP bridge (e.g. wrapper exports `NAN_MEDIA_MCP=0` or `dotf pi packages apply` ensures `nan-provider.json` is deployed).
- spec: Change `features.json` `f5`'s verification to `exit 0` after printing the message, so the harness can mark it passing.
