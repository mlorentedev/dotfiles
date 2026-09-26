---
id: "CI-004-testing-surface-optimization"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-09-25"
issue: "mlorentedev/dotfiles#1739"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, ci, testing, bats]
template_version: "1.0"
---

# CI-004: testing surface optimization

## Why

<!-- from issue #1739: CI-004: the PR loop waits on test-harness waste (a 30s orphaned sleep, 17 redundant deploys, a serial bats run) and on a 45-minute ceiling whose loan has expired -->

The repository is public, so runner minutes cost nothing. What a slow pipeline costs is **feedback
latency** (a PR waits 8–15 minutes, a push to `main` 15–26) and **concurrency slots**, which every
repository on the account shares. A large part of that wait is test-harness waste rather than
verification: a guard test that waits out an orphaned `sleep 30`, a smoke file that re-deploys the
whole harness 17 times, a serial bats run on a multi-core runner, and an uncached image build. At the
same time two signals are weaker than they look: the Windows doctor gate runs without a token, so 43
of its checks answer "not verified", and a required context named `lint` is reported by two
workflows.

Source: the five-audit synthesis in the vault, `10_projects/dotfiles/research/2026-09-25-ci-testing-surface-synthesis.md`.
Its rows are **claims from other models**, and every one in scope was re-probed against `main` on
2026-09-25 before it was written here. The probes corrected four of them; see "Corrections to the
synthesis" below.

## What

Each row ships as its own PR, in the order below. The numbering follows the synthesis so its
evidence can be traced; the order follows readiness.

