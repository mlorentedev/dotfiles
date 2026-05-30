---
tags: [spec, verification, sdd-008]
created: "2026-05-26"
updated: "2026-05-29"
---

# Verification - SDD-008-skill-pipeline

> **Partial — Phases 0-3 done, 4-6 pending.** Machine-readable contract: `features.json` (8 entries, one per AC). Phases 4 (vault migration) and 6 (vault pattern doc) are gated on parallel-session vault activity; Phase 5 (setup wiring + de-symlink) follows the migration. The PR opens after Phase 6.

## Evidence (per AC)

- [x] **AC3** (`--check` detects rendered-skill drift, offline) -> `bats tests/compile-harness.bats -f 'render: --check'` (2 green) — clean tree passes; tamper + offline (no vault) detected.
- [x] **AC4** (provenance) -> `bats … -f 'render: --refresh renders'` — `generated_from` + `generated_sha:` on claude/opencode outputs. **Reconciled:** provenance as `generated_*` frontmatter fields (skill/command) or leading comment (prompt) — a leading comment would break the YAML frontmatter agents parse.
- [x] **AC5** (frontmatter schema) -> `bats … -f 'schema:'` (2 green) — missing `name` + unterminated frontmatter fail with file context. **Reconciled:** required = `name`+`description` (universal), not the proposal's `id`/`type`/`allowed-tools` (would fail all 17 repo skills).
- [x] **AC6** (per-skill targets[] + agy + copilot catalog) -> `bats … -f 'targets|catalog|agy'` — opt-out honored; agy native skill + flat prompt; copilot marker-injected catalog.
- [ ] **AC1** (zero symlinks in deployed paths) — Phase 5, pending.
- [ ] **AC2** (single SSOT; 17 skills migrated to vault) — Phase 4, **gated** on parallel-session vault activity (irreversible cross-repo).
- [ ] **AC7** (vault `pattern-cross-agent-skill-pipeline.md`) — Phase 6, pending (vault write).
- [ ] **AC8** (post-deploy smoke: /spec discoverable) — Phase 6, depends on Phases 4-5.

## Test status

- `bats tests/compile-harness.bats` -> **20/20** (12 ENGINE-001 + 8 SDD-008 render/schema/catalog). Run targeted, NOT `tests/*.bats`.
- `shellcheck scripts/compile-harness.sh` clean; `bash -n` pass.
- No regression: real-repo `compile-harness.sh --check` still green (the real manifest has no skills block yet, so existing enforced/targets behavior is unchanged).

## Decisions made during implementation

- **Consume ENGINE-001, not a separate render-all.sh** — skills are a new manifest `kind: render` (whole-file transform) alongside `kind: native|pointer` (marker injection). One engine, one `--refresh`/`--check`/healthcheck surface. Reconciled the pre-ENGINE-001 proposal (`## Engine` section).
- **Committed source-of-record** (`harness/skills/<name>/SKILL.md`) makes `--check` offline → resolves the proposal's original "CI can't see the vault" risk (Phase 2.6 idempotence gate no longer needed).
- **Provenance in frontmatter, not a leading comment** (AC4 reconciliation, above).
- **Schema required set = name+description** (AC5 reconciliation, above).
- **copilot = catalog**, not per-skill files (no per-skill mechanism); **agy = native skill + flat prompt** (verified against setup-linux.sh:443-456).

## Promotion candidates

- [ ] **Lesson?** Maybe — "reconcile a pre-existing spec to a newly-shipped engine before implementing" (the ENGINE-001 → SDD-008 reconciliation). Defer until archive.
- [ ] **ADR-worthy?** No new ADR — ADR-013 (deploy engine) already covers the substrate; `kind: render` is an extension within it.
- [ ] **New `00_meta/patterns/` pattern?** **Yes, planned** — `pattern-cross-agent-skill-pipeline.md` is AC7 (Phase 6).

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/SDD-008-skill-pipeline/` -> `specs/archive/SDD-008-skill-pipeline/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
