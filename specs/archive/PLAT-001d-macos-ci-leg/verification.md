---
tags: [spec, verification, templates]
created: "2026-10-07"
---

# Verification - PLAT-001d-macos-ci-leg

## Evidence

> Delivered in three PRs (see `tasks.md`). The evidence below was measured on the combined change, rebased onto `main` at e65182a7 after #2070 replaced `tests/knowledge-crystallize-go-parity.bats` with a Go test. That file left the tier, which went from 285 tests to 272. Re-measured after #2150 retired `tests/vault-health-golden.bats` (19 tagged tests) and #2146 added one runner test: the tier is 254, the full suite 1804; the figures below are that re-measurement, on the head of #2147 over `main` at 3d1f07d1. The ticks below record that measurement: AC4 is made true by #2145, AC2, AC3 and AC6 by #2146, AC5 by #2147, so on #2145's tree alone the commands for AC2, AC3, AC5 and AC6 (and `features.json` f2 to f6) name files that do not exist yet. Every `features.json` entry stays `pending` until `dotf spec archive` runs them on `main` after the third PR.

- [x] AC1 (Go injection seams) -> `cd cli && go test ./internal/doctor/ -run 'TestContractOS|TestCheckContractPath_Dialects|TestCheckContractEnvVars_WindowsDialect'`: doctor reads `System.GOOS` for the darwin and windows dialects, so layer 1 needs no new work. Correction at archive (review F1): the gap this line recorded, `checks_catalog.go` and `checks_repodir.go` reading `runtime.GOOS` directly, was already closed by #2095 (`7f648fd6`, with `goos_seam_test.go`) before this spec's base; #2061 is closed.
- [x] AC2 (tier tagged, non-empty, passes under bash 3.2 with no `sha256sum`) -> `PATH="/bin:...(no /sbin)" ./scripts/run-bats.sh --expect-bash 3 --filter-tags os-sensitive` -> `bash 3.2.57 at /bin/bash`, `254 test(s) tagged os-sensitive`, 254 ok, 0 not ok
- [x] AC3 (tag mistakes fail) -> `tests/guard-bats-tags.bats` (mutation: `file_tag=` in `tests/utils.bats` turns test 2 red), `tests/run-bats-real.bats` ("a tag no test carries fails the run")
- [x] AC4 (classified and resolved) -> table below; `bats --jobs 8 --no-parallelize-within-files tests/*.bats` -> 1804 tests, 0 failed, 99 skipped (each with a reason), with the pinned Python 3.12 on `PATH`. Under the system Python 3.9, the 13 `pr-agent-config` tests that `import tomllib` (3.11+) fail; that is class (b), environment only, tracked in #2062
- [x] AC5 (workflow) -> job declared and checked by `tests/workflow-job-names.bats`, `tests/workflow-timeouts.bats`, `tests/ci-path-filtering.bats`; first hosted run on PR #2063 (run 37590992834, job `test-macos`): pass in 4m19s, every step success (age, bats, GNU parallel install, shells, Go build/vet/`GOOS=darwin` vet/test, tier under bash 3.2; the `GOOS=darwin` line was later dropped as a repeat of the native vet on that runner, #2147 review); the full-suite step is `push`-only and ran skipped, so its first hosted run is the merge to main
- [x] AC6 (one script) -> `tests/run-bats.bats`, `tests/run-bats-real.bats`; mutation: putting `$(nproc)` back into `ci.yml` turns the `refute_grep` test red (re-run on #2146's head: red, the match printed at the restored line; `refute_grep` is the ERE form, `_refute_grep_impl E`). Empty option values, from the #2146 review: `--filter-tags ""` used to read as no filter and ran the whole suite in place of the tier, and an empty `--expect-bash` or `--report-dir` skipped its check the same way. Each is now a usage error (exit 2). The test `an option given an empty value is a usage error` fails on the old script (`--filter-tags exited 0`) and passes on the new one; `run-bats`, `run-bats-real`, `guard-bats-tags`: 15/15.

Review choices recorded here because they shape the tier rather than one PR:
- The tier guard keeps a floor (`>= 100`) rather than an exact count or a file manifest. The tag comments are the single source of truth for the tier, and a second copy would have to change with every legitimately tagged test. The floor catches wholesale loss (a broken tag spelling, a renamed tag), and the guard already rejects near-miss spellings.
- The `ci.yml` structure test needs PyYAML and does not `skip` without it. A skip there would turn a broken runner image into a pass; both CI jobs install PyYAML, and `ModuleNotFoundError` already names the cause.

### Classification of the 88 failures (macOS 26, arm64, bash 3.2, 2026-10-07)

| Class | Tests | Root cause | Resolution |
|---|---|---|---|
| (b) missing tool | 27 | no PyYAML for the Python that reads workflow YAML (pr-agent-config 13, agent-runtime-deployment, check-review-attestation, dependabot-ci, release-pr-body-refs, review-attestation-workflow, secrets-only-scope, model-canary) | environment; CI uses `actions/setup-python` plus `pip install pyyaml` |
| (b) missing tool | 13 | macOS Python is 3.9, no `tomllib` (pr-agent-config) | environment; Python 3.12 from `versions.conf` in CI |
| (a) script | 15 | BSD `wc` pads counts: `vault-health.sh` (13 golden tests), `compile-harness.sh` (2) | `\| tr -d ' '`; goldens recaptured, byte-identical, only the ORACLE hash moved. The `vault-health.sh` half was dropped on rebase: #2150 retired the script and its golden suite, and the corpus is now replayed by a Go test |
| (a) script | 3 | `check-bats-names.sh` used `grep -P` and discarded stderr, so it passed every file | `grep -E` with `[:print:]`; grep exit 2 now fails; test with an unreadable file |
| (a) script | 3 | no `sha256sum` (absent before macOS 26, `/sbin` after): `install-dotf.sh` | `_dotf_sha256` falls back to `shasum -a 256`; `sha_of` in `compile-harness.sh` likewise |
| (a) script | 1 | `chmod --reference` is GNU-only: pi `enabledModels` sync in `setup-linux.sh` | mode read with GNU or BSD `stat`; two `stat -c %s` fall back to `stat -f %z` |
| (a) script | 2 | logical against physical path: `memory-sink-guard.sh`, `chain-local-hook.sh` | `cd … && pwd -P` on the configured vault and store |
| (a) suite | 2 | `pgrep -a` lists no command line on BSD, so the stray-GUI detector in `setup_suite.bash` matched nothing | `ps -axo pid=,args=` plus awk on the command's basename |
| (b) test | 12 | crystallize parity sandbox is a logical `/var/folders` path, dotf prints the physical one | fix dropped on rebase: #2070 replaced the suite with `cli/internal/cmd/vault_crystallize_golden_test.go` |
| (b) test | 2 | same, `precommit-fallback.bats` fixtures | `WORK` canonicalised |
| (b) test | 2 | GNU `sed \s` in `claude-plugins.bats`, `stat -c` in `hermes-setup.bats` | `[[:space:]]`; `file_mode` in `tests/lib/os.bash` |
| (b) test | 1 | copying `/bin/bash` and running it is killed at exec on Apple silicon (`install-dotf.bats`) | ad-hoc `codesign` of the copy on Darwin |
| (b) test | 1 | `antigravity.bats` reads the real `~/.gemini`; agy had created an empty config, the deploy had not run | skip with that reason when the file exists and is empty |
| (c) Linux-only | 4 | the release-PR body step runs on ubuntu with GNU `sed` (`\b`, `I` flag); 3 more of its tests passed vacuously on BSD `sed` | `require_gnu_sed` skips all 7 with the reason |

Totals: (a) 26, (b) 58 (40 missing tool, 18 test), (c) 4. No failure was left unexplained.

### The tagged tier (257 tests: 22 whole files, 6 single tests in 4 more)

Whole files, tagged because their subject is a BSD/GNU or shell difference: `bash32-portable`, `utils`, `aliases`, `shell-functions`, `shell-wrapper-dedup`, `shell-alias-collision`, `shell-profile`, `local-override`, `version-gte` (shell functions in bash and zsh); `install-dotf`, `precommit-config` (was `install-precommit`, whose installer was deleted), `executable-bit`, `gitattributes-eol`, `age-scripts` (installers, modes, line endings, the age binary); `check-bats-names`, `guard-memory-sink`, `precommit-fallback`, `hermes-setup`, `harness-generated-sha` (each one a defect found in the triage; `knowledge-crystallize-go-parity` and `vault-health-golden` were too, until #2070 and #2150 replaced them with Go tests); `run-bats`, `run-bats-real` (the runner itself); `pr-agent-model-preflight-real`, added after the first full-suite run on `main` (run 37722585322) failed its three tests in setup: `HTTPServer.server_bind`'s `getfqdn` reverse lookup outlived the 5 s start-up wait on the macOS runner, so the file's local-server tests are an OS difference too. Single tests: two over-cap doctrine warnings (`compile-harness`), two stray-detector tests (`guard-no-gui`), the Windows plugin list (`claude-plugins`), the pi settings sync (`pi-config`). Not tagged: tests that only read repo text (workflow and docs lints), which answer the same on every OS.

### Follow-up: the tier's `[[ ]]` assertions were partly vacuous (#2164)

The tier runs under `/bin/bash` 3.2 on purpose, and before bash 4.1 a failing `[[ ]]` does not trip `set -e`. 54 mid-test `[[ ]]` lines in the tier could not fail their test on this leg, so until #2164 the leg checked less than the 257-test count suggests. #2164 appends `|| false` to all 668 bare lines in the suite and adds `tests/guard-bats-dbracket.bats`. The tier still passes 257/257 under bash 3.2 afterwards, and the full suite fails the same 13 tests before and after (all from #2062). So no assertion had been hiding a real defect.

## Test status

- bats, full, macOS arm64, bash 3.2: `bats --jobs 8 --no-parallelize-within-files tests/*.bats` -> 1804 tests, 0 failed (pinned Python 3.12 on `PATH`; see AC4)
- bats tier: `./scripts/run-bats.sh --expect-bash 3 --filter-tags os-sensitive` (no `/sbin` on PATH) -> 254 ok
- Go: `go build ./... && go vet ./... && GOOS=windows go vet ./... && GOOS=darwin go vet ./... && go test ./...` -> ok; `golangci-lint run` (2.12.2) -> 0 issues
- `shellcheck` on every changed `.sh` at warning severity: clean. `actionlint`: not installed here; the workflow is parsed by PyYAML and by the repo's workflow bats guards
- Manual: the vault-health goldens recaptured on bash 3.2 differed from Linux's by nothing but the ORACLE hash (moot since #2150 made the ORACLE a historical record)

### Follow-up: the mock-environment step hid its own failures (#2152)

PR-Agent flagged on #2147 that the new job's "Setup mock environment" step was copied from the Linux `test` job together with its silencers: age-keygen's stderr went to `/dev/null` and the fixture copy ended in `2>/dev/null || true`. The suite skips on a missing key or fixture, so a broken setup would have shown up as skipped tests on a green leg, which is the failure this spec exists to prevent on macOS.

Declined on #2147 because the Linux job shares the defect; it is fixed for both jobs in the follow-up PR:

- `scripts/setup-mock-env.sh` is now the only definition. Both `Setup mock environment` steps run it, and nothing else.
- It refuses to run outside the repository root or without `age-keygen` on PATH, then asserts a non-empty key and non-empty `scripts/` and `sensitive/` copies, failing with `::error::`.
- The two `ln -sf ~/.dotfiles/.{zshrc,bashrc}` lines are gone: the mock never holds those files, so the links always dangled.
- Every failure inside the script, including a failing `mkdir` or `cp` that `set -e` alone would have stopped with plain stderr, goes through `fail()`, so the step always carries its annotation (PR-Agent on #2160). `ci.yml` may no longer contain `2>/dev/null || true` anywhere.
- Evidence: `tests/setup-mock-env.bats`, 11 ok with the fix and 11 not ok with the script and workflow change removed. `tests/setup-mock-env-real.bats` runs the real age-keygen in CI, as BUG-055 pairing requires. On the `test-macos` runner the step printed `mock environment ready`, so BSD `find -quit` works there. Every bats file that references `ci.yml`: 308 ok.

## Decisions made during implementation

- The tier is the OS-sensitive subset, justified per file above, not "everything": a whole-suite macOS leg would mostly re-test repo text that does not vary by OS. The full suite still runs on macOS on a push to main, so the tier cannot hide a regression for long.
- `run-bats.sh` fails an empty tag selection itself. bats exits 0 on zero matches, so a mistyped tag would turn the leg into a green no-op.
- No separate "zsh leg": bats runs under bash on every OS, and the tagged tests start zsh themselves. The job asserts zsh exists and the bash major version is 3 instead.
- GNU parallel is installed by a workflow step (`brew install parallel`), with the same `::error` preflight as Linux. Its declaration belongs to #2013 P5b (a system-package source in the catalog); `packages.json` is untouched here.
- Gaps found and ticketed rather than fixed: `test-windows` still selects by file list instead of the tag (noted on #2059, the Windows-leg slimming issue); Python 3.12 and PyYAML are an undeclared dependency of the suite on a developer Mac (#2062).

- AC5 closed (2026-10-09): `test-macos` was green on the PR that added it (#2063, run 37590992834, branch `ci/macos-leg`), and on main both tiers pass. Run 38014809265 (push of `e04219c2`, #2222) reports `bats, os-sensitive tier (bash 3.2)=success` and `bats, full suite (bash 3.2, main only)=success`.

## Review dispositions (archive review, `nan/mimo-v2.6-flash`, PASS-WITH-GAPS)

- F1 (Minor, REAL), stale seam gap: corrected in this file (AC1 line and the gaps line). The same sentence in `proposal.md` "Risks" is declined as an edit: the review closed the contract set, and a post-review edit to it would re-open the verdict. The correction lives here.
- F2 (Minor, REAL), counts drifted: re-measured at archive on `b6fd39e9` by the reviewer: os-sensitive tier **268**, full suite **1896** tests, 0 failed. The figures above (254/257, 1804) stay as the measurements of their date; the tier grew with later PRs that tagged their own tests.
- F3 (Minor, SPECULATIVE), the `>= 100` floor tolerates partial tier loss: declined. A per-file tag manifest is the second copy of the tag list this spec rejected on purpose (see the design decision above); the floor catches the wholesale loss it was built for.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-342-linux-green-says-nothing-about-the-bsd-half-of-every-unix-tool.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: no new architecture, the tier follows ADR-020's two-loop split and ADR-044's mise decision
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: the tag-tier idea is specific to this suite until a second repo needs it

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/PLAT-001d-macos-ci-leg/` -> `specs/archive/PLAT-001d-macos-ci-leg/`
- [x] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018): the spec's only ticket is the epic #2013, which stays open for its other tracks; the archive is recorded on the epic with this PR's link
- [x] Promotions above executed (if any)
