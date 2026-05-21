---
id: "REFACTOR-003-diff-check-ps1"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-05-21"
tags: [spec, proposal]
template_version: "1.0"
---

# REFACTOR-003-diff-check-ps1

> **Naming**: file lives at `<repo>/specs/REFACTOR-003-diff-check-ps1/proposal.md`. `REFACTOR-003-diff-check-ps1` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: *(P2, queued 2026-05-21 from WIN-001 SKIP rationale)*: Port `scripts/diff-check.sh` to PowerShell (`scripts/diff-check.ps1`) so `healthcheck.ps1` section 12 (Repo ↔ Deploy-Dir Drift) stops emitting `SKIP: diff-check.ps1 not implemented` and reaches full cross-OS parity. **Why:** `diff-check.sh` walks files tracked by git, compares each against its counterpart in `~/.dotfiles/`, and reports divergences (the drift-detection complement of `doctor.sh`). It's wired into `healthcheck.sh` as section 12/12 and reachable via the `dch` alias on Linux. Without a `.ps1` sibling, the Windows healthcheck has a permanent 1-section blind spot exactly where regressions land most (setup → deploy drift). **Surface (~180-220 LOC + 11+ bats asserts):** mirror the `.sh` structure (allowlist filtering tied to what `setup-windows.ps1` deploys, per-file `Compare-Object` or hash diff, summary with divergence count + suggested resync command). Alias `dch` in `powershell/profile.ps1`. Bats parity asserts following the `knowledge-crystallize-ps1.bats` / `obs-cli-ps1.bats` pattern (PSScriptAnalyzer + structural greps). Update `healthcheck.ps1` sec 12 from `SKIP` to actual invocation. **Anti-scope:** do NOT extend the allowlist (cross-OS divergence in what's deployed is a separate concern). Independent of WIN-001 — does not block its merge; closes one of WIN-001's known SKIPs after the fact. -->
Single paragraph. The user or business problem this feature solves. Link to the vault roadmap or `11-tasks.md` entry if applicable. If you cannot write this in 3 sentences, you do not understand the problem yet.

## What

Concrete behavior change. What does the system do after this PR that it did not do before? Observable, not implementation-focused.

## Out of scope

Things this PR explicitly does NOT include. Forces a sharp boundary and prevents scope creep.

-
-

## Risks / open questions

Failure modes, dependencies, and unknowns to clarify before implementation. If any item here is unresolved, do not move to `tasks.md` yet.

-
-

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] Outcome 1
- [ ] Outcome 2
- [ ] Outcome 3

## References

- Vault: `10_projects/<repo>/11-tasks.md` (backlog entry)
- Related ADR: `<area>/30-architecture/adr-XXX.md` (if any)
- Related patterns: `00_meta/patterns/<pattern>.md` (if any)
