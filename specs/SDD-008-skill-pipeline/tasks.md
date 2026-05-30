---
tags: [spec, tasks, sdd-008]
created: "2026-05-26"
updated: "2026-05-30"
---

# Tasks - SDD-008-skill-pipeline

> TDD order. Implemented as **option A** (committed records + render-at-deploy), consuming the ENGINE-001 engine (`compile-harness.sh` + `harness/manifest.json`) via a `skills` block — NOT a separate `render-all.sh`. Full-migration scope (no partial) per the Q3 directive. All phases done; PR open.

## Setup

- [x] Branch `feat/sdd-008-skill-pipeline` off main (post-ENGINE-001)
- [x] `proposal.md` reconciled to consume-engine design; CI-blindness risk RESOLVED
- [x] Open questions resolved (bootstrap preflight, drift gate; migration history materialized in Phase 4)

## Phase 1 — Engine: skill records + render-at-deploy (AC3, AC4)

- [x] **(green)** `--refresh` copies vault `00_meta/skills/<n>` → committed record `harness/skills/<n>` (whole dir + aux files); no render, no `$HOME` write
- [x] **(green)** new `--deploy`: render each record to its per-agent `$HOME` path (de-symlink first), prune stale outputs, inject copilot catalog
- [x] **(green)** `--check`: validate each committed record renders cleanly, offline (no vault)
- [x] **(refactor)** shellcheck clean; helpers reused (render_skill / skill_out_path / validate_skill_frontmatter / build_skill_catalog); `bash -n`

## Phase 2 — Frontmatter schema (AC5)

- [x] **(green)** `harness/skill-frontmatter.schema.json` (required = name+description) + validator before record write; tests: missing name, unterminated frontmatter

## Phase 3 — Per-agent renderers + targets[] (AC6)

- [x] **(green)** renderers: claude (skill dir), opencode (command), agy (skill + flat prompt), copilot (catalog); dispatcher honors `targets[]`

## Phase 4 — Atomic migration to vault SSOT (AC2) — CROSS-REPO

- [x] **(green)** `scripts/skills/migrate-to-vault.sh` transactional; ran against the real vault (17 migrated; 22 total)
- [x] `targets:[claude]` added to the 5 Claude Code-only skills in the vault (replaces the skills-to-opencode skip-list)

## Phase 5 — Deploy wiring + de-symlink (AC1)

- [x] **(green)** manifest `skills` block (deploy[] matrix + catalog, `$HOME`-relative dirs); 22 records committed
- [x] **(green)** `setup-linux.sh`: `--refresh` (if vault) + `--deploy`; removed the 5 legacy skill loops (agy prompts/native, claude skills, vault symlinks, opencode commands)
- [x] **(green)** `setup-windows.ps1`: `Deploy-SkillRecords` (PowerShell port of `--deploy`); removed the parallel loops + junction loop
- [x] **(green)** `git rm` ai/skills (README stub), skills-to-opencode.sh, ai/opencode/commands; ci.yml gate → `compile-harness --check`
- [x] healthcheck `check_skills`: `--check` (records render) + on-machine symlink-free assertion

## Phase 6 — Smoke, docs, closeout (AC7, AC8)

- [x] **(green)** round-trip smoke `tests/skills-pipeline.bats`: deployed `/spec` discoverable (claude + opencode + agy)
- [x] `00_meta/patterns/pattern-cross-agent-skill-pipeline.md` (bidirectional cross-link ↔ pattern-spec-driven-development)
- [x] Every AC → ≥1 test; `features.json` states honest (`verifying`, no sham `passing`)
- [x] `verification.md` filled; targeted bats green (NOT `tests/*.bats` — hangs per #167)
- [ ] PR opened referencing this spec + umbrella #162

## Machine-readable features

`features.json` (sibling) per [[pattern-feature-list-as-primitive]]: one feature per AC (AC1–AC8) with an executable `verification`. Only the harness sets `"state": "passing"` after capturing exit 0.
