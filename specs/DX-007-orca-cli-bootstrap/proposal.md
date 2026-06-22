---
id: "DX-007-orca-cli-bootstrap"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-20"
issue: "mlorentedev/dotfiles#462"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# DX-007-orca-cli-bootstrap

> **Naming**: file lives at `<repo>/specs/DX-007-orca-cli-bootstrap/proposal.md`. `DX-007-orca-cli-bootstrap` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #462: DX-007: Provision Orca ADE + CLI via bootstrap (full profile, cross-OS) -->

Orca (Stably AI, MIT — `github.com/stablyai/orca`) is the ADE we use to run parallel agents in isolated git worktrees, and our agent workflows now invoke its bundled `orca` CLI directly. But the bootstrap neither installs Orca nor wires its CLI onto PATH, so on a fresh machine `orca` is simply unavailable — verified: `command -v orca` fails even with the app installed, because the binary ships inside the app (`…\Programs\Orca\resources\bin\orca.cmd`) and the app does not PATH-wire it. Per Standing Order #1 (automate, don't instruct), any workflow we depend on must be reproducibly provisioned, not assumed present.

## What

After this PR, the `full`/optional bootstrap profile installs Orca cross-OS and wires its CLI onto PATH, idempotently:

- A fresh shell resolves `orca` (Windows/Linux) / `orca-ide` (Linux) post-bootstrap, via the env-contract → RC-file seam (ADR-025), with no manual step.
- The Orca app is installed through each OS's scriptable channel (brew cask / AUR / `.deb` / AppImage / silent `.exe`), skipping cleanly when already present.
- `dotf doctor` reports Orca CLI presence.
- The base profile and CI are unaffected when the profile is off (#350 parity).

## Out of scope

- **Orca settings tuning** (DX-005 #436) and the **Copilot PreToolUse hook** (DX-006 #442) — separate concerns, already handled.
- **Headless / CI / server environments** — Orca is a desktop GUI app; provisioning is gated behind the `full` profile and never runs on the base path.
- **App auto-update** — Orca self-updates via electron-updater after first install; the bootstrap only performs the initial install + PATH wiring.

## Risks / open questions

> Unresolved items block the move to `tasks.md`. The first three are genuine design decisions (owner: user).

- **R1 — Windows install mechanics.** `orca-windows-setup.exe` is electron-builder NSIS → per-user install to `%LOCALAPPDATA%\Programs\Orca` (no admin) and supports silent `/S`. **Open:** confirm `/S` runs fully unattended on a clean box and whether any `/D=` dir override is needed; otherwise pin to the default per-user path.
- **R2 — CLI bin discovery (not hardcode).** The PATH target `…\Orca\resources\bin` is app-relative; hardcoding the absolute path is brittle across machines/versions. **Open:** discover the install dir robustly — fixed per-user `Programs\Orca\resources\bin` vs the registry uninstall key (Windows); equivalent for the Linux `.deb`/AppImage layout where the CLI is `orca-ide`.
- **R3 — profile-gating dependency.** This rides on a `full`/optional profile mechanism (#143 DX-001). **Open + unverified:** does that flag already exist in `setup-*`, or must DX-007 define the gating itself? Resolve before tasks.md — it changes the blast radius.
- **R4 — env-contract is the SSOT for PATH.** PATH wiring must go through `env-contract.json` → `paths.{sh,ps1}` (ADR-025, #227), reusing the Obsidian-CLI idempotent pattern (`setup-windows.ps1:~828`), not an ad-hoc PATH append. Mechanical, but non-negotiable.
- **R5 — idempotency detection per OS.** "Already installed" must be detected without launching the GUI: `command -v orca` first, then OS-native checks (`brew list --cask`, `dpkg -l`, Programs-dir / registry presence).

## Acceptance criteria

- [ ] **AC1** — On a clean Windows box, `full`-profile bootstrap installs Orca silently (`orca-windows-setup.exe /S`) and `orca` resolves in a fresh shell; re-running is a no-op. *Verify:* Pester + manual clean-VM smoke.
- [ ] **AC2** — On Linux, `full`-profile bootstrap installs via `.deb`/AppImage (or AUR on Arch) and `orca`/`orca-ide` resolves in a fresh shell; re-run no-op. *Verify:* bats.
- [ ] **AC3** — PATH wiring is emitted through `env-contract.json` → `paths.{sh,ps1}` (not an ad-hoc append); skip when `orca` already resolves. *Verify:* grep generated RC + bats/Pester idempotency assert.
- [ ] **AC4** — Base profile unaffected; `test-windows`/`test-linux` CI green with the profile off. *Verify:* CI (#350 parity).
- [ ] **AC5** — `dotf doctor` reports Orca CLI presence/absence. *Verify:* `cli/internal/doctor` table-driven test.

## References

- Issue: `mlorentedev/dotfiles#462` (work-gate)
- Upstream: `github.com/stablyai/orca` (MIT), official install docs `onorca.dev/docs/install`, release v1.4.80 assets
- Related issues: DX-005 #436 (settings tune), DX-006 #442 (Copilot hook), DX-001 #143 (profiles), OPS-008 #350 (optional-tools CI), REFACTOR-009 #227 (env-contract SSOT)
- Related ADR: `docs/adr/adr-025-cross-machine-path-resolution.md` (env-contract → paths.* seam)
- Existing pattern to mirror: Obsidian-CLI idempotent PATH install in `setup-windows.ps1`
