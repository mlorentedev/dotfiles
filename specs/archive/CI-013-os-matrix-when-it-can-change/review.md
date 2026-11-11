---
spec: "CI-013-os-matrix-when-it-can-change"
verdict: "PASS"
reviewed_sha: "d0ffa96fe28cd47cc60a39c54c1dcd596f744c56"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-11-11"
---

## Adversarial review

**Scope**: CI-013-os-matrix-when-it-can-change
**Sources**: `specs/CI-013-os-matrix-when-it-can-change/{proposal,tasks,verification}.md`, `features.json`, `git diff a72fd533a1dc02f136adfd343a2e5e04e9bd9319...HEAD`

### Spec and task alignment

- **AC1** (Docs-only PR starts no Windows or macOS jobs except required `test-windows`, whose steps skip): Verified. In `cli.yml` and `cli-smoke.yml`, the OS matrix dynamically evaluates to `['ubuntu-latest']` when paths-filter detects docs-only changes. `test-macos` in `ci.yml` and `from-zero` in `from-zero.yml` skip before starting a runner on docs-only diffs. `test-windows` remains unchanged as a required check with step-level skipping (governed by CI-004 until gate composite rollout). Pinned by `tests/ci-os-matrix.bats` tests 1–4.
- **AC2** (Push to `main` and tag pushes run full matrix): Verified. Expression `(github.event_name != 'pull_request' || steps.os.outputs.code == 'true')` ensures pushes and tags evaluate to `['ubuntu-latest', 'windows-latest']` without consulting paths-filter. Pinned by `tests/ci-os-matrix.bats` tests 2–3.
- **AC3** (Warm integration container build uses GHA layer cache for pre-COPY layers): Verified. `docker/setup-buildx-action` and `docker/build-push-action` configured with `cache-from: type=gha,scope=integration` and `cache-to: ${{ github.event_name == 'push' && 'type=gha,scope=integration,mode=max' || '' }}` with `load: true`. Pinned by `tests/ci-os-matrix.bats` test 6 and `tests/integration-builds-dotf.bats`.
- **AC4** (Single source of truth for docs-only exclusions): Verified. `.github/filters/code.yml` provides the SSOT, read by `cli.yml`, `cli-smoke.yml`, and `tests/ci-path-filtering.bats`. Disjointness from `ci.yml`'s `code` filter is verified by `tests/ci-os-matrix.bats` test 5.
- **Tasks & Features**: All tasks in `tasks.md` marked complete correspond to concrete commits in the diff. Features in `features.json` match behavioral gates and pass. Unchecked post-merge live verification tasks in `tasks.md` and `verification.md` are documented with rationale (current branch touches `.github/` so live documentation-only path can only be observed post-merge).
- **Knowledge hygiene**: Lesson 578 (`docs/lessons/lesson-578-a-path-filter-saves-only-what-its-diff-class-costs.md`) recorded in-flight.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| Minor | REAL | verification | Command line in `verification.md` line 21 passes `tests/ci-os-matrix.bats` twice (`git grep` already lists it), causing `bats` to plan 321 tests, execute 315, warn of unexecuted tests, and exit 1. | Running `bats $(git grep -l -E 'cli\.yml\|cli-smoke\.yml\|ci\.yml\|filters/code' -- 'tests/*.bats') tests/ci-os-matrix.bats` exits 1 with warning. Running without the duplicate argument executes 315 tests cleanly (exit 0). | `bats $(git grep -l -E 'cli\.yml\|cli-smoke\.yml\|ci\.yml\|filters/code' -- 'tests/*.bats')` | spec (`verification.md`) |
| Minor | THEORETICAL | tests | Test 1 in `tests/ci-os-matrix.bats` inspects `literal = str(job.get("runs-on", "")) + os_list` for `"windows"` / `"macos"`. Because `cli.yml` and `cli-smoke.yml` use dynamic expressions `fromJSON(needs.changes.outputs...)`, `literal` lacks literal OS strings and skips the loop. Mutation tests ensure current workflows are covered, but a newly added job using an unfiltered dynamic expression would not be caught by Test 1 alone. | Inspection of `tests/ci-os-matrix.bats` lines 47–50. Specific coverage for `cli.yml` and `cli-smoke.yml` exists in Tests 2 and 3. | `tests/ci-os-matrix.bats` (@test "CI-013: every Windows or macOS job...") | tests (`tests/ci-os-matrix.bats`) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | Acceptance criteria verified; mutation checks verify tests fail when contract is breached. |
| Verification       | B | Comprehensive bats test suite (315 tests passing) and mutation testing; minor command duplication in `verification.md`. |
| Scope              | A | Diff strictly adheres to proposal; out-of-scope items (golangci cache, gate composite) documented and excluded. |
| Reliability        | A | Fails closed: failing `changes` job leaves OS output unset and `cli-gate` trips `if: always()` on non-success. |
| Maintainability    | A | Eliminates duplication by centralizing docs-only filters in `.github/filters/code.yml`; clear comments and tests. |
| Handoff-readiness  | A | In-flight documentation captured in Lesson 578; post-merge verification instructions clearly declared. |

### Verdict
PASS

### Recommended next steps

- **In `verification.md` (outside contract set)**:
  - Correct the verification command line in `verification.md` line 21 to remove the redundant `tests/ci-os-matrix.bats` argument, ensuring copy-paste executions exit 0 cleanly.
- **Post-merge verification**:
  - Verify AC1 live on the first documentation-only PR merged to `main` (confirming `cli/test` and `cli-smoke` run on `ubuntu-latest` only while required `test-windows` steps skip).
  - Verify AC3 live on the first PR following the merge push, inspecting integration build logs for `CACHED` layers on the pre-COPY steps.
