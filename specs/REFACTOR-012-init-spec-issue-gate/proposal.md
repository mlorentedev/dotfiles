---
id: "REFACTOR-012-init-spec-issue-gate"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-06-09"
issue: "dotfiles#304"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# REFACTOR-012: Init spec issue gate

> **Naming**: file lives at `<repo>/specs/<feature-id>/proposal.md`. `<feature-id>` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #304: init-spec.{sh,ps1} still enforces the retired vault 11-tasks.md gate (ADR-018 drift) -->

ADR-018 moved task/work state out of the vault into the bitácora GitHub Project — the vault no longer holds `11-tasks.md`. But `init-spec.{sh,ps1}` still hard-requires a matching `**ID**` entry in `$VAULT_PATH/10_projects/<repo>/11-tasks.md`, exiting 3 otherwise. The *default* path is now wrong by design: every scaffold has to be forced through `--force-no-vault` (hit during OPS-001 #295), which trains users to bypass the gate — the gate stops gating anything.

## What

The work-gate becomes an **open GitHub issue**, matching ADR-018:

- `init-spec.sh <id> --issue <N>` / `init-spec.ps1 <id> -Issue <N>` verifies via `gh issue view <N>` that the issue exists and is OPEN, then scaffolds and injects `<!-- from issue #N: <title> -->` into the proposal's `## Why`.
- Running without `--issue` (and without bypass) fails with exit 3 and guidance — the gate is now the issue, not a vault line.
- The bypass is renamed `--force-no-gate` / `-ForceNoGate`; the old `--force-no-vault` / `-ForceNoVault` remains a working deprecated alias.
- The vault `11-tasks.md` lookup and the `--task` / `-Task` flag are removed. Templates still come from `$VAULT_PATH/00_meta/templates` — only the task gate changes.
- `AGENTS.md` SDD-workflow wording and the vault `/spec` skill SSOT (+ compiled `harness/skills/spec/SKILL.md` record) updated to describe the issue gate.

## Out of scope

- `archive-spec.{sh,ps1}` and the `/spec archive` backlog-tick step (vault `11-tasks.md` tick) — separate drift, separate ticket if confirmed.
- Auto-discovery of the issue number from the feature-id (issue titles don't contain spec ids; explicit `--issue` is the contract).
- Any change to the spec-gate CI workflow (`check-spec-gate.sh`) — it gates PRs on spec folders, not on the vault.

## Risks / open questions

- `gh` availability/auth (offline, CI, fresh machines): gate fails closed with a clear message; `--force-no-gate` is the documented escape hatch. Tests stub `gh` on PATH — no network in bats.
- Closed issues: treated as gate failure (work-gate is an *open* issue per AGENTS.md); message says so explicitly.
- Backward compatibility: `--force-no-vault` kept as alias so existing muscle memory / docs don't hard-break; emits a deprecation note.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] `init-spec.sh ID --issue N` with an open issue scaffolds the spec and injects `<!-- from issue #N: <title> -->` into `## Why` (exit 0).
- [ ] Missing `--issue`, nonexistent issue, or closed issue → exit 3, no spec dir created, message names the issue gate.
- [ ] `--force-no-gate` and legacy `--force-no-vault` both bypass the gate without invoking `gh`.
- [ ] No `11-tasks.md` reference remains in `init-spec.{sh,ps1}`; full bats suite passes with `gh` stubbed.

## References

- GitHub issue: `dotfiles#304` (work-gate per ADR-018)
- Related ADR: `docs/adr/adr-018-de-vault-task-placement.md`
- Related patterns: `00_meta/patterns/pattern-spec-driven-development.md`
