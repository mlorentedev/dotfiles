---
id: "CI-017-gate-required-check"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-12"
issue: "mlorentedev/dotfiles#3306"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, ci, gate]
template_version: "1.0"
owner: manu
---

# CI-017-gate-required-check

> **Naming**: file lives at `<repo>/specs/CI-017-gate-required-check/proposal.md`. `CI-017-gate-required-check` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

`test-windows` is a required check, so it cannot carry a job-level `if:`: a required job skipped at job level passes without running, and CI-004's checker (`tests/lib/check-workflow-contexts.py`, rule 4) rejects it. CI-013 therefore left a Windows runner starting on every documentation-only pull request just to skip its steps. ci-workflows ADR-001 decision 3 removes the constraint: one aggregate check, run under `always()`, is what protection requires, and the jobs behind it may be skipped. Tracked in #3306, part of mlorentedev/ci-workflows#8.

## What

PR 1 (this spec's implementation): `ci.yml` gains a `ci-gate` job that runs under `always()`, needs every other job in the workflow, and judges them with the released ci-workflows `gate` composite pinned by SHA. It reports beside the current required checks and changes nothing about what blocks a merge.

PR 2 (owner, after `ci-gate` has reported on `main`): `forge/branch-protection.json` requires `ci-gate` in place of `changes`, `lint`, `lint-powershell`, `test` and `test-windows`; `cli-gate`, `from-zero-gate`, `knowledge-gate`, `spec-gate` and `review-attestation` stay. Then `test-windows` takes a job-level `if:` from `changes`, and the fleet rule `fleet/os-on-pull-request` stops reporting it.

## Out of scope

- Folding the five pull-request workflows into one so a single `gate` covers them all (open question below).
- `cli.yml`'s `test` and `cli-smoke.yml`'s `smoke` Windows legs: CI-013 already gates them.
- Amending ADR-001's literal check name.

## Risks / open questions

- **The name is `ci-gate`, not `gate`.** `spec-gate.yml` has a job with id `gate` and no `name:`, so it already reports `gate`; CI-004 rule 1 rejects a second reporter. ADR-001 decision 3 says `gate`; whether to rename spec-gate's job and amend the ADR is the owner's.
- **Acceptance criterion 2 of #3306 ("gate and review-attestation only") is unreachable** while dotfiles has five pull-request workflows: `needs` cannot cross a workflow. PR 2 keeps one aggregate per workflow. Folding them is an owner decision.
- **PR 2 makes `integration`, `test-macos` and `pi-nan-package` blocking.** CI-004 rule 3 makes `ci-gate` need every job in `ci.yml`, and none of the three is required today. `integration` failed on every branch during the Azure mirror outage of 2026-11-11. The owner decides at PR 2 whether that is wanted; if not, those jobs move to their own workflow first.
- The composite needs `jq` on the runner; `ubuntu-latest` has it, and the composite fails closed without it.

## Acceptance criteria

- [ ] AC1: `ci.yml` has a `ci-gate` job under `always()` that needs every other job in the workflow and runs the ci-workflows `gate` composite pinned by full SHA; the CI-004 checker passes.
- [ ] AC2: On a pull request, `ci-gate` reports, and fails when any job it needs failed or was cancelled.
- [ ] AC3 (owner, PR 2): branch protection requires `ci-gate` in place of the five `ci.yml` checks, keeping the other workflows' aggregates and `review-attestation`.
- [ ] AC4 (PR 2): `test-windows` skips at job level on a documentation-only pull request, and the CI-004 checker passes.

## References

- Bitácora board: #3306; parent mlorentedev/ci-workflows#8
- Related ADR: mlorentedev/ci-workflows `docs/adr/adr-001-one-ci-standard-for-every-forge.md`, decision 3
- The composite: mlorentedev/ci-workflows `.github/actions/gate`, v0.6.0
