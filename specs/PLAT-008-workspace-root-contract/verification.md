---
tags: [spec, verification, templates]
created: "2026-11-05"
id: "PLAT-008-workspace-root-contract"
type: spec-verification
status: active
owner: manu
---

# Verification - PLAT-008-workspace-root-contract

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [ ] Criterion 1 -> commits `722c2a60`, `684a6a5b`; tests `TestResolveDerivedPathUsesResolvedBase`, `TestLoadContractRejectsInvalidDerivedPaths`, `TestDerivedDefaultMatchesCompatibilityFallback`, `TestRealContractDerivesProjectWorkspacePaths`, `TestGeneratedDriftChecksOnlyMarkedValues`, `TestCheckPersistedEnv_ByStatus`, `TestCheckPathFilesFailsOnStaleGeneratedProcessValue`, `TestAgentTrustTemplatesUseProjectBoundaryTokens`
- [ ] Criterion 2 -> commit `<hash>` / test `<name>`
- [ ] Criterion 3 -> commit `<hash>` / test `<name>`

## Test status

- Focused feature command: PASS (`internal/env`, `internal/deploy`, changed `internal/doctor` tests)
- Vet: PASS (`go vet ./internal/env ./internal/doctor ./internal/deploy`)
- Full local doctor package: not authoritative on this workstation; unrelated symlink-privilege and domain-account fixtures fail before this change. CI is the required full-suite evidence.
- Commit-hook exception: the spec-evidence-only commit used one bounded `SKIP=check-doc-paths` because the guard has a verified Windows process-explosion defect tracked by #3199. The commit changes no governed instruction file; the bypass is not permitted for implementation commits or as a permanent configuration.
- Manual smoke test: pending the released binary and machine override; no live path or worktree was changed.
- No regressions in existing test suite: pending CI

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- `derived_from` is additive and keeps the absolute default as an older-binary compatibility fallback; arbitrary token interpolation was rejected.
- Generated path files record only values they assign in `DOTF_GENERATED_PATHS`, so doctor can distinguish stale generated process state from an unmarked operator override.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/PLAT-008-workspace-root-contract/` -> `specs/archive/PLAT-008-workspace-root-contract/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
