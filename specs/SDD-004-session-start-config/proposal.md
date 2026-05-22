---
id: "SDD-004-session-start-config"
type: spec
status: draft
created: "2026-05-21"
tags: [spec, proposal, sdd-004, cross-os, session-start, ssot, audit-002-followup]
template_version: "1.0"
---

# SDD-004: Session start config

> **Naming**: file lives at `<repo>/specs/SDD-004-session-start-config/proposal.md`.

## Why

<!-- from 10_projects/dotfiles/11-tasks.md: Extract session-start-config.json for claude-session-start.{sh,ps1} (highest-frequency cross-OS pair, 937 LOC combined). -->

[AGENT-DRAFT — review before archive]

`claude-session-start.{sh,ps1}` is 937 LOC combined (497 sh + 440 ps1) and is the highest-change-frequency cross-OS pair in the repo — it gains a new injector almost every sprint (BUG-003, SDD-001, SDD-021, the claude-mem heal cascade, etc.). Every change today is a two-place edit, so probe-order or injection-layout drift between Linux and Windows is invisible until a real Claude session prints a malformed SessionStart bundle on the wrong OS. If we don't ship this, the drift surface keeps growing and every new injector is a coin-flip on whether Windows parity will be remembered. AUDIT-002 ranks this pair as the #1 SSOT-extractable candidate; `doctor.{sh,ps1}` + `env-contract.json` is the reference pattern already proven in-repo.

## What

[AGENT-DRAFT — review before archive]

A new `session-start-config.json` at repo root becomes the SSOT for: (1) the ordered list of injectors that run on SessionStart, (2) each injector's probe command + content source, and (3) on-failure behavior (skip vs warn). `claude-session-start.{sh,ps1}` are refactored into thin readers (~787 LOC combined, -150 LOC target) that parse the JSON via `jq` (Linux) / `ConvertFrom-Json` (Windows) and dispatch. Adding a new injector becomes a one-place JSON edit instead of a two-file shell edit, and `git diff` on the config will be the canonical record of what changed for any future SessionStart enhancement.

## Out of scope

[AGENT-DRAFT — review before archive]

- Refactoring other AUDIT-002 SSOT-extractable candidates (`knowledge-crystallize.{sh,ps1}`, `dotfiles-sync.{sh,ps1}`) — deferred per AUDIT-002 §"SSOT-isation priority list" (rank 2-3).
- Touching the `load-secrets.{sh,ps1}` 0.24-ratio anomaly — separate BUG-006 cross-OS-completeness audit, not an SSOT-isation target.
- Changing the SessionStart hook output contract itself. Injection layout, ordering, and content MUST stay byte-identical to pre-refactor for all existing injectors; only the *source of truth for what gets injected* moves.

## Risks / open questions

[AGENT-DRAFT — review before archive]

- **R1 (BLOCKER — must resolve before code):** Byte-equivalence regressions. SessionStart fires on EVERY Claude session; any output difference breaks all subsequent sessions silently. The first task MUST be capturing golden output (one per OS) BEFORE any refactor, and the implementation MUST assert byte-identity post-refactor. This is a non-negotiable invariant.
- **R2:** PowerShell ASCII-only rule (pattern-powershell-ascii-only — hit twice already: Mar 2026 + PR #36 May 2026). The JSON file itself MUST be ASCII-only; any em-dash / smart-quote sneaking into a probe message or comment via JSON triggers PSScriptAnalyzer non-ASCII fail when the ps1 reads it.
- **R3 (open question — resolve before tasks.md freeze):** Schema shape. Two candidates: (a) flat list of injector objects `[{id, probe, content_source, on_failure}, ...]`, or (b) nested by category (`{vault: {...}, claude_mem: {...}, sdd_reminder: {...}}`). AUDIT-002 explicitly warns "keep schemas minimal" (single `jq` / `ConvertFrom-Json` call). Recommendation: flat list (a) for parser simplicity, but needs explicit decision.

## Acceptance criteria

[AGENT-DRAFT — review before archive]

- [ ] `session-start-config.json` exists at repo root, is valid JSON, and is schema-locked by a bats test that parses it via `jq` and asserts every injector entry has the agreed fields.
- [ ] `claude-session-start.sh` and `claude-session-start.ps1` both read the config and emit SessionStart output that is **byte-identical** to a pre-refactor golden file (one golden captured per OS before any code change).
- [ ] Total LOC of `claude-session-start.{sh,ps1}` drops ≥100 LOC vs pre-refactor (audit target: ~150 LOC saved). Counted via `wc -l` on the two files combined.
- [ ] Changing a threshold value in `session-start-config.json` (e.g., `memory_md_max_lines: 150 → 1`) affects `claude-session-start.sh` behavior on the next run with NO code edit — proven by a bats parity test (cross-OS drift detector in `tests/session-start-config.bats`). Note: the original AC envisioned full per-injector dispatch ("add new injector = JSON-only edit") but verify-before-act on the actual script revealed 12 injectors with bespoke logic each (date math, slugify, file enumeration). That dispatcher-level refactor is deferred to a hypothetical SDD-004-v2. This PR (v1-tight) extracts only the 6 thresholds + provides `cfg_injector_enabled` helper for future incremental gating.

## Completeness review

[AGENT-DRAFT — review before archive]

Standard items considered:
- Rate limit / cost guard — N/A (local hook, no external API).
- Idempotency — already guaranteed; probes are read-only.
- Regression test — covered by R1 byte-equivalence assertion + 4th acceptance criterion.

Adding (not in template, but load-bearing here):
- **Migration plan:** for the existing 7 injectors documented in AUDIT-002, each one needs a 1-line entry in the JSON + a corresponding golden assertion. PR cannot land until all 7 are migrated.
- **Rollback plan:** config + both readers ship in the same PR. `git revert` on the single merge commit atomically restores pre-refactor `.sh`/`.ps1` AND removes the JSON. No partial-state failure.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` (backlog entry #7, P1)
- Related ADR: `10_projects/dotfiles/30-architecture/audit-002-cross-os-duplication.md` (surfaces SDD-004 as top candidate)
- Related pattern: `10_projects/dotfiles/30-architecture/audit-002-cross-os-duplication.md` reference `doctor.{sh,ps1}` + `env-contract.json`
