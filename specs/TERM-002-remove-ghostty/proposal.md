---
id: "TERM-002-remove-ghostty"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-06-10"
issue: "dotfiles#281"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# TERM-002: Remove Ghostty

## Why

<!-- from issue #281: TERM-002: Remove Ghostty terminal (config, setup, tests, docs) -->

Ghostty (TERM-001) is no longer used and will not be — the owner has dropped it as a terminal. Keeping its config deploy, healthcheck section, pinned version, and doc surface is dead weight: every setup run deploys a config nobody reads, the healthcheck carries a whole section for it, and docs recommend a workflow that no longer exists.

## What

- Delete `terminals/` (only held `ghostty/config`), `tests/ghostty.bats`, `docs/runbooks/guide-ghostty-setup.md`, and the `GHOSTTY_VERSION` pin.
- Remove the install-check + config-deploy block from `setup-linux.sh` and the Ghostty section from `healthcheck.{sh,ps1}` — sections renumbered 13 -> 12 in both, parity tests updated.
- Strip mentions from living docs (README, opencode READMEs/runbook, architecture map) and active specs (AI-021, REFACTOR-006, REFACTOR-010); reword DX-003-related comments to reference the abandoned investigation instead of the terminal. Historical records (CHANGELOG, archived specs, ADRs/audits, lessons) keep their mentions.
- Abandon spec `DX-003-ghostty-opencode-hang` -> `specs/archive/_abandoned/` (its subject no longer exists). The `--pure` workaround in `oc` wrappers stays (harmless, terminal-agnostic).
- **Companion (user-requested in-PR): yarn managed cross-OS** — `YARN_VERSION` pin in `versions.conf`, npm-pinned guarded install with drift reconciliation in `setup-{linux,windows}`, version-match checks in `healthcheck.{sh,ps1}`, bats coverage.

## Out of scope

- Uninstalling the ghostty binary / removing `~/.config/ghostty` on machines (user-run: `sudo apt remove ghostty; rm -rf ~/.config/ghostty`).
- Rewriting historical records (archived specs, CHANGELOG, ADR/audit snapshots, lessons).
- Removing the `--pure` flag from `oc` wrappers (separate decision; works on any terminal).

## Risks / open questions

- Healthcheck section renumbering is assertion-coupled (cross-OS parity bats count sections) — both scripts and both bats files move 13 -> 12 in the same commit.
- yarn installs via npm global (classic 1.22.x): chosen over corepack for cross-OS uniformity and no Node-version coupling.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] No ghostty reference remains outside `specs/archive/` and `CHANGELOG.md`.
- [ ] `healthcheck.{sh,ps1}` have exactly 12 sections each; parity bats pass.
- [ ] `versions.conf` pins `YARN_VERSION`; both setups install/reconcile yarn pinned (npm-guarded); both healthchecks verify the version match.
- [ ] shellcheck clean (no new findings); full bats suite green.

## References

- GitHub issue: `dotfiles#281` (work-gate)
- Replaces: TERM-001 (archived spec `specs/archive/TERM-001-ghostty-bootstrap/`)
- Abandoned alongside: `specs/archive/_abandoned/DX-003-ghostty-opencode-hang/`
