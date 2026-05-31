---
id: "SDD-012-tasks-integrity-guard"
type: spec
status: archived
created: "2026-05-31"
tags: [spec, proposal, vault-health, incident-to-guard, backlog-hygiene, sdd-006-sibling]
template_version: "1.0"
---

# SDD-012-tasks-integrity-guard

> **Naming**: file lives at `<repo>/specs/SDD-012-tasks-integrity-guard/proposal.md`.

## Why

The dotfiles backlog (`vault/10_projects/dotfiles/11-tasks.md`, 390 lines) silently drifted: a sprint-planning overview was layered on top of older per-session detailed entries without reconciliation, so **34 ticket IDs now appear 2+ times** across the two hand-maintained views — and the views disagree (e.g. `ENGINE-001` / `POLISH-004` show `[ ]` despite being merged; the count reads 63 open vs ~35 real). This is the exact failure mode of the user's own "incident → guard" rule: a one-time cleanup alone rots again within a few sessions because nothing detects the recurrence. We need an automated guard so duplication / stale-tick drift surfaces every session, not a manual re-tidy. Direct sibling of SDD-006 (`check-md-escapes.sh`), which guards the literal-`\n` markdown-corruption class the same way.

## What

Concrete, observable behavior after this PR:

1. **A standalone guard `scripts/check-backlog-integrity.sh`** (mirrors `check-md-escapes.sh`) takes `<tasks-file>...` and flags, per file: (a) **duplicate ticket IDs** — the same `[A-Z]+-[0-9]+[a-z]?` core ID on 2+ entry lines (the structural root cause: forces "one ID = one entry"), and (b) **status contradictions** — the same ID carrying both `- [ ]` and `- [x]`. Exit `0` clean / `1` issues found / `2` usage. No Obsidian dependency → CI/fixture-testable.
2. **Wired into the `vault.sh` dispatcher** as `vault check-tasks <path>...`, standalone-runnable like its siblings.
3. **A GUI-independent section in `vault-health.sh`** runs the guard over every `10_projects/*/11-tasks.md` in the vault, reporting `pass`/`fail` — so the drift auto-surfaces at SessionStart (where `vault health` already runs), making the prevention visible without anyone opting in.
4. **Sub-ID safety**: `WIN-002` and `WIN-002a` are distinct (the optional single-letter suffix is captured), so legitimate sub-tickets are not false-positives.

## Out of scope

- **The 11-tasks.md consolidation itself** — restructuring to one canonical list + ticking the stale merged entries is a follow-up vault edit (AC5), tracked separately because the vault checkout is currently busy with a parallel session. The guard PR ships the *mechanism*; the consolidation brings the real file to green afterward.
- **"Merged-but-open" cross-referencing** (matching a `[ ]` ID against a merged PR / archived spec) — needs PR/spec cross-ref, brittle; deferred. The dup-ID + contradiction checks already catch the observed drift.
- **The README / docs reorg** — that is AUDIT-006, separate.
- **Grouped one-line entries** (`BUG-008 / BUG-009 / BUG-010` on a single line) — the guard reads the leading ID per entry line; multi-ID lines are a documented limitation, not a target.

## Risks / open questions

- **False positives on sub-IDs** (`WIN-002` vs `WIN-002a`). **Mitigated:** the ID regex captures an optional trailing single letter, so suffixed sub-tickets are distinct IDs. Covered by a fixture test.
- **The guard is local-only** (vault is private; dotfiles CI has no vault access). **Accepted:** consistent with ADR-012/ENGINE-001 — drift guards live in `healthcheck`/`vault-health` (local, SessionStart), not CI. The bats test uses fixtures, so CI still verifies the *logic*.

## Acceptance criteria

- [ ] **AC1 — duplicate IDs flagged**: a fixture tasks file with one ID on two entry lines → `check-backlog-integrity.sh` exits 1 and names the duplicated ID. **Verify:** bats fixture.
- [ ] **AC2 — status contradiction flagged**: a fixture with the same ID as both `- [ ]` and `- [x]` → exit 1 naming the ID. **Verify:** bats fixture.
- [ ] **AC3 — clean file passes + sub-IDs safe**: a fixture where every ID is unique and consistent (incl. `WIN-002` + `WIN-002a` distinct) → exit 0. **Verify:** bats fixture.
- [ ] **AC4 — wired + discoverable**: `vault check-tasks <file>` dispatches to the guard; `vault-health.sh` carries a GUI-independent section invoking it over `10_projects/*/11-tasks.md`. **Verify:** bats (dispatcher exit code) + grep (vault-health section).
- [ ] **AC5 — real backlog reaches green (follow-up, vault)**: after the 11-tasks.md consolidation, `check-backlog-integrity.sh 10_projects/dotfiles/11-tasks.md` exits 0. **Verify:** run on the consolidated file. *(Tracked as the vault-side follow-up; not in the guard PR.)*

## References

- Sibling precedent: `specs/archive/SDD-006-vault-integrity-check/` + `scripts/check-md-escapes.sh` (incident→guard for the `\n` class)
- Dispatcher: `scripts/vault.sh` (REFACTOR-005), `scripts/vault-health.sh`
- Vault: `10_projects/dotfiles/11-tasks.md` (the drifted file); backlog entry SDD-012 to be added when the vault checkout is free
- AGENTS.md Standing Order #4 (clean as you go) + the "incident → guard" feedback guardrail

<!-- archived 2026-05-31 — PR: https://github.com/mlorentedev/dotfiles/pull/183 -->
