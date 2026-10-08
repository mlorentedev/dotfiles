---
tags: [spec, verification, templates]
created: "2026-10-07"
---

# Verification - PLAT-001d-macos-ci-leg

## Evidence

> Delivered in three PRs (see `tasks.md`). The evidence below was measured on the combined change, rebased onto `main` at e65182a7 after #2070 replaced `tests/knowledge-crystallize-go-parity.bats` with a Go test. That file left the tier, which went from 285 tests to 272.

- [x] AC1 (Go injection seams) -> `cd cli && go test ./internal/doctor/ -run 'TestContractOS|TestCheckContractPath_Dialects|TestCheckContractEnvVars_WindowsDialect'`: doctor reads `System.GOOS` for the darwin and windows dialects, so layer 1 needs no new work. Gap noted, not fixed: `checks_catalog.go:95` and `checks_repodir.go:101` read `runtime.GOOS` directly.
- [x] AC2 (tier tagged, non-empty, passes under bash 3.2 with no `sha256sum`) -> `PATH="/bin:...(no /sbin)" ./scripts/run-bats.sh --expect-bash 3 --filter-tags os-sensitive` -> `bash 3.2.57 at /bin/bash`, `272 test(s) tagged os-sensitive`, 272 ok, 0 not ok
- [x] AC3 (tag mistakes fail) -> `tests/guard-bats-tags.bats` (mutation: `file_tag=` in `tests/utils.bats` turns test 2 red), `tests/run-bats-real.bats` ("a tag no test carries fails the run")
- [x] AC4 (classified and resolved) -> table below; `bats --jobs 8 --no-parallelize-within-files tests/*.bats` -> 1825 tests, 0 failed, 99 skipped (each with a reason), with the pinned Python 3.12 on `PATH`. Under the system Python 3.9, the 13 `pr-agent-config` tests that `import tomllib` (3.11+) fail; that is class (b), environment only, tracked in #2062
- [x] AC5 (workflow) -> job declared and checked by `tests/workflow-job-names.bats`, `tests/workflow-timeouts.bats`, `tests/ci-path-filtering.bats`; first hosted run on PR #2063 (run 37590992834, job `test-macos`): pass in 4m19s, every step success (age, bats, GNU parallel install, shells, Go build/vet/`GOOS=darwin` vet/test, tier under bash 3.2); the full-suite step is `push`-only and ran skipped, so its first hosted run is the merge to main
- [x] AC6 (one script) -> `tests/run-bats.bats`, `tests/run-bats-real.bats`; mutation: putting `$(nproc)` back into `ci.yml` turns the `refute_grep` test red

### Classification of the 88 failures (macOS 26, arm64, bash 3.2, 2026-10-07)

| Class | Tests | Root cause | Resolution |
|---|---|---|---|
| (b) missing tool | 27 | no PyYAML for the Python that reads workflow YAML (pr-agent-config 13, agent-runtime-deployment, check-review-attestation, dependabot-ci, release-pr-body-refs, review-attestation-workflow, secrets-only-scope, model-canary) | environment; CI uses `actions/setup-python` plus `pip install pyyaml` |
| (b) missing tool | 13 | macOS Python is 3.9, no `tomllib` (pr-agent-config) | environment; Python 3.12 from `versions.conf` in CI |
| (a) script | 15 | BSD `wc` pads counts: `vault-health.sh` (13 golden tests), `compile-harness.sh` (2) | `\| tr -d ' '`; goldens recaptured, byte-identical, only the ORACLE hash moved |
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

### The tagged tier (272 tests: 22 whole files, 6 single tests in 4 more)

Whole files, tagged because their subject is a BSD/GNU or shell difference: `bash32-portable`, `utils`, `aliases`, `shell-functions`, `shell-wrapper-dedup`, `shell-alias-collision`, `shell-profile`, `local-override`, `version-gte` (shell functions in bash and zsh); `install-dotf`, `install-precommit`, `executable-bit`, `gitattributes-eol`, `age-scripts` (installers, modes, line endings, the age binary); `check-bats-names`, `guard-memory-sink`, `precommit-fallback`, `hermes-setup`, `harness-generated-sha`, `vault-health-golden` (each one a defect found in the triage; `knowledge-crystallize-go-parity` was one too, until #2070 replaced it with a Go test); `run-bats`, `run-bats-real` (the runner itself). Single tests: two over-cap doctrine warnings (`compile-harness`), two stray-detector tests (`guard-no-gui`), the Windows plugin list (`claude-plugins`), the pi settings sync (`pi-config`). Not tagged: tests that only read repo text (workflow and docs lints), which answer the same on every OS.

## Test status

- bats, full, macOS arm64, bash 3.2: `bats --jobs 8 --no-parallelize-within-files tests/*.bats` -> 1825 tests, 0 failed (pinned Python 3.12 on `PATH`; see AC4)
- bats tier: `./scripts/run-bats.sh --expect-bash 3 --filter-tags os-sensitive` (no `/sbin` on PATH) -> 272 ok
- Go: `go build ./... && go vet ./... && GOOS=windows go vet ./... && GOOS=darwin go vet ./... && go test ./...` -> ok; `golangci-lint run` (2.12.2) -> 0 issues
- `shellcheck` on every changed `.sh` at warning severity: clean. `actionlint`: not installed here; the workflow is parsed by PyYAML and by the repo's workflow bats guards
- Manual: the vault-health goldens recaptured on bash 3.2 differ from Linux's by nothing but the ORACLE hash

## Decisions made during implementation

- The tier is the OS-sensitive subset, justified per file above, not "everything": a whole-suite macOS leg would mostly re-test repo text that does not vary by OS. The full suite still runs on macOS on a push to main, so the tier cannot hide a regression for long.
- `run-bats.sh` fails an empty tag selection itself. bats exits 0 on zero matches, so a mistyped tag would turn the leg into a green no-op.
- No separate "zsh leg": bats runs under bash on every OS, and the tagged tests start zsh themselves. The job asserts zsh exists and the bash major version is 3 instead.
- GNU parallel is installed by a workflow step (`brew install parallel`), with the same `::error` preflight as Linux. Its declaration belongs to #2013 P5b (a system-package source in the catalog); `packages.json` is untouched here.
- Gaps found and ticketed rather than fixed: `test-windows` still selects by file list instead of the tag (noted on #2059, the Windows-leg slimming issue); `checks_catalog.go` and `checks_repodir.go` bypass the `GOOS` seam (#2061); Python 3.12 and PyYAML are an undeclared dependency of the suite on a developer Mac (#2062).

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-342-linux-green-says-nothing-about-the-bsd-half-of-every-unix-tool.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: no new architecture, the tier follows ADR-020's two-loop split and ADR-044's mise decision
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: the tag-tier idea is specific to this suite until a second repo needs it

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/PLAT-001d-macos-ci-leg/` -> `specs/archive/PLAT-001d-macos-ci-leg/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
