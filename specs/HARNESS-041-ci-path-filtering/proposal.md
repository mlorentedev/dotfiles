---
id: "HARNESS-041-ci-path-filtering"
type: spec
status: implementing
created: "2026-08-20"
issue: "mlorentedev/dotfiles#552"
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-041-ci-path-filtering

## Why

Every `pull_request` to `main` currently runs the full matrix (`lint`, `lint-powershell`, `test`, `test-windows`, `integration`) with no path filtering. Docs-only PRs (specs, documentation, and root markdown edits) run the entire Go + Bats + Windows + Docker matrix for nothing, wasting 3-4 minutes of CI runner time per PR.

## What

Add a `changes` job using `dorny/paths-filter@v3` at the top of `.github/workflows/ci.yml` providing `code` and `powershell` filter outputs. Guard the expensive steps in all matrix jobs with `if: github.event_name == 'push' || needs.changes.outputs.code == 'true'` (or `powershell` for `lint-powershell`). This ensures all required job names run and report green in ~3-5 seconds on docs-only PRs without violating GitHub branch protection requirements.

## Out of scope

- Direct AI reviewer integration (handled independently by CodeRabbit/PR-Agent).
- Modifying repository branch protection settings.

## Risks / open questions

- **Required check blockage**: If entire jobs are skipped at the job level, strict branch protection rules on GitHub may wait indefinitely for status reports. Mitigated by keeping jobs active and skipping only heavy build/test steps.
- **Embedded templates drift**: Go CLI embeds markdown templates under `cli/internal/*/templates/*.md`. Filtering by code path (`cli/**`) rather than file extension ensures template edits still trigger Go tests.

## Acceptance criteria

- [ ] `changes` job exists in `.github/workflows/ci.yml` using `dorny/paths-filter@v3`.
- [ ] `lint`, `lint-powershell`, `test`, `test-windows`, `integration` depend on `changes`.
- [ ] Heavy steps in all matrix jobs carry conditional guards.
- [ ] Regression suite `tests/ci-path-filtering.bats` passes.

## References

- Issue: https://github.com/mlorentedev/dotfiles/issues/552

