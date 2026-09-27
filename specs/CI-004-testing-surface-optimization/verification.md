---
tags: [spec, verification, templates]
created: "2026-09-25"
---

# Verification - CI-004-testing-surface-optimization

## Evidence

- [x] AC1 -> `38fd969`. `guard: a fake GUI launched by this file does not hold bats' fd 3` failed before the fix (`orphans holding bats' fd 3 (pipe:[5657529])`). Wall time with `tests/vault-health.bats` as companion: 1953, 1906 and 1911 ms (baseline 30.7 s with the old companion, which alone takes ~3 s). Mutation: `_gui_guard_test_shaped_processes` returning 0 at once fails `guard: the stray detector matches a test-shaped GUI process and not a human's` ("detector missed a test-shaped process").
- [x] AC2 -> `4a0f6c2`. `skills-pipeline deploys at most 6 times per run` failed at 17 before the shared homes ("compile-harness.sh --deploy ran 17 times, budget 6"), passes at 6. All 24 original tests pass, plus the 2 new ones. Local wall time per run: ~117 s to ~43 s.
- [x] AC3 -> `4a0f6c2`. `a write into a shared setup_file home fails`. Mutation: replacing the `chmod -R a-w` with `:` fails it. It skips under root, where mode bits do not bind.
- [x] AC4 -> `21c4eef`. `timeout-minutes: 20`; the CI-001 loan comment is replaced by the distribution and the rule for raising it.
- [x] AC5 -> `b48a288`. `guard: the stray detector ignores a test-shaped process from another bats run` failed before the fix ("matched another run's process; teardown would kill it"). It also asserts the detector matches nothing when `BATS_RUN_TMPDIR` is empty; mutation: dropping that guard fails it ("matched with BATS_RUN_TMPDIR empty").
- [ ] AC6 -> PR 2 switches CI to `bats --jobs "$(nproc)" --no-parallelize-within-files`. Local, GNU parallel 20231122 (the runner's version), `--jobs 4` under a 3G cap, three runs: 89, 90 and 89 s; each 1679 tests, 1678 ok, the one failure #1641 (environmental, also failing serially). Serial on the same tree: 250 s. `git status` clean afterwards. Fixture audit: no test writes a fixed `/tmp` path shared across processes (`check-spec-gate.bats` keys its fixture on `$$`), none binds a port, and `compile-harness.bats` runs `--refresh` against its own temporary repo. Pending: the CI step on the PR run and the mean of the first 5 runs on `main`.
- [ ] AC7-AC10 -> PRs 3 to 6.

CI step "Run bats test suite" before and after: recorded from this PR's run.

## Test status

- `shellcheck -s bash` clean on both edited bats files.
- `tests/guard-no-gui.bats`, `tests/skills-pipeline.bats` and `tests/guard-lesson-numbers-unique.bats`: green, twice each.
- `actionlint` on `ci.yml`: one pre-existing SC2129 style note at line 491 (a step this PR does not touch); left as is.

## Decisions made during implementation

- P0.1 closes the descriptors through a `_launch_fake` helper instead of editing each launch, so the new fd test and the detector test exercise the same launch path. A mutation of the helper is a mutation of both.
- P0.2's `stub_copilot` helper had no caller left once the two copilot tests moved to the shared home; it was removed and its rationale moved into `setup_file`.
- f1 moved to a lighter companion and millisecond precision: at whole seconds, a 4.6 s run could read as 5 and fail at random.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons/`? Yes: lesson 309, a test filter that matches nothing passes.
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CI-004-testing-surface-optimization/` -> `specs/archive/CI-004-testing-surface-optimization/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
