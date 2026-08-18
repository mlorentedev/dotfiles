---
id: lesson-082-setup-time-compile-harness-sh-refresh-leaves-silen
type: lesson
status: active
created: "2026-06-09"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 082: setup-time `compile-harness.sh --refresh` leaves silent drift -- surface it, don't delete it

**Context:** A live `setup-linux.sh` run left `harness/skills/handoff/SKILL.md` modified in the working tree. It read like a parallel session's WIP and derailed reasoning about an unrelated fast-forward pull (OPS-001, #295) -- time was spent treating a designed signal as mystery dirt.
**Problem:** `setup` runs `compile-harness.sh --refresh` whenever the vault is present, regenerating the committed `harness/` records (and the generated blocks in `AGENTS.md` / `ai/claude/CLAUDE.md`) from the vault SSOT. When the vault is ahead, that leaves uncommitted changes -- but setup only logged a success line, so the drift was invisible-as-intent. The instinct ("just delete the `--refresh` from setup to stop the dirt") is a trap: the dotfiles CI has **zero visibility into the private vault** (ADR-013), so `--refresh` is the *only* thing propagating vault skill edits into the committed records. Removing it = silent vault<->repo divergence -- exactly the bug class SDD-008's "the build IS setup" design exists to prevent.
**Solution:** Treat the drift as a signal, not noise. OPS-003 (#307 -> #308): after a successful `--refresh`, setup checks the working tree and, if records changed, **warns loudly** with the file list + the exact commit command -- turning silent drift into an actionable `chore(harness): refresh records from vault` commit. Keep the mechanism; make it announce. The directionality is the design: vault = authoring SSOT, `harness/` = committed cache for offline deploy + CI. Before deleting any setup step that "dirties" the tree, ask what it propagates -- here it was load-bearing.
**Tags:** `#setup` `#harness` `#compile-harness` `#drift` `#vault-ssot` `#generate-and-commit` `#verify-before-act` `#dont-delete-load-bearing-code`
