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
- [x] AC6 -> PR 2 (#1781) switches CI to `bats --jobs "$(nproc)" --no-parallelize-within-files`. Local, GNU parallel 20231122 (the runner's version), `--jobs 4` under a 3G cap, three runs: 89, 90 and 89 s; each 1679 tests, 1678 ok, the one failure #1641 (environmental, also failing serially). Serial on the same tree: 250 s. `git status` clean afterwards. Fixture audit: no test binds a port, and `compile-harness.bats` runs `--refresh` against its own temporary repo. The audit's first claim, that no test writes a fixed `/tmp` path, was wrong. `check-doc-paths.bats`, `docs-drift.bats` and `check-md-escapes.bats` built their scratch dir at a fixed name under `$BATS_TMPDIR`, which is `/tmp`, shared by every bats process, not a per-run dir. Within one run `--no-parallelize-within-files` hid it. Two full runs in two worktrees at once (2026-09-27, after the rebase on `c6f3bcf`) failed four `check-doc-paths` tests and one `docs-drift` test, a different subset in each. Fixed in `176f217`: they use `$BATS_TEST_TMPDIR`, and `tests/guard-bats-tmpdir-isolation.bats` fails on any `$BATS_TMPDIR` path not made unique by `mktemp` (red on the three files before the fix). After it, three concurrent pairs of the three files in two worktrees: 6/6 green. In CI, the "Run bats test suite" step took 98 s on the PR's final run (36369686203). On the first 5 `main` runs after the merge (36370606813, 36371533180, 36372314336, 36373587838, 36374139451) it took 96, 95, 97, 97 and 97 s, mean 96.4 s. The limit is 99.5 s, half the 199 s baseline. The two serial `main` runs just before (36366382809, 36368275056) took 172 and 165 s.
- [x] AC7 -> PR 3 (#1782). `lint` appears once in `gh pr checks` on #1782, #1796 and #1797. `cli-gate` is green on a PR with no Go change (#1796: `cli-lint` skipped, gate green) and on one with a Go change (#1797: five `cli/` files, `test` and `cli-lint` green, gate green). The review of #1782 found that `cli-gate` did not need `release-snapshot`, so a red goreleaser snapshot let the gate go green. The follow-up PR adds it. It also adds a third check to `tests/lib/check-workflow-contexts.py`: an aggregate gate must need every job in its workflow that can run on a pull request. That check was red on `release-snapshot` before the fix. The apply ran later on 2026-09-28. Live required checks are now `cli-gate,lint,lint-powershell,review-attestation,spec-gate,test,test-windows`, and `dotf forge protection check --repo mlorentedev/dotfiles` reports 1 ok, 0 drift.
- [x] AC8 -> PR 4. Housekeeping gate: `dotf spec audit` on `main` at `20aa19e` reports `[OK] 34 active spec(s), every one tracking an open issue`. Baseline: the doctor-gate step of `test-windows` in main run 36293780297 prints "set the GH_TOKEN environment variable" 43 times, from `spec-issue-state` (every active spec `not verified`) and `branch-protection`; the setup step's own doctor prints another 43, which this change leaves alone. The premise that the workflow already grants `issues: read` was wrong: it grants `contents: read` only, so the job now declares both. After, on #1813's run 36508874767 (job 109216637295): 0 in the doctor-gate step and the gate green (`0 known runner-only FAIL(s), 0 unexpected, 0 stale`); `spec-issue-state` reports its section ok with no `not verified` WARN; `branch-protection` now WARNs `Resource not accessible by integration (HTTP 403)` instead of the missing-token line. All 44 remaining lines in the job come from the setup step.
- [x] AC9 -> PR 5, **P1.3 declined.** Layer timings of the "Build integration test container" step on main (71 s in total): base image 5.1 s, apt 11.6 s, age 0.7 s, bats 0.7 s and the Go toolchain 2.7 s. Everything before the repo `COPY`, about 21 s, can be cached. After the `COPY`, `go build` takes 15.9 s, `setup-linux.sh` 26.5 s and the image export 5.7 s. That work runs again on every commit because it is the test, so no layer cache, GHA or published image, can bring the step to 20 s. Critical path on 11 PR runs of CI: `integration` finished 103-164 s after the first job started, and `test-windows` 401-1024 s after; every run ended with `test-windows`. A faster `integration` shortens no PR's wait and would add two third-party actions. Lesson 318.
- [x] AC10 -> PR 6 (carried in the same PR as PR 5, #1819), **P1.5 declined.** `go test ./...` on `windows-latest` in `cli.yml`: 81, 114, 98, 108, 107, 93 and 110 s over 7 runs. The `test-windows` job over the last 20 CI runs has a p50 of 486 s and a max of 1018 s, and it is already the job that ends every PR run. Folding the Go tests into it would add about 100 s of serial time to the critical path, which the P1.5 rule declines.

## Test status

- `shellcheck -s bash` clean on both edited bats files.
- `tests/guard-no-gui.bats`, `tests/skills-pipeline.bats` and `tests/guard-lesson-numbers-unique.bats`: green, twice each.
- `actionlint` on `ci.yml`: one pre-existing SC2129 style note at line 491 (a step this PR does not touch); left as is.

## Decisions made during implementation

- AC9 was amended from "ships" to "declined with numbers" in PR 5, and the original text stays struck through in `proposal.md`. The criterion had been written from job durations without the layer split or the critical path. Lesson 318 records the rule.
- AC8 changes what the gate can fail on. With a token, `spec-issue-state` reads each spec's issue, so an active spec whose issue is closed is a `[FAIL]` and turns `test-windows` red. On a pull request the issue it closes is still open, so the PR passes; the push run on `main` after the merge is where an unarchived spec with a closed issue shows up. That is the doctrine the check already states, now enforced. `branch-protection` stays a WARN: a workflow token cannot read branch protection.
- P0.1 closes the descriptors through a `_launch_fake` helper instead of editing each launch, so the new fd test and the detector test exercise the same launch path. A mutation of the helper is a mutation of both.
- P0.2's `stub_copilot` helper had no caller left once the two copilot tests moved to the shared home; it was removed and its rationale moved into `setup_file`.
- f1 moved to a lighter companion and millisecond precision: at whole seconds, a 4.6 s run could read as 5 and fail at random.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons/`? Yes: lesson 309 (a test filter that matches nothing passes) and lesson 318 (a speed-up must name the critical path it shortens).
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? No: every decision here is local to the CI jobs and is recorded in this spec.
- [x] New pattern candidate for `00_meta/patterns/`? No: lesson 318 is the first occurrence. Promote it if a second project repeats it.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CI-004-testing-surface-optimization/` -> `specs/archive/CI-004-testing-surface-optimization/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
