---
id: "REFACTOR-007-scripts-dir-env-contract"
type: spec
status: abandoned # draft | implementing | verifying | archived
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
review: waived
review_waived_reason: "Abandoned in owner triage 2026-08-20. The spec required an (a)/(b) decision on the cross-OS SCRIPTS_DIR contract that was never taken, and no env-contract.json was ever created. The underlying path-resolution problem was solved by another route: ADR-025 shipped dotf env, which resolves per-machine paths without this contract."
---

# REFACTOR-007-scripts-dir-env-contract

> **Naming**: file lives at `<repo>/specs/REFACTOR-007-scripts-dir-env-contract/proposal.md`. `REFACTOR-007-scripts-dir-env-contract` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: (formalises Phase 2.7): `setup-windows.ps1` deploys scripts to `$USERPROFILEscripts` and PATHs that, but `env-contract.json` declares `SCRIPTS_DIR = $USERPROFILE.dotfilesscripts`. #109 worked around this for `hc`; root fix is one decision: align setup with contract OR amend contract to reflect Windows reality. Affects PATH, `dch`, `project-init` resolution chains. -->

`setup-windows.ps1` deploys scripts to `$env:USERPROFILE\scripts` and adds that to PATH. But `env-contract.json` declares `SCRIPTS_DIR = $env:USERPROFILE\.dotfiles\scripts`. The two paths drift; PR #109 worked around the inconsistency for the `hc` alias specifically, but the root contract violation persists. Downstream consumers (`dch`, `project-init`, future tools that read `SCRIPTS_DIR`) hit the gap. The Windows-side resolution chain is fundamentally inconsistent with the env-contract that AGENTS.md and the test harness rely on.

## What

A **decision** PR followed by alignment edits. Two valid paths:

- **(a) Move setup to honour contract** — `setup-windows.ps1` deploys to `$env:USERPROFILE\.dotfiles\scripts` (matches Linux convention and env-contract.json). PATH entry updates. Affects `hc`, `dch`, `project-init` invocations; PR #109's workaround removed.
- **(b) Amend contract to reflect Windows reality** — `env-contract.json` updated so `SCRIPTS_DIR` for Windows is `$env:USERPROFILE\scripts` (Linux remains `$HOME/.dotfiles/scripts`). Cross-OS divergence accepted.

PR body documents the decision with rationale. Implementation is whichever (a) or (b) wins.

## Out of scope

- **Other env-contract.json drift** — REFACTOR-002 (archived) extended the contract with `*_HOME` paths; this ticket touches only `SCRIPTS_DIR`.
- **Linux-side changes** — Linux already aligns with contract; no edits there.
- **Migration of existing user state** — if a user's existing `~/scripts` directory contains user files, document the migration path; this PR doesn't auto-move them.

## Risks / open questions

- **R1**: Decision rationale. (a) preserves cross-OS uniformity at the cost of moving the deploy site on every existing Windows install. (b) accepts permanent divergence but avoids relocation churn. The user's Windows daily-driver state matters: how many places hard-code `~/scripts`?
- **R2**: PR #109 workaround. If (a) wins, the workaround disappears; trace any callers that depend on it. If (b) wins, the workaround becomes load-bearing — document.
- **R3**: PATH entry contention. Windows User PATH has a 2048-char limit. Adding/removing an entry near the limit can drop subsequent entries. Mitigate by reading current PATH, performing the swap, writing back atomically.
- **R4**: bats coverage gap. The drift's existence implies no test was asserting cross-OS parity for `SCRIPTS_DIR`. Add one regardless of (a)/(b) choice.

## Acceptance criteria

- [ ] PR body's `## Decision` section picks (a) or (b) with rationale.
- [ ] If (a): `setup-windows.ps1` deploys to `$env:USERPROFILE\.dotfiles\scripts`; PATH updated atomically; PR #109's workaround removed; existing user files migration documented.
- [ ] If (b): `env-contract.json` SCRIPTS_DIR amended for Windows; cross-OS-divergence rationale documented; PR #109's workaround re-documented as canonical.
- [ ] Bats parity test asserts `SCRIPTS_DIR` agrees with deployed path on both OSes.
- [ ] `hc`, `dch`, `project-init` aliases resolve cleanly on Windows post-PR.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` → REFACTOR-007 (formalises Phase 2.7).
- Predecessor: PR #109 (hc workaround).
- Related (archived): REFACTOR-002 (env-contract `*_HOME` extension).
- Pattern: `00_meta/patterns/pattern-env-contract.md` (if exists; else create as part of this PR).
