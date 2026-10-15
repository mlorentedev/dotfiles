---
spec: "CI-012-dedupe-ci-tests"
verdict: "PASS"
reviewed_sha: "88675c83cea00409b3f57ac567de654f366d71ed"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-10-14"
---

## Adversarial review

**Scope**: CI-012-dedupe-ci-tests
**Sources**: `specs/CI-012-dedupe-ci-tests/{proposal,tasks,verification}.md`, `git diff ed7e1fad08e878c6818c72eb8afae0e032013092...HEAD` (PRs #2064, #2500, #2505, commits `6e701882`, `d1a72f17`, `6d92b561`, `a8d5b821`, `88675c83`)

### Spec and task alignment

The implementation spans three landed PRs corresponding to rows N4/N5/N7 (#2064), N3 (#2500), and N6 (#2505) of issue #2059. Deferred rows (N1, N2, N8) remain tracked on #2059 behind their respective blockers. All acceptance criteria (AC1–AC6) have corresponding regression tests and feature entries in `features.json`.

All implementation tasks in `tasks.md` are marked complete with verified git evidence. The setup rows in `tasks.md` remain unticked per recorded convention (work landed on three distinct feature branches rather than a single monolithic branch).

### Adversarial verification summary

1. **AC1 & AC2 (dotf-bin helper & fail-open elimination)**:
   - Verified with `bats tests/dotf-bin-helper.bats` (14/14 pass).
   - Validated that `DOTF_BIN` invalid/non-executable paths fail immediately and never silently fall back to building.
   - Tested fail-open guard: no bats test builds `dotf` outside the helper, eliminating the BUG-055 / #807 skip-on-compile-failure class.

2. **AC3 (Go twins & deleted agent run cases)**:
   - Verified with `go test -v ./internal/cmd/ -run 'TestAgentRun'` (13/13 subtests pass) and `go test -v ./cmd/dotf/ -run 'TestRunExitsWithTheCodeAnExhaustedChainCarries'`.
   - Executed mutation testing: mutated `main.go` exit code mapping from `cmd.ExitCode(err)` to `return 1`; `TestRunExitsWithTheCodeAnExhaustedChainCarries` immediately turned RED (`exit code = 1, want 3`), confirming true coverage. Reverted cleanly.
   - The two kept cases in `tests/dotf-agent-run.bats` (dispatch pipe reading and stderr refusal) independently pass (`bats tests/dotf-agent-run.bats`).

3. **AC4 & AC5 (JUnit reporting & read-only fixture templates)**:
   - Verified that `ci.yml` wires `run-bats.sh --report-dir "$RUNNER_TEMP/bats-report"` and uploads `bats-junit` artifact under `!cancelled()`.
   - Verified live GitHub Actions artifact pagination: `gh api --paginate 'repos/mlorentedev/dotfiles/actions/artifacts?name=bats-junit&per_page=100'` returns live artifacts on `main`.
   - Verified fixture template isolation via `bats tests/fixture-template-helper.bats` (8/8 pass).
   - Verified all 5 fixture-heavy test suites (`compile-harness.bats`, `spec-gate-archive.bats`, `check-knowledge-gate.bats`, `check-spec-gate.bats`, `precommit-fallback.bats`): all 221 tests pass without leakage or write permission issues.

4. **AC6 (Static assertion deduplication)**:
   - Verified with `bats tests/duplicate-assertions.bats` (16/16 pass) and `python3 tests/lib/check-duplicate-assertions.py .` (scanned 839 static assertions in 166 bats files, 0 duplicates).
   - Executed mutation testing: introduced an artificial duplicate assertion into `tests/aliases.bats`; `check-duplicate-assertions.py` failed with exit code 1, correctly reporting the duplicate line and test names. Reverted cleanly.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | fixture-template | `_fixture_template_dir` and `fixture_template_release` assume `BATS_FILE_TMPDIR` is populated; if invoked bare without the variable set, path evaluates to `/template-$1` | `tests/lib/fixture-template.bash:31,65` | UNTESTED (tested only within Bats context) | code (non-contract: add parameter expansion fallback `${BATS_FILE_TMPDIR:?...}`) |
| Minor | THEORETICAL | check-duplicate-assertions | Scanner only scans top-level `tests/*.bats` via `os.listdir`, ignoring nested directories | `tests/lib/check-duplicate-assertions.py:192` | UNTESTED | code (tracked on #2059) |
| Minor | REAL | code-quality | `scan` (60 lines) and `grep_args` (54 lines) exceed the 40-line function guideline with high cyclomatic complexity | `tests/lib/check-duplicate-assertions.py:131,187` | `tests/duplicate-assertions.bats` | code (tracked on #2059) |
| Minor | REAL | ci-config | `ci.yml` bats timing upload uses `if-no-files-found: warn` rather than `error`, allowing a missing report to pass silently | `.github/workflows/ci.yml:384` | `tests/dotf-bin-helper.bats` | code (tracked on #2059) |
| Minor | REAL | spec-drift | `proposal.md` N4 prose mentions five files using the helper, while two were replaced by Go golden tests in #2070 | `tests/dotf-bin-helper.bats:192` vs `proposal.md:29` | `tests/dotf-bin-helper.bats` | verification.md (accounted for; contract set frozen) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | A | All ACs verified; Go twins proven by mutation; no lost test coverage from deletions. |
| Verification | A | Comprehensive evidence spanning Go unit tests, Bats helper tests, live API queries, and mutation checks. |
| Scope | A | Strictly scoped to CI test deduplication rows; deferred rows tracked cleanly on #2059. |
| Reliability | A | Read-only fixture templates, fail-closed CI guards, and hermetic helper test isolation. |
| Maintainability | B | Excellent test architecture; Python scanner function lengths and complexity tracked for follow-up. |
| Handoff-readiness | A | Lessons 343 & 430 captured; count drifts and minor open items documented in verification.md. |

### Verdict

PASS

### Recommended next steps

The contract set (`proposal.md`, `tasks.md`, `features.json`) is closed and valid. Do not modify contract files as that would invalidate this verdict.

For post-archive follow-up (outside the contract set):
1. **Tracked on #2059**: Address Minor findings (hardening `if-no-files-found: error` in `ci.yml`, recursive `.bats` discovery, and refactoring Python helper function length/complexity).
2. **Archive command**: `dotf spec archive CI-012-dedupe-ci-tests` is **advisable** in the current state.
