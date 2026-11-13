---
tags: [spec, verification, templates]
created: "2026-11-12"
id: "CI-017-gate-required-check"
type: spec-verification
status: active
owner: manu
---

# Verification - CI-017-gate-required-check

## Evidence

- [x] AC1 -> PR 1, test `ci.yml: ci-gate runs under always(), needs every other job, and uses the pinned gate composite` in `tests/workflow-job-names.bats`, and `tests/lib/check-workflow-contexts.py` (rule 3)
- [x] AC2 -> PR 1's own CI run on 6929dd90: `ci-gate` reported success with its eight needs green (#3387). The failure path is the composite's, covered by ci-workflows `tests/gate.bats`
- [ ] AC3 -> PR 2 declares it (`forge/branch-protection.json`); `dotf forge protection check --unmerged` shows only the expected dotfiles drift. Closes when the owner applies it after merge
- [x] AC4 -> PR 2: job-level `if:` on `test-windows`, held by `tests/ci-os-matrix.bats` (CI-013, which exempts only required checks) and `tests/ci-path-filtering.bats` (HARNESS-041, which now accepts a job-level guard on the right output)

## Test status

- PR 1: `bats tests/workflow-job-names.bats` -> 1..7, no failures; `python3 tests/lib/check-workflow-contexts.py .` -> exit 0; `actionlint .github/workflows/ci.yml` -> clean
- PR 1 mutations, each red: `integration` dropped from `needs` (the new test and the checker), `if: always()` removed (the new test)
- PR 1: `dotf lint fleet --dir .` reports the same four non-checkout findings as main; `ci-gate` adds none
- PR 2: every bats file naming `test-windows`, `branch-protection` or `ci.yml` -> 1..308, no failures; checker exit 0; `actionlint` clean
- PR 2 mutation: `test-windows`' job-level `if:` removed -> CI-013 and HARNESS-041 both red
- PR 2: `dotf lint fleet --dir .` no longer reports `fleet/os-on-pull-request`; the three left are #3336's (`cli.yml` skip-cache x2, `repo-hygiene.yml` cancel-in-progress)

## Decisions made during implementation

- Named `ci-gate`: `spec-gate.yml`'s job `gate` already reports `gate` (CI-004 rule 1).
- The composite's tag, v0.6.0, is pinned as commit `2355330358344bb08c0b7a305be4ed6488f23d0c`, the fleet's sha-pin rule.
- PR 2 drops `test-windows`' 13 step-level guards, which the job-level `if:` makes redundant, and reduces the two compound ones to their remaining condition (`failure()`; `push` and a cache miss).
- `integration`, `test-macos` and `pi-nan-package` become blocking through `ci-gate` (owner, 2026-11-12). Last 32 runs on main: `integration` failed 1, the other two 0. A job that turns flaky is quarantined with a ticket, not left non-blocking.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CI-017-gate-required-check/` -> `specs/archive/CI-017-gate-required-check/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
