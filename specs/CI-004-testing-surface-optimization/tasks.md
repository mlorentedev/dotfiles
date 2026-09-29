---
tags: [spec, tasks, templates]
created: "2026-09-25"
---

# Tasks - CI-004-testing-surface-optimization

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Gating issue open and self-assigned: #1739
- [x] Worktree `../dotfiles-wt-ci-004`, branch `feat/ci-004-testing-surface` from `origin/main` (`f42c212`)
- [x] `proposal.md` complete, with measured baselines for every criterion
- [x] Owner approves the plan (proposal + this file) before any code: 2026-09-27, "ok dale" on taking CI-004 over
- [x] Out-of-scope rows recorded where they now live: P0.4 and P1.4 on #1628, P1.2 on #1478, P1.1 as #1741 (CI-005), P2.1–P2.4 as #1742–#1745 (CI-006 to CI-009)

Heavy runs go through `systemd-run --user --scope -p MemoryMax=3G -p MemorySwapMax=0`. The box has ~3 GB free.
The bats verifications need two files so `setup_suite` loads. The companion for timing is `tests/vault-health.bats` (~0.5 s); `tests/guard-lesson-numbers-unique.bats` takes ~3 s alone and would dominate a 5 s bound.

## Implementation

### PR 1: the spec, P0.1, P0.2 and P0.6

- [x] [AC1] Write a failing timing assertion. `tests/guard-no-gui.bats` gets a test that bounds how long the fake GUI's descendants outlive the test. Run `bats tests/guard-no-gui.bats tests/guard-lesson-numbers-unique.bats`. Expected: the new test fails, and the file still takes ~31 s.
- [x] [AC1] Fix the leak at `tests/guard-no-gui.bats:121-127` by closing the descriptors the orphan inherits: launch both fakes with `>/dev/null 2>&1 3>&- &`. bats-core documents that a background process must close fd 3, or bats waits for it. Do not reach for `exec sleep` first: `pgrep -a obsidian` matches on the process name, and `exec` renames the process to `sleep`, which would blind the existing detector test. Expected: the file finishes in under 5 s, and every test passes.
- [x] [AC1] Mutation. Make `_gui_guard_test_shaped_processes` return nothing. Expected: the stray-detector test fails. Record the diff and the output in `verification.md`.
- [x] [AC2] Add a deploy counter. The tests call `"$SCRIPT" --deploy` by absolute path, so a PATH stub would never see them. Instead, `setup_file` writes a wrapper into `$BATS_FILE_TMPDIR` that appends one line per `--deploy` to `$BATS_FILE_TMPDIR/deploys` and `exec`s the real `scripts/compile-harness.sh`, and `setup` points `SCRIPT` at it. Add a last test, `skills-pipeline deploys at most 6 times per run`, that asserts on the count. Expected: it fails at 17.
- [x] [AC2] In `setup_file`, deploy once into `$BATS_FILE_TMPDIR/home-clean` (ambient PATH; tests 1–8, 10, 11 and 19) and once into `$BATS_FILE_TMPDIR/home-copilot` (the `stub_copilot` PATH; tests 12 and 13). Those tests stop deploying and read the shared home. Tests 9, 14, 15 and 16 seed state or change PATH, so they keep their own `FAKEHOME`. Expected: 24/24 pass, and the count is 6.
- [x] [AC3] `chmod -R a-w` both shared homes at the end of `setup_file`; `teardown_file` restores `u+w` before removing them. Add the test `a write into a shared setup_file home fails`, which runs `touch` on the clean home and asserts that it fails. Mutation: drop the `chmod`, and that test fails. Record it.
- [x] [AC4] Set `.github/workflows/ci.yml` `test-windows` `timeout-minutes: 45` to `20`. Replace the CI-001 loan comment with the distribution from `proposal.md` and the rule for revisiting it: raise it only with a run that crossed it and a ticket. Verify with `grep -n 'timeout-minutes: 20' .github/workflows/ci.yml` inside the `test-windows` block.
- [x] [AC1] [AC2] Before and after timings go in `verification.md`: local wall time for both bats files, and the CI "Run bats test suite" step on the PR run.

### PR 2: P0.3, a run-scoped stray detector and `bats --jobs`

- [x] (housekeeping, a prerequisite of AC6) Verify that GNU `parallel` exists on `ubuntu-latest`, with a throwaway workflow step or the runner-images manifest. If it is missing, add an apt install step pinned in `versions.conf` (`PARALLEL_VERSION`) and declare the dependency in the PR body.
- [x] [AC5] Write a failing test in `tests/guard-no-gui.bats`, named `guard: the stray detector ignores a test-shaped process from another bats run`. Launch two fakes, one with `--user-data-dir` under `$BATS_RUN_TMPDIR` and one under a sibling `/tmp/bats-run-OTHER…`. Expected: the detector must report only the first. Today it reports both, so the test fails.
- [x] [AC5] Narrow the match in `tests/setup_suite.bash` (`_gui_guard_test_shaped_processes`) from `*--user-data-dir=*bats-run*` to `*--user-data-dir="$BATS_RUN_TMPDIR"/*`, and move the existing test's fake from `/tmp/bats-run-FAKE` to `$BATS_RUN_TMPDIR`. Expected: both halves pass. Mutation: revert the pattern, and the other-run half fails.
- [x] [AC6] Fixture-collision audit. Grep `tests/` for writes to a fixed `/tmp/` path, to the real `$HOME`, to a fixed port, or **into the repo checkout itself**. For example, check whether `compile-harness.sh --deploy` or `--refresh` writes rendered files under `$REPO` (catalogs, `.github/copilot-instructions.md`). Two files doing that concurrently collide no matter how `$HOME` is isolated. Fix or isolate each hit, or list it as known-serial in `tests/.bats-serial` if bats supports it; otherwise keep that file out of the parallel set.
- [x] [AC6] Locally, run `bats --jobs 4 --no-parallelize-within-files tests/*.bats` three times under the memory cap. Expected: three green runs, each with the same test count as the serial run.
- [x] [AC6] Switch `ci.yml` "Run bats test suite" to `bats --jobs "$(nproc)" --no-parallelize-within-files tests/*.bats`. Record the step time of the PR run, then the mean of the first 5 runs on `main`, in `verification.md`.

