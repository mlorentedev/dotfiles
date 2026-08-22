---
id: "AI-023-orca-ade-and-iac-doctrine"
type: spec
status: archived
created: "2026-08-22"
issue: "mlorentedev/dotfiles#1175"
tags: [spec, proposal, orca, iac, doctrine]
template_version: "1.0"
---

# AI-023-orca-ade-and-iac-doctrine

## Why

Orca ADE is used as a parallel agent orchestrator on centralized Dev Nodes (like `ace2`), but previously lacked a first-class overlay in dotfiles (`ai/orca/`), a standardized repo worktree lifecycle template (`orca.yaml`), and automated SourceControlAI governance prompts enforcing our human-authored conventional commit, branch naming, and PR standards. Furthermore, cross-agent doctrine lacked a hard non-negotiable rule forbidding ad-hoc manual server operations and enforcing strict Infrastructure as Code (IaC) idempotence.

## What

1. Add `ai/orca/` overlay directory with:
   - `ORCA.md`: Agent constitution pointer with injected doctrine.
   - `orca.yaml`: Canonical polyglot worktree setup and archive lifecycle template.
   - `governance.json`: SourceControlAI prompt recipes enforcing conventional commits, phase-free branch names, and structured PRs.
2. Embed `orca.yaml` into `cli/internal/initrepo/templates/orca.yaml` for `dotf init-repo`.
3. Add `iac-and-idempotence` rule to `harness/manifest.json` from `pattern-git-workflow.md#12-infrastructure-as-code-zero-manual-operations-policy` and inject it into all target agents.

## Out of scope

- Direct binary distribution of Orca Desktop.
- Modifying non-Orca IDE configurations.

## Risks / open questions

- None. Fully compatible with existing harness deployment and test suites.

## Acceptance criteria

- [x] [AC1] `ai/orca/ORCA.md`, `ai/orca/orca.yaml`, and `ai/orca/governance.json` exist and are tracked.
- [x] [AC2] `cli/internal/initrepo/templates/orca.yaml` is tracked and passes embed tests.
- [x] [AC3] `harness/manifest.json` declares `iac-and-idempotence` and `compile-harness.sh --check` passes without drift.
- [x] [AC4] Full test suite (bats + Go tests) passes.

## References

- Issue: mlorentedev/dotfiles#1175
- Vault Pattern: `00_meta/patterns/pattern-git-workflow.md#12-infrastructure-as-code-zero-manual-operations-policy`
- Runbook: `00_meta/runbooks/runbook-orca-remote-cde-host-setup.md`
