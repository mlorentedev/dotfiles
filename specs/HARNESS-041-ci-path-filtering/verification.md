---
tags: [spec, verification, templates]
created: "2026-08-20"
---

# Verification - HARNESS-041-ci-path-filtering

## Evidence

- [x] Criterion 1 (`changes` job in `ci.yml`) -> `tests/ci-path-filtering.bats` "HARNESS-041: ci.yml defines a changes job with dorny/paths-filter"
- [x] Criterion 2 (Matrix jobs depend on `changes`) -> `tests/ci-path-filtering.bats` "HARNESS-041: matrix jobs depend on changes job"
- [x] Criterion 3 (Heavy steps conditional) -> `tests/ci-path-filtering.bats` "HARNESS-041: heavy steps carry changes conditional filter"
- [x] Criterion 4 (Regression suite) -> `bats tests/ci-path-filtering.bats` (3/3 pass)

## Test status

- Test suite: `bats tests/ci-path-filtering.bats tests/workflow-timeouts.bats` -> 4/4 pass
- Go test suite: `cd cli && go test ./...` -> all packages pass
- No regressions: yes

## Decisions made during implementation

- **Step-level conditional guards**: Maintained job-level executions while placing `if: github.event_name == 'push' || needs.changes.outputs.code == 'true'` on resource-intensive steps. This guarantees 100% compliance with GitHub branch protection required status checks while reducing docs-only PR runtimes to ~3-5 seconds.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons/`? <yes / no - one line of what>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-041-ci-path-filtering/` -> `specs/archive/HARNESS-041-ci-path-filtering/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