### PR 3: P0.7, one reporter for `lint`, and a real Go gate

- [x] [AC7] Add `tests/workflow-job-names.bats` over `tests/lib/check-workflow-contexts.py`. It asserts two things: no two pull-request jobs (or API-posted statuses) report the same name, with matrix names expanded; and every required context in `forge/branch-protection.json` is reported by a job that reports on every pull request (no `paths` filter on the workflow, not a job-level-skippable matrix job). Expected: it fails on `lint`.
- [x] [AC7] `cli.yml`: drop the `pull_request` paths filter, add a `cli-changes` job, skip `test`, `cli-lint` and the snapshot when the diff has no Go change, and add `cli-gate` (`if: always()`, red unless every Go job succeeded or was skipped). Add `cli-gate` to the required checks in `forge/branch-protection.json`.
- [x] [AC7] Mutations: a `paths` filter restored on `pull_request` fails the test on `cli-gate`, and so does requiring `test (ubuntu-latest)` directly.
- [x] [AC7] On the PR: `lint` appears once in `gh pr checks`; `cli-gate` is green both with and without a Go change.
- [x] [AC7] Review follow-up (#1782): `cli-gate` needs `release-snapshot`, and `check-workflow-contexts.py` fails an aggregate gate that leaves out a job able to run on a pull request. Mutation: dropping `release-snapshot` from `needs` fails `tests/workflow-job-names.bats`.
- [x] The owner runs `dotf forge protection apply --repo mlorentedev/dotfiles` after the merge. (Done 2026-09-28, on the owner's request. Live required checks: `cli-gate,lint,lint-powershell,review-attestation,spec-gate,test,test-windows`; `dotf forge protection check` reports 1 ok, 0 drift.)

### PR 4: P0.5, a token for the Windows doctor gate

- [x] (housekeeping, a gate on AC8) On `main`, `dotf doctor` reports no spec-issue-state `[FAIL]`. The six specs (CLI-057, CLI-062, GUARD-005, GUARD-006, HARNESS-106, WIN-007) must be archived or abandoned first.
- [x] [AC8] Add `GH_TOKEN: ${{ github.token }}` to the **doctor-gate step's** `env:` in `ci.yml`, not the job's, so no other step inherits it. The workflow already has read permissions for issues; confirm the `permissions:` block covers `issues: read`. (It did not: the workflow grants `contents: read` only. `test-windows` now declares `contents: read` and `issues: read`.)
- [x] [AC8] On the PR run, count "set the GH_TOKEN environment variable" in the gate's log. Expected: 0, and the gate is green. Record the count before and after.

### PR 5: P1.3, the integration image cache

- [x] (housekeeping, a design decision for AC9) Decide between the BuildKit GHA cache (`docker/setup-buildx-action` + `docker/build-push-action` with `cache-from/cache-to: type=gha`, `load: true`, pinned by SHA) and a published base image. Record the choice and its reason in `proposal.md`. (Neither: P1.3 is declined. The reason is in `proposal.md` and the numbers are in `verification.md`.)
- [x] ~~[AC9] Implement it.~~ Not applicable: declined by the measurement above. On a second run with an unchanged `tests/Dockerfile.integration`, the build step takes ≤ 20 s. Record both runs.

### PR 6: P1.5, one Go test surface (conditional)

- [x] [AC10] Measure `go test ./...` on `windows-latest` from `cli.yml` runs, and the `test-windows` p50. If folding the first into the second lengthens the critical path, decline: record the numbers in `verification.md` and close the row.
- [x] ~~[AC10] Otherwise, move `go test ./...` into the existing jobs and drop the `cli.yml` test matrix. Show that the `test-windows` p50 did not grow over the next 10 runs.~~ Not applicable: the measurement above declined P1.5.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test or recorded measurement
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] `shellcheck` and `bats tests/*.bats` are green; `actionlint` is clean on the edited workflows, if it is installed — 2026-09-29 on the PR 5 branch: `shellcheck -S warning` exits 0 (CI runs `--severity=error` on the root scripts); `bats tests/*.bats` 1695/1696, the one failure being the known environmental #1641 (the local oh-my-zsh install has `glolm`, the vendored snapshot does not); `actionlint` on `ci.yml` is clean after grouping the `versions` step's redirects (SC2129, pre-existing since 465fd2a)
- [x] No unrelated changes in any PR (no scope creep); every PR body states "no `.sh`/`.ps1` setup script touched" (No CI-004 PR touches a `setup-*.sh` or `.ps1`, checked on the file lists of #1780, #1781, #1782, #1809, #1813 and this one. The bodies of #1809 and #1813 do not say so; the rest do.)
- [x] `verification.md` filled in, with before and after numbers per row
- [ ] Independent review through `dotf spec review CI-004-testing-surface-optimization` before archive
- [x] Lesson in `docs/lessons/` (PR 1, lesson 309): `bats -f` with a filter that matches nothing prints `1..0` and exits 0, so a feature check that selects a test by name passes before the test exists. Measured while writing this spec; the checks require the `ok N <name>` line instead

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/CI-004-testing-surface-optimization/features.json`):

```json
[
  {
    "id": "CI-004-testing-surface-optimization-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
