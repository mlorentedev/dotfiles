---
id: lesson-097-minting-a-new-script-to-delete-four-others-fights-
type: lesson
status: active
created: "2026-06-15"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 097: Minting a new script to delete four others fights a reduce-the-surface goal — remove and ticket-restore, don't extract

**Context**: CLI-014 folds `init-project.sh` + the `init-repo-*.sh` helpers into `dotf init`, executing ADR-021's north star: shrink the per-OS shell-script surface. `init-project.sh` carried a vault-only sub-mode (`--work-sdk <family> <component>` -> writes a `50_work/45-development/…` vault entry, scaffolds no repo). The prior session's plan was to **extract** that mode into a standalone transitional `init-work-sdk.sh` so the capability survived the deletion.

**Problem**: `--work-sdk` is vault work, not repo scaffolding, so it has no place in the `dotf init` orchestrator — that part was right. But "extract it to a new `.sh`" mints a brand-new per-OS script at the exact moment the whole change exists to *delete* four of them. Spending a new artifact to preserve an infrequently-used capability (onboarding a work-SDK component) is a poor trade: a transitional shim that adds surface in order to remove surface, fighting the goal it claims to serve.

**Solution**: **Remove** the mode on both OSes (the Linux `.sh` dies with `init-project.sh`; the Windows `init-project.ps1 -WorkSdk` block plus its `-Family`/`-Component` params are stripped) and **ticket its restoration in the right home** — #388 restores it inside `dotf vault` (ADR-021 step 3: cross-platform Go, no `.ps1` twin). Rejected: "extract" (a transitional artifact fighting the north star) and "port into CLI-014" (scope-bleeds the init flagship into vault concerns).

**Rule**: When a consolidation's whole point is to reduce surface, preserving a misplaced capability by extracting it into a *new* unit is usually net-negative — it adds surface to remove surface. Prefer remove-now + ticket-restore-in-the-correct-home over extract-to-a-transitional-shim. Re-weigh inherited "extract" plans against the north star mid-flight: a strangler's job is fewer units, not relocated ones.
