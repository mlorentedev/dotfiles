---
id: lesson-120-a-delete-ripples-past-the-direct-caller-token-guar
type: lesson
status: active
created: "2026-06-21"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 120: A delete ripples past the direct caller — token guard-greps miss transitive refs, and "orphaned" fixtures can have hidden consumers

**Context**: Deleting `healthcheck.ps1`/`doctor.ps1` + `tests/healthcheck-ps1.bats` (CLI-018 PR-B). A guard test greps the production files for the `(healthcheck|doctor)\.ps1` token.

**Problem**: Two blind spots the guard did not cover. (1) The grep matches the token `healthcheck.ps1` but NOT `tests/healthcheck-ps1.bats` (hyphen, different extension) — so a stale invocation of the deleted bats file survived in `ci.yml`'s bats subset and would have errored the step once the gating step ahead of it was removed. (2) A CI fixture commented "minimal vault tree for healthcheck section 7" looked orphaned by the deletion, but `setup-windows.ps1` itself consumes it: the auto-memory junction deploy is gated on `Test-Path $VaultRoot`, and a stub `obsidian.cmd` on PATH makes setup take its skip-install branch. Deleting on the comment's word would have silently cut end-to-end coverage.

**Solution**: Read the whole CI job, not just the grep hits. Removed the stale `healthcheck-ps1.bats` entry and the genuinely-orphaned eza/zoxide flake-guard step (it only fed the removed diagnostic), but KEPT the vault/obsidian fixtures and corrected their stale comments to name the real consumer (`setup-windows.ps1`).

**Rule**: A token guard-grep covers layer 1 (direct references in the exact filename form). It does not catch transitive references (CI test lists, glob runners) or setup steps that only fed the deleted thing. Before deleting "orphaned" setup, grep its consumers by *capability* (the dir/binary/PATH entry it provides), not just by the comment — a comment documents one original reason, not every later consumer.
