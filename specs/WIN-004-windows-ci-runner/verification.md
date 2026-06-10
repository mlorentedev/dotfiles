---
tags: [spec, verification, templates]
created: "2026-05-13"
---

# Verification - WIN-004-windows-ci-runner

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (`test-windows` job on `windows-latest`) -> `features.json` f1 grep passes locally (2026-06-09); `.github/workflows/ci.yml` YAML validated with pyyaml `safe_load`
- [x] AC2 (runs `setup-windows.ps1`, exit 0) -> f2 grep passes; step uses `shell: powershell` (PS 5.1) so the BUG-005 re-exec path executes in CI. Exit-0 proof pending first live PR run
- [x] AC3 (runs `healthcheck.ps1`, exit 0) -> f3 grep passes. Enabling fix verified locally: BUG-015 probe gated on `installed_plugins.json`; on this installed box the probe still PASSes (`claude-mem hook path resolves to: ...\13.3.0`), and `bash -n scripts/healthcheck.sh` + PS parser report 0 errors. Exit-0-on-clean-runner proof pending first live PR run
- [x] AC4 (named bats subset) -> f4 grep passes: all 7 files referenced; bats pinned to `versions.conf` `BATS_VERSION` via tarball + `install.sh` under Git Bash
- [ ] AC5 (branch protection required check) -> post-merge, repo admin (see tasks.md "Post-merge")
- [ ] AC6 (wall-time <= 7 min) -> measure on first green PR run (`gh run view <id> --json jobs`)

## Test status

- Static assertions: all `features.json` f1-f5 verification commands exit 0 locally (2026-06-09)
- New parity bats test added: `tests/setup-linux.bats` "BUG-015 hook probe skips when claude-mem was never installed (WIN-004)" -- greps the `claude-mem hook probe n/a` skip branch in BOTH healthchecks
- BUG-023 lock assertions re-verified after the gate refactor (materialize+break form intact, no pipe-to-head)
- Local `healthcheck.ps1` run: 78 passed / 2 failed / 21 skipped -- the 2 FAILs are pre-existing on this box (opencode version drift, fixed by the drift-convergence change pending setup re-run; pi `{env:}` placeholder, pending user age key) and unrelated to this spec
- Live `windows-latest` execution CANNOT be reproduced locally -- the first PR run is the execution verification; iterate there until green

## Decisions made during implementation

- BUG-015 probe FAIL on a never-ran-Claude machine reclassified as SKIP (gated on `installed_plugins.json`, the same record the BUG-014 check uses). Semantically correct beyond CI: a clean box has nothing to resolve.
- CI sandbox re-encrypts `sensitive/nan.api-key.secret.age` with a throwaway age key instead of mocking substitution -- the SDD-009 deploy-time path runs for real in CI.
- Job is PR-only (`if: github.event_name == 'pull_request'`) per proposal R4 (windows minutes bill 2x).
- Pester suites added to the job (beyond the issue's literal scope): `tests/sdd-009-deploy-time-secrets.Tests.ps1` declares "WIN-004 will pick this up in CI" in its header and had never executed anywhere (local box has only Pester 3.4.0).

## Promotion candidates

- [ ] Lesson for repo `docs/lessons.md`? yes -- "tests written for a runner that doesn't exist yet are dead weight: sdd-009 Pester suite sat unexecuted for two weeks until WIN-004 landed" (add at archive time with PR link)
- [ ] ADR-worthy decision? no
- [ ] New pattern candidate for `00_meta/patterns/`? no (single-repo concern)

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/WIN-004-windows-ci-runner/` -> `specs/archive/WIN-004-windows-ci-runner/`
- [ ] GitHub issue #125 closed (built-in workflow moves it to Done on the bitácora board)
- [ ] Promotions above executed (if any)
