---
spec: "HARNESS-061-effectiveness-probes"
verdict: "PASS"
reviewed_sha: "4d153a0aacfd6c6e6c9d37ad5c1203ce10a380b8"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-18"
---

## Adversarial review

**Scope**: HARNESS-061-effectiveness-probes — hook effectiveness probes, stub pairing, DR readiness checks.
**Sources**: `specs/HARNESS-061-effectiveness-probes/{proposal,tasks,verification}.md`, `cli/internal/doctor/{hookprobe.go,checks_guard.go,checks_vault_hooks.go,checks_dr.go,fs.go,config.go}`, `tests/stub-real-pairing.bats`, `tests/precommit-fallback-real.bats`, `tests/precommit-fallback.bats`, plus commits `b412597` (#853), `e5a2ecc` (#949), `8053396` (#1006), `4d153a0` across the `feat/harness-061-archive` branch and `main`.

### Spec and task alignment

- All 10 implementation tasks are ticked `[x]`. The one `[n/a]` (branch) is honestly recorded as never existed, consistent with the spec's thesis about representation-over-behaviour.
- All 7 acceptance criteria are verifiably addressed by named tests.
- All 3 verification items (`go test`, `bats`, `shellcheck`) are confirmed green.
- The re-verification section (2026-08-18) records and cross-checks 5 mutations plus the 3rd-copy defect, with evidence of having found and fixed a real gap.
- No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain in any spec artifact file (the transcript JSONL is a machine log, not a spec artifact).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|-------------|
| Minor | THEORETICAL | scope | AC1's "the affected repo named" is underspecified — the probe loop in `guardProbeRepos` covers exactly 2 repos (DOTFILES_REPO_DIR, VAULT_PATH). A repo-local override in any other repo on the machine causes the guard to be inert there, but `checkGuardHooks`'s probe loop will not report it. The proposal's AC1 text says "a repo-local override is detected and the affected repo named" without qualifying coverage scope. The code documents the scoping in a comment, but the spec does not. | Code read of `guardProbeRepos` in `checks_guard.go:100-112` — the filter is `isDir(filepath.Join(r, ".git"))` for exactly 2 repos from env vars. | UNTESTED — no test asserts that a third repo with a local override is silently skipped, because the decision is by design. | spec (proposal.md — clarify AC1's coverage scope) |
| Minor | THEORETICAL | consistency | `guardProbeRepos` uses `isDir(filepath.Join(r, ".git"))` to determine if a path is a git checkout, while `checkVaultHooks` uses `isGitCheckout(sys, vault)` which asks git directly. For a linked worktree (where `.git` is a `gitdir:` pointer FILE, not a directory), `isDir` returns false — so if VAULT_PATH is a linked worktree, the guard probe silently skips it while the vault hooks check (correctly) detects it. The guard's effectiveness in the vault worktree goes unverified. | `checks_guard.go:103` (`!isDir(filepath.Join(r, ".git"))`) vs `checks_vault_hooks.go:39` (`!isGitCheckout(sys, vault)`). The `isGitCheckout` docstring at `hookprobe.go:35-39` explicitly names this trap. | UNTESTED — no test exercises a linked worktree through `guardProbeRepos`. | code (align `guardProbeRepos`'s checkout detection to `isGitCheckout`) |
| Minor | THEORETICAL | tests | `hookForStage`'s green direction (executable hook resolves to the path) is not independently asserted. The green path is exercised implicitly through `TestGuardHooks_NoOverrideGuardFires_MustPass` (which creates a dispatcher with executable hooks), but no standalone test asserts that `hookForStage` returns a non-empty path for an executable hook. | Code read of `hookprobe_test.go` — `TestHookForStage_NonExecutableIsNotAHook` covers the red direction only. | UNTESTED — no named test for the green executable case. | tests (add `TestHookForStage_ExecutableIsAHook`) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | A | All acceptance criteria verified with red-direction tests, mutations confirmed independently. |
| Verification | A | Evidence proves each criterion with reproducible commands, mutation battery documented and independently re-confirmed. |
| Scope | B | Minor: work landed across several PRs rather than one branch; the current branch is only spec artifacts. The scope is honestly documented and the diff is coherent. |
| Reliability | A | Error paths handled (nil seam, unreadable escrow, stat errors, missing pre-commit tool, install failure). Idempotent --fix. |
| Maintainability | A | Clear naming, well-commented business logic, the WHY of every function is documented. `isGitCheckout`'s docstring narrates the bug it prevents. |
| Handoff-readiness | A | Spec updates included, mutation battery documented, re-verification evidence, the third-copy defect found and fixed in-session. |

### Verdict

**PASS (adversarial)**

No blockers or majors. Two Minor findings, both THEORETICAL and UNTESTED — neither moves the verdict below PASS when weighted by severity × reality. The rubric is all B or above (Correctness A, Verification A, Scope B, Reliability A, Maintainability A, Handoff-readiness A).

### Recommended next steps (before archive)

1. **Minor — clarify AC1 scope in proposal.md**: Add a sentence to AC1 stating "the check probes the repos that matter by effect (the dotfiles repo and the vault, where the guard directly protects the single sink). A repo-local override in a non-probed repo is not detected by this check, which is a deliberate scoping decision." This closes the spec-vs-code gap without changing code.

2. **Minor — align `guardProbeRepos` to `isGitCheckout`**: Replace `isDir(filepath.Join(r, ".git"))` with `isGitCheckout(sys, r)` in `guardProbeRepos`. This is a one-line change that makes the probe loop consistent with `checkVaultHooks` and handles linked worktrees correctly. Add a test for the worktree case.

3. **Minor — add green-direction test for `hookForStage`**: Add `TestHookForStage_ExecutableIsAHook` that asserts an executable hook file resolves to its path. This is a one-function test, straightforward.

Neither (1) nor (2) nor (3) is a blocker. `dotf spec archive` is **advisable** in the current state — the two Minor findings are THEORETICAL and UNTESTED, but neither affects the actual correctness of the checks on the target machine's standard configuration. The core logic (effective hooksPath, dispatcher detection, secret gate, DR escrow, pairing guard) is verified with red-direction tests and confirmed by independent mutation.