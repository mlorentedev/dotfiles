---
id: "PLAT-008-workspace-root-contract"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-11-05"
issue: "mlorentedev/dotfiles#3191"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# PLAT-008-workspace-root-contract

> **Naming**: file lives at `<repo>/specs/PLAT-008-workspace-root-contract/proposal.md`. `PLAT-008-workspace-root-contract` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3191: PLAT-008: one project workspace boundary for repositories and worktrees -->

Repository moves currently require updating independent absolute paths. When WORKTREE_ROOT was left behind, worktrees were moved manually, splitting Git metadata from their directories and hiding recoverable work from coding agents. Without one per-machine workspace boundary and Git-safe reconciliation, every OS or agent can recreate the drift, break resumable sessions, or prune work that still exists.

## What

A machine declares one PROJECTS_PATH workspace boundary, and repository, managed-worktree, trust, and persisted-environment paths converge from it while explicit exceptions remain supported. A plan-first worktree reconciler inventories broken registrations, repairs recoverable moved worktrees in place, and relocates only safe checkouts through Git. Doctor and worktree creation fail loudly on hidden roots, stale generated values, ambiguous or orphaned metadata, and unsafe Windows path budgets.

## Out of scope

Things this PR explicitly does NOT include. Forces a sharp boundary and prevents scope creep.

- Writing GitHub Copilot Desktop's internal database, creating junction aliases, or promising one exact root for unsupported native worktree commands.
- Automatically deleting or guessing the branch/base of the unrepairable kubelab orphan; rescue requires an explicit operator decision.
- Fixing BUG-164 (#3193) or changing Copilot telemetry policy; that startup defect remains an independent change.

## Risks / open questions

Failure modes, dependencies, and unknowns to clarify before implementation. If any item here is unresolved, do not move to `tasks.md` yet.

- Rollout-order risk: `derived_from` support and provenance-aware persistence must ship and be pinned before contract relationships are activated; tests must cover old- and new-binary behavior.
- Destructive-recovery risk: repair requires one unambiguous checkout/admin-record match, while active, locked, dirty, ahead, or path-keyed sessions must be skipped; orphan rescue is never automatic.
- Windows path risk: worktree creation hard-fails at a projected path length of 260 or more and warns when less than 20 characters of headroom remain, regardless of LongPathsEnabled. No unresolved blocker remains if this threshold is accepted.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] Given a machine-level PROJECTS_PATH override, all declared derived defaults, generated path files, managed persistent environment values, and agent trust projections resolve consistently; compatibility fallbacks remain valid and a second converge reports changed=0.
- [ ] A synthetic manually moved worktree is classified and repaired in place only when one checkout and one administrative record match; prunable content is never discarded, unsafe sessions are skipped with reasons, and an orphan returns a terminal rescue-required result.
- [ ] A sound, clean, idle worktree relocates through Git into WORKTREE_ROOT with branch, content, registration, and lease preserved; the second reconciliation plan has no repair or move action.
- [ ] Doctor and worktree creation fail on linked checkouts hidden under excluded/dot-prefixed roots, workspace-boundary violations, stale generated environment values, ambiguous metadata, and projected Windows paths >=260; creation warns below 20 characters of headroom.
- [ ] Every repair, orphan, relocation, and path-budget test uses synthetic repositories and redirected HOME/state; no test or migration probe touches live user worktrees.
- [ ] Reconciliation plan mode is side-effect-free and exposes stable machine-readable classifications for coding agents, with parity tests on Windows and POSIX paths.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related ADR: `docs/adr/adr-057-project-workspace-boundary-and-worktree-roots.md`
- Related patterns: `pattern-contract-defaults-per-machine-override`, `pattern-setup-script-idempotence`, `pattern-verification-fails-toward-unproven`
