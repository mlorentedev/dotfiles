---
tags: [spec, verification, templates]
created: "2026-11-05"
id: "BUG-165-doc-path-lint-engine"
type: spec-verification
status: active
owner: manu
---

# Verification - BUG-165-doc-path-lint-engine

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [ ] Criterion 1 -> tests `TestCheckFilesPathOutcomes`, `TestCheckFilesVaultPaths`, `TestCheckFilesRejectsBareSourceCommand`, `TestDiscoverGovernedFiles`
- [ ] Criterion 2 -> `TestCurrentRepository` measured 0.29 s in-process; warm dev CLI measured 0.261 s
- [ ] Criterion 3 -> `TestLintDocPathsReportsFindingsAndCleanFiles`; diff guard confirms shell hook and BATS remain unchanged

## Test status

- Test suite: `go test ./internal/docpaths` -> PASS
- Command tests: focused `go test ./internal/cmd -run TestLintDocPaths` -> PASS
- Vet: `go vet ./internal/docpaths ./internal/cmd` -> PASS
- Manual smoke: dev binary scanned all 14 governed files with exit 0; warm run 261 ms
- Commit hook: PR-1 commits use bounded `SKIP=check-doc-paths` because #3199 is the defect being replaced; the dev Go command is run explicitly against the full repository before each push. No permanent hook configuration changes in PR 1.
- No regressions in existing test suite: pending CI

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- Performance alone did not choose Go; an optimized shell could also be fast. ADR-020 ownership and retiring incident-encrusted dual-shell logic are the architectural reasons.
- `cli.yml` already runs Go tests on every pull request with no paths filter, so PR 1 needs no workflow edit for current-repository coverage.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-165-doc-path-lint-engine/` -> `specs/archive/BUG-165-doc-path-lint-engine/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