| PR | Row | Behavior after the change |
|---|---|---|
| 1 | P0.1 | `tests/guard-no-gui.bats` no longer holds bats' output descriptors open through an orphaned `sleep`. The file takes ~1 s instead of 30.9 s (local, measured) |
| 1 | P0.2 | `tests/skills-pipeline.bats` deploys once per shared precondition in `setup_file`: 6 deploys instead of 17. The shared homes are read-only, so a test that writes to one fails loudly instead of contaminating its siblings |
| 1 | P0.6 | `test-windows` has a 20-minute ceiling instead of the 45 that CI-001 raised as a loan "until the reconcile is off the blocking path". CI-002 (#1482) did that |
| 2 | P0.3 | The stray-GUI detector in `tests/setup_suite.bash` matches only **this run's** `$BATS_RUN_TMPDIR`, and CI runs the Linux suite with `bats --jobs N --no-parallelize-within-files` |
| 3 | P0.7 | `cli.yml`'s lint job reports as `cli-lint`, so the required context `lint` has one reporter, `ci.yml` |
| 4 | P0.5 | The `test-windows` doctor-gate step gets `GH_TOKEN`, so the remote checks answer instead of reporting "not verified". **Blocked**: see the risks |
| 5 | P1.3 | The `integration` image build reuses a layer cache between runs |
| 6 | P1.5 | Go tests run once per OS, **only if** a measurement shows it does not lengthen the critical path (see the risks) |

### Corrections to the synthesis (probed 2026-09-25)

- **The id.** The synthesis names this spec CI-003. That id is archived (#1486, the pi reconcile observability).
- **P0.3's hazard.** With a single `bats --jobs` invocation, `setup_suite` runs once, so "single-file shards skip the guard" does not arise unless CI shards across invocations, which this spec does not do. The real hazard is the stray detector's pattern `*--user-data-dir=*bats-run*`: it matches **any** bats run on the machine, so one suite's `teardown_suite` can report and `kill -TERM` another concurrent suite's fixtures. On a box that runs several agent sessions at once, that is the normal case, not an edge case.
- **P0.5's risk.** It is not "low risk". With a token, `checks_spec_issue_state.go:57` turns an open spec with a closed issue into a `[FAIL]`, and six specs are in that state today (CLI-057, CLI-062, GUARD-005, GUARD-006, HARNESS-106, WIN-007). The token would turn the Windows gate red on its first run. Those failures are real, and invisible today *because* the token is missing. The known-failures list is only for runner-only conditions, so the fix is to dispose of the six, not to list them. Of the 43 "not verified" warnings measured on run `2026-09-26T00:09Z`, 33 are spec-issue checks that a repo-scoped `github.token` can answer. The 10 branch-protection checks stay unverified, because reading protection needs administration access that `GITHUB_TOKEN` cannot be granted.
- **P1.5's direction.** Moving `go test` into `test-windows` adds its runtime to the job that is already the critical path (p50 438 s on PRs). It saves a concurrency slot at the price of latency, which is the objective this spec optimizes.

## Out of scope

- **P0.4 and P1.4** (the pi CLI install's discarded output, fanning out the pi package installs). Both edit the setup twins, and new logic does not go into `setup-*.sh`/`.ps1` while the reconciler is being ported to Go (#1628, owner decision on #1625). They are recorded on #1628.
- **P1.2** (splitting the `code` path filter). CI-002 (#1478) declares it as its own PR 2.
- **P1.1** (a Windows PR fast path without the OS bootstrap). It reduces TEST-003's end-to-end coverage, so it needs its own spec and an ADR.
- **P2.x** (`dotf test <tier>`, the grep-to-Go migration, a duration budget guard, parity ports). Filed as separate tickets.
- Adding `cli-lint` to the required contexts. That is an edit to `forge/branch-protection.json` applied through GUARD-017 PR-B, and it is the owner's decision.

## Risks / open questions

- **P0.5 is blocked on six spec archives.** CLI-057, CLI-062, GUARD-005, GUARD-006, HARNESS-106 and WIN-007 must be disposed of first (`dotf spec archive`, or `--abandoned`). They are queued in the `feat-harness-hardening` thread (W1.4). PR 4 does not open before `dotf doctor` reports zero spec-issue-state `[FAIL]`s on `main`.
- **P0.3 needs GNU `parallel`** (`bats --jobs` requires it), which is a new CI dependency. It is not installed on this machine. It must be verified present on `ubuntu-latest` or installed, and pinned if it is installed. Resolve this before PR 2's code.
- **P0.3 may expose fixture collisions.** Files that write a fixed path (`/tmp/...`, the real `$HOME`) collide when they run concurrently. #1409 was this class. PR 2 audits for them and runs the parallel suite repeatedly before switching CI.
- **P0.2's shared homes rely on the tests being read-only.** They are enforced with `chmod -R a-w` rather than trusted; a write fails the test instead of corrupting a sibling.
- **P0.6 on `main`.** A push still runs the full pi reconcile, which CI-003 measured at 24x variance on identical input. Since 2026-09-05, `push`: n=111, p50 544 s, p99 858 s, max 961 s; `pull_request`: n=198, max 1000 s. 20 minutes is 1200 s, 25% above the maximum observed. A slow registry night can still cross it, and that is the signal the ceiling exists to give.
- **P1.3 adds an action or a registry.** A BuildKit GHA cache needs `docker/setup-buildx-action` plus `docker/build-push-action` (pinned by SHA); a GHCR base image needs a publish workflow. Decide in PR 5.
- **P1.5 may be declined.** If measurement confirms that it lengthens the critical path, the row closes as "declined, with numbers" rather than shipping.

## Acceptance criteria

- [ ] AC1 — `guard-no-gui.bats`, run together with another file so `setup_suite` loads, finishes in under 5 s of wall time (baseline 30.9 s), and its stray-detector test still fails when the detector is broken (mutation).
- [ ] AC2 — `skills-pipeline.bats` invokes `compile-harness.sh --deploy` at most 6 times per run (baseline 17), counted by a wrapper rather than by grep, and every one of its 24 tests still passes.
- [ ] AC3 — A test that writes into a shared `setup_file` home fails. This is shown by a deliberate write under mutation.
- [ ] AC4 — `test-windows` declares `timeout-minutes: 20`, and the CI-001 loan comment is replaced by the measured distribution that justifies it.
- [ ] AC5 — The stray detector ignores a test-shaped process whose `--user-data-dir` lies in a **different** bats run's tmpdir, and still reports one inside its own. Both halves are asserted.
- [ ] AC6 — CI runs the Linux bats suite with `--jobs`, and the "Run bats test suite" step's mean over the first 5 runs on `main` is at most half the baseline (mean 199 s over 8 runs of `main`, 2026-09-25).
- [ ] AC7 — Exactly one workflow reports a check named `lint`, and `cli.yml`'s lint reports as `cli-lint`.
- [ ] AC8 — The `test-windows` doctor gate runs with `GH_TOKEN`. Its log shows zero "set the GH_TOKEN environment variable" lines (baseline 43), and the gate is green.
- [ ] AC9 — The `integration` job's "Build integration test container" step takes at most 20 s on a cache hit (baseline mean 72 s over 8 runs).
- [ ] AC10 — P1.5 ships, or it is declined with the critical-path measurement recorded in `verification.md`.

## References

- Bitácora: #1739. Related: #1478 (CI-002, P1.2), #1628 (pi reconciler to Go, P0.4 and P1.4), #1472 (CI-001, the ceiling loan), #1486 (CI-003, archived), #1409 (fixture isolation), #1625 (W1.4 review queue).
- Synthesis: vault `10_projects/dotfiles/research/2026-09-25-ci-testing-surface-synthesis.md`.
- Patterns: `00_meta/patterns/pattern-spec-driven-development.md`, `00_meta/patterns/pattern-git-workflow.md`.
