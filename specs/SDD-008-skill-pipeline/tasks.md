---
tags: [spec, tasks, sdd-008]
created: "2026-05-26"
updated: "2026-05-29"
---

# Tasks - SDD-008-skill-pipeline

> TDD order. One task = one focused commit. Reconciled 2026-05-29 to **consume the ENGINE-001 engine** (`compile-harness.sh` + `harness/manifest.json`) via a new `kind: render` target — NOT a separate `render-all.sh`. Full-migration scope (no partial) per the Q3 directive.

## Setup

- [x] Branch `feat/sdd-008-skill-pipeline` off main (post-ENGINE-001)
- [x] `proposal.md` reconciled to consume-engine design; CI-blindness risk RESOLVED
- [ ] No open questions left in `proposal.md` "Risks / open questions"

## Phase 1 — Engine: `kind: render` (AC3, AC4)

- [ ] **(red)** bats: a `kind: render` target writes an agent-native output from a vault `SKILL.md` with a `GENERATED … sha256:` provenance header
- [ ] **(green)** add render path to `compile-harness.sh`: read `kind`, dispatch native/pointer → existing inject, render → whole-file transform
- [ ] **(red)** `--check` exits 0 on a freshly rendered tree; non-zero after a rendered output is hand-edited (offline, no vault)
- [ ] **(green)** `--check` for render targets: recompute output from source-of-record, diff vs deployed
- [ ] **(refactor)** shellcheck clean; helpers extracted; `bash -n` + `zsh -n`

## Phase 2 — Frontmatter schema (AC5)

- [ ] **(red)** bats: malformed/missing `id`/`type`/`description`/`allowed-tools` fails render with file:line
- [ ] **(green)** `00_meta/templates/skill-frontmatter.schema.json` + validator step before render

## Phase 3 — Per-agent renderers + targets[] (AC6)

- [ ] **(red)** bats: `targets: [claude]` renders ONLY to `~/.claude/skills/<n>/SKILL.md`; default = all four
- [ ] **(green)** renderers: claude (SKILL.md copy+header), opencode (command `.md`), agy, copilot (generated section); dispatcher honors `targets[]`

## Phase 4 — Atomic migration to vault SSOT (AC2) — CROSS-REPO

- [ ] **(red)** bats: `migrate-to-vault.sh` is transactional — simulated mid-move failure restores the pre-migration snapshot, exits non-zero
- [ ] **(green)** `scripts/skills/migrate-to-vault.sh`: snapshot → move 17 `ai/skills/<n>` → `vault/00_meta/skills/<n>` → stub/remove `ai/skills`; history-loss handled per chosen option
- [ ] **CONFIRM with user before running against the real vault** (irreversible, separate private repo)

## Phase 5 — Deploy wiring + de-symlink (AC1)

- [ ] **(red)** bats: post-render, `find ~/.claude/skills ~/.config/opencode/commands -type l` is empty (fixture HOME)
- [ ] **(green)** `setup-linux.sh` + `setup-windows.ps1` invoke render; deployed skill paths are regular copies; replace existing vault symlinks
- [ ] healthcheck `check_skills` drift assertion (offline `--check`)

## Phase 6 — Smoke, docs, closeout (AC7, AC8)

- [ ] **(red/green)** round-trip smoke: deployed `/spec` discoverable (claude + opencode CLI surface)
- [ ] `00_meta/patterns/pattern-cross-agent-skill-pipeline.md` (links ↔ pattern-spec-driven-development)
- [ ] Every AC → ≥1 test; `features.json` emitted (states `verifying`, no sham `passing`)
- [ ] `verification.md` filled; targeted bats green (NOT `tests/*.bats` — hangs per #167)
- [ ] PR opened referencing this spec + umbrella #162

## Machine-readable features

Emit `features.json` (sibling) per [[pattern-feature-list-as-primitive]]: one feature per AC (AC1–AC8) with an executable `verification`. Only the harness sets `"state": "passing"` after capturing exit 0.
