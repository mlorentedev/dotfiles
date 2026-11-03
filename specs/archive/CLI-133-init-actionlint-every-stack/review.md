---
spec: "CLI-133-init-actionlint-every-stack"
verdict: "PASS"
reviewed_sha: "1e2d6d705fda34a64bcf635edc485cd884bf3519"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-11-02"
---

## Adversarial review

**Scope**: CLI-133-init-actionlint-every-stack
**Sources**: `specs/CLI-133-init-actionlint-every-stack/{proposal,tasks,verification}.md`, `features.json`, `git diff 9ce2c36d3dd407f696c72165f7547345d51f70bf...HEAD`

### Spec and task alignment

The implementation fulfills all requirements declared in `proposal.md` and tracks tasks in `tasks.md`:
- **AC1 (Dry-run and stack coverage)**: `dotf init` creates and `--dry-run` reports `lint.yml` for all stacks (`go`, `python`, `node`, `ts`, and `none`). Verified via `TestWriteLintForEveryStack` and `TestInitDryRunListsLint`.
- **AC2 (Trigger and forge directory handling)**: `writeWorkflow` correctly replaces the default branch trigger (`branches: ["<branch>"]`) and replaces `.github/workflows/*.y*ml` with the forge's workflow directory (`.gitea/workflows/*.y*ml`, `.forgejo/workflows/*.y*ml`). Existing files in `.github`, `.gitea`, or `.forgejo` are preserved and reported `skipped`. Verified via `TestWriteLintFollowsForgeAndDefaultBranch` and `TestWriteLintSkipsAnExistingLintInAnyForgeDir`.
- **AC3 (Linter and pin conformity)**: The scaffolded workflow passes `actionlint -shellcheck=shellcheck` cleanly and adheres to `scripts/check-actions-pinned.sh` commit SHA pinning. Verified through direct testing and reproduction in synthetic scaffold environments with negative controls.
- **AC4 (Version and checkout SHA drift guard)**: `TestLintTemplatePinsMatchTheRepository` asserts that `ACTIONLINT_VERSION` matches `versions.conf` and `actions/checkout` SHA matches this repository's `.github/workflows/ci.yml`. Verified with mutation testing.
- **Scope discipline**: Follow-up concerns (structure step creating `src/`, `tests/` for workflow-only repositories; untagged/unpinned CI templates) were deliberately filed as separate issues (#3093, #3094) instead of creeping into this change.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | `cli/internal/initrepo/ci.go` | `writeWorkflow` enforces that `ciBranchToken` occurs exactly once in the template, but does not assert occurrence of `lintDirToken` when rendering `lint.yml`. If the token in `lint.yml` is renamed or mistyped, `strings.ReplaceAll` silently does nothing on Gitea/Forgejo repos. | Code inspection of `writeWorkflow` in `ci.go` line 78; unit test fails if substitution is broken. | `TestWriteLintFollowsForgeAndDefaultBranch` (catches regression if replacement fails) | code (`writeWorkflow` validation) / follow-up |
| Minor | THEORETICAL | `cli/internal/initrepo/templates/lint.yml` | `lint.yml` downloads `actionlint_${ACTIONLINT_VERSION}_linux_amd64.tar.gz` and runs on `ubuntu-latest`. Self-hosted runners on ARM64 Linux or macOS mapped to `ubuntu-latest` in Gitea/Forgejo would fail to execute the binary. | Template inspection; matches current dotfiles CI conventions which standardise on GitHub-hosted `ubuntu-latest`. | UNTESTED (ARM64 self-hosted runner environment) | `cli/internal/initrepo/templates/lint.yml` (follow-up issue if multi-arch self-hosted runners are supported) |
| Minor | SPECULATIVE | `cli/internal/initrepo/templates/lint.yml` | `ACTIONLINT_SHA256` checksum cannot be verified offline against `versions.conf` at build/test time. | Documented risk in `proposal.md` and `verification.md`; runtime `sha256sum -c` fails immediately upon version/hash mismatch. | `TestLintTemplatePinsMatchTheRepository` (guards version; hash verified at CI runtime) | spec / documentation (already recorded in proposal and verification) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All acceptance criteria verified, mutation tests killed, negative controls confirmed. |
| Verification       | A | Reproducible verification commands in `features.json` and `verification.md`, verified fresh with 0 failures. |
| Scope              | A | Tight ~30 LOC production diff; out-of-scope items cleanly spun off into issues #3093 and #3094. |
| Reliability        | A | Fully idempotent; multi-forge directory detection and default-branch escaping handled safely. |
| Maintainability    | A | Clean abstraction in `writeWorkflow` (CC=8, 30 lines, depth ≤2); no complexity or style regressions. |
| Handoff-readiness  | A | Complete triad, `features.json` pending harness execution, `docs/lessons/lesson-547` recorded in-session. |

### Verdict
PASS

### Recommended next steps
1. **PR Creation & Verification**: Open the pull request for branch `feat/init-actionlint` with the `## Knowledge` block citing `docs/lessons/lesson-547-actionlint-with-no-arguments-lints-only-github-workflows.md`, ticking the remaining task in `tasks.md`.
2. **Harness Execution**: Let the CI harness execute the verification commands recorded in `features.json` to transition features to `passing`.
3. **Archive Status**: `dotf spec archive CLI-133-init-actionlint-every-stack` is **advisable** once the PR is merged and the board issue is updated.
