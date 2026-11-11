---
tags: [spec, verification, templates]
created: "2026-11-11"
id: "CI-013-os-matrix-when-it-can-change"
type: spec-verification
status: active
owner: manu
---

# Verification - CI-013-os-matrix-when-it-can-change

## Evidence

- [ ] AC1 (docs-only PR starts no Windows/macOS job but the required `test-windows`) -> pinned by `tests/ci-os-matrix.bats` tests 1-4; live proof is the first documentation-only pull request after merge (this PR touches `.github/**`, so it is code)
- [ ] AC2 (push runs the full matrix) -> pinned by `tests/ci-os-matrix.bats` tests 2-3 (`github.event_name != 'pull_request'` branch yields both OSes); live proof is the push run of the merge commit
- [ ] AC3 (warm integration build takes the pre-COPY layers from cache) -> pinned by `tests/ci-os-matrix.bats` test 6; live proof is the first pull-request run after the merge's push writes the cache. Cold reference: main run 42964129736, build layers 1-11 about 53 s, then go build 22 s, setup-linux.sh 100 s, export 26 s
- [x] AC4 (one docs-only list) -> `.github/filters/code.yml`, read by cli.yml, cli-smoke.yml and `tests/ci-path-filtering.bats`; disjointness from ci.yml's `code` filter in `tests/ci-os-matrix.bats` test 5

## Test status

- `bats $(git grep -l -E 'cli\.yml|cli-smoke\.yml|ci\.yml|filters/code' -- 'tests/*.bats')` -> 1..315 (the list includes tests/ci-os-matrix.bats; naming it twice makes bats plan it twice and exit 1), no failures
- `actionlint` over ci.yml, cli.yml, cli-smoke.yml -> exit 0
- `python3 tests/lib/check-workflow-contexts.py .` -> exit 0
- Mutations: literal OS list in cli-smoke (tests 1, 3 red), `'!tests/**'` added to the filter (test 5 red), push branch dropped from cli's `test_os` (test 2 red), the `test` matrix pointed at an output without the pull-request fallback (test 1 red)

## Decisions made during implementation

- `test-windows` keeps no job-level `if`: CI-004's checker rejects one on a required check. First draft added it; the checker caught it.
- golangci-lint `skip-cache: true` stays (#2834). Declined in the proposal.
- The integration cache is written by the push to main only; pull requests read it.
- `predicate-quantifier: some-with-excludes` rather than `every`: same result for `**` plus exclusions, and it reads as what it means.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-578-a-path-filter-saves-only-what-its-diff-class-costs.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: a CI configuration choice recorded in the proposal; the required-check change it waits on is decided in mlorentedev/ci-workflows
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: the shared gate composite already carries the cross-repository shape
