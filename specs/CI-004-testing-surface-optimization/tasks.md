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
- [ ] Owner approves the plan (proposal + this file) before any code
- [x] Out-of-scope rows recorded where they now live: P0.4 and P1.4 on #1628, P1.2 on #1478, P1.1 as #1741 (CI-005), P2.1–P2.4 as #1742–#1745 (CI-006 to CI-009)

Heavy runs go through `systemd-run --user --scope -p MemoryMax=3G -p MemorySwapMax=0`. The box has ~3 GB free.
The bats verifications need two files so `setup_suite` loads; `tests/guard-lesson-numbers-unique.bats` is the cheap companion.

## Implementation

### PR 1: the spec, P0.1, P0.2 and P0.6

- [ ] [AC1] Write a failing timing assertion. `tests/guard-no-gui.bats` gets a test that bounds how long the fake GUI's descendants outlive the test. Run `bats tests/guard-no-gui.bats tests/guard-lesson-numbers-unique.bats`. Expected: the new test fails, and the file still takes ~31 s.
- [ ] [AC1] Fix the leak at `tests/guard-no-gui.bats:121-127`. The fake becomes `exec sleep 30`, so the pid that `kill -9` receives is the sleeper itself. Its argv keeps `--user-data-dir` only if `exec` preserves it, so check with `pgrep -a`. If it does not, redirect the fake's descriptors (`>/dev/null 2>&1 3>&-`) and keep `sh`. Expected: the file finishes in under 5 s, and every test passes.
- [ ] [AC1] Mutation. Make `_gui_guard_test_shaped_processes` return nothing. Expected: the stray-detector test fails. Record the diff and the output in `verification.md`.
- [ ] [AC2] Add a deploy-counting wrapper. It is a test-local stub in front of `scripts/compile-harness.sh` that appends one line per `--deploy` to `$BATS_FILE_TMPDIR/deploys`, plus an assertion in `teardown_file` that the count is ≤ 6. Expected: it fails at 17.
- [ ] [AC2] In `setup_file`, deploy once into `$BATS_FILE_TMPDIR/home-clean` (ambient PATH; tests 1–8, 10, 11 and 19) and once into `$BATS_FILE_TMPDIR/home-copilot` (the `stub_copilot` PATH; tests 12 and 13). Those tests stop deploying and read the shared home. Tests 9, 14, 15 and 16 seed state or change PATH, so they keep their own `FAKEHOME`. Expected: 24/24 pass, and the count is 6.
- [ ] [AC3] `chmod -R a-w` both shared homes at the end of `setup_file`; `teardown_file` restores `u+w` before removing them. Mutation: add a `touch` into the shared home in test 1. Expected: test 1 fails. Record it.
- [ ] [AC4] Set `.github/workflows/ci.yml` `test-windows` `timeout-minutes: 45` to `20`. Replace the CI-001 loan comment with the distribution from `proposal.md` and the rule for revisiting it: raise it only with a run that crossed it and a ticket. Verify with `grep -n 'timeout-minutes: 20' .github/workflows/ci.yml` inside the `test-windows` block.
- [ ] Before and after timings go in `verification.md`: local wall time for both bats files, and the CI "Run bats test suite" step on the PR run.

### PR 2: P0.3, a run-scoped stray detector and `bats --jobs`

- [ ] Verify that GNU `parallel` exists on `ubuntu-latest`, with a throwaway workflow step or the runner-images manifest. If it is missing, add an apt install step pinned in `versions.conf` (`PARALLEL_VERSION`) and declare the dependency in the PR body.
- [ ] [AC5] Write a failing test in `tests/guard-no-gui.bats`. Launch two fakes, one with `--user-data-dir` under `$BATS_RUN_TMPDIR` and one under a sibling `/tmp/bats-run-OTHER…`. Expected: the detector must report only the first. Today it reports both, so the test fails.
- [ ] [AC5] Narrow the match in `tests/setup_suite.bash` (`_gui_guard_test_shaped_processes`) from `*--user-data-dir=*bats-run*` to `*--user-data-dir="$BATS_RUN_TMPDIR"/*`, and move the existing test's fake from `/tmp/bats-run-FAKE` to `$BATS_RUN_TMPDIR`. Expected: both halves pass. Mutation: revert the pattern, and the other-run half fails.
- [ ] [AC6] Fixture-collision audit. Grep `tests/` for writes to a fixed `/tmp/` path, to the real `$HOME`, or to a fixed port. Fix or isolate each hit, or list it as known-serial in `tests/.bats-serial` if bats supports it; otherwise keep that file out of the parallel set.
- [ ] [AC6] Locally, run `bats --jobs 4 --no-parallelize-within-files tests/*.bats` three times under the memory cap. Expected: three green runs, each with the same test count as the serial run.
- [ ] [AC6] Switch `ci.yml` "Run bats test suite" to `bats --jobs "$(nproc)" --no-parallelize-within-files tests/*.bats`. Record the step time of the PR run, then the mean of the first 5 runs on `main`, in `verification.md`.

### PR 3: P0.7, one reporter for `lint`

- [ ] [AC7] Add a failing check under `tests/` that parses `.github/workflows/*.yml` and asserts that every job display name is unique across workflows. Expected: it fails on `lint`.
- [ ] [AC7] Rename `.github/workflows/cli.yml` job `lint` to `name: cli-lint`. Expected: the check passes, and `forge/branch-protection.json` is unchanged (the required `lint` stays `ci.yml`'s).
- [ ] [AC7] Verify on the PR that the `lint` context appears once in `gh pr checks`.

### PR 4: P0.5, a token for the Windows doctor gate (blocked)

- [ ] Gate: on `main`, `dotf doctor` reports no spec-issue-state `[FAIL]`. The six specs (CLI-057, CLI-062, GUARD-005, GUARD-006, HARNESS-106, WIN-007) must be archived or abandoned first.
- [ ] [AC8] Add `GH_TOKEN: ${{ github.token }}` to the **doctor-gate step's** `env:` in `ci.yml`, not the job's, so no other step inherits it. The workflow already has read permissions for issues; confirm the `permissions:` block covers `issues: read`.
- [ ] [AC8] On the PR run, count "set the GH_TOKEN environment variable" in the gate's log. Expected: 0, and the gate is green. Record the count before and after.

### PR 5: P1.3, the integration image cache

- [ ] Decide between the BuildKit GHA cache (`docker/setup-buildx-action` + `docker/build-push-action` with `cache-from/cache-to: type=gha`, `load: true`, pinned by SHA) and a published base image. Record the choice and its reason in `proposal.md`.
- [ ] [AC9] Implement it. On a second run with an unchanged `tests/Dockerfile.integration`, the build step takes ≤ 20 s. Record both runs.

### PR 6: P1.5, one Go test surface (conditional)

- [ ] [AC10] Measure `go test ./...` on `windows-latest` from `cli.yml` runs, and the `test-windows` p50. If folding the first into the second lengthens the critical path, decline: record the numbers in `verification.md` and close the row.
- [ ] [AC10] Otherwise, move `go test ./...` into the existing jobs and drop the `cli.yml` test matrix. Show that the `test-windows` p50 did not grow over the next 10 runs.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test or recorded measurement
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] `shellcheck` and `bats tests/*.bats` are green; `actionlint` is clean on the edited workflows, if it is installed
- [ ] No unrelated changes in any PR (no scope creep); every PR body states "no `.sh`/`.ps1` setup script touched"
- [ ] `verification.md` filled in, with before and after numbers per row
- [ ] Independent review through `dotf spec review CI-004-testing-surface-optimization` before archive

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
