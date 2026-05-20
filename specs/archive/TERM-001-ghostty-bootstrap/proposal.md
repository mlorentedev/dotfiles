---
id: "TERM-001-ghostty-bootstrap"
type: spec
status: archived
created: "2026-05-18"
tags: [spec, proposal, terminal, ghostty]
template_version: "1.0"
---

# TERM-001-ghostty-bootstrap

## Why

Ghostty is the GPU-accelerated terminal emulator chosen as primary daily driver during the AI-011-validation session (2026-05-17) to host the opencode TUI, which suffered visible render lag under tmux. The terminal was installed manually (`apt install ghostty` on Ubuntu 26.04), a minimal `~/.config/ghostty/config` was authored on disk (Catppuccin Mocha theme + JetBrainsMono Nerd Font Mono + copy-on-select + `confirm-close-surface`), and JetBrainsMono Nerd Font was fetched and registered manually. **None of that is versioned in the repo** -- if the machine is reformatted or replicated, the work is lost. PR #38 already merged the tmux truecolor passthrough (`xterm-ghostty:Tc` override) but the Ghostty side itself remains ad-hoc. This spec replicates the AI-011 bootstrap pattern for Ghostty: idempotent install on Linux, two-tier config deploy, healthcheck section, bats tests, vault runbook for GUI-only steps.

Backlog entry: `10_projects/dotfiles/11-tasks.md` -> "TERM-001-ghostty-bootstrap" under P1 active sprint. Related ADR: `30-architecture/adr-009-multi-agent-runtime` (Ghostty + opencode + tmux split workflow rationale).

## What

After this PR, on a clean Ubuntu box where `setup-linux.sh` has been run:

1. The `ghostty` binary is installed at the pinned version (`GHOSTTY_VERSION` in `versions.conf`), via `apt install ghostty` -- detect-and-act gating (install only if missing, mirrors BUG-001 pattern).
2. `~/.config/ghostty/config` exists as a deployed copy of `terminal/ghostty/config` (two-tier deploy: repo -> `~/.dotfiles/terminal/ghostty/config` -> destination via `cp` reconcile-not-skip).
3. `ghostty --version` returns the pinned version; `ghostty +validate-config` exits zero.
4. `scripts/healthcheck.sh` section 12/12 verifies binary present, version matches `versions.conf`, config file deployed.
5. `tests/ghostty.bats` asserts config has the expected fields (font-family, theme, `confirm-close-surface`), config is valid for the installed ghostty version, and the version reported matches `versions.conf`.
6. Runbook `~/Projects/knowledge/10_projects/dotfiles/40-runbooks/guide-ghostty-setup.md` documents: GUI-only one-time steps (JetBrainsMono Nerd Font install via wget + `fc-cache`, set Ghostty as GNOME default via `gsettings`), SSH terminfo workaround (`infocmp -x xterm-ghostty | ssh host tic -x -`), and the recommended workflow (Ghostty native splits/tabs for opencode TUI, tmux for shell sessions + `sshmux`).

## Out of scope

- **Cross-distro install paths.** Ubuntu-only (`apt install ghostty`). Arch/Fedora support deferred to a future PR if/when a non-Ubuntu Linux box enters the user's fleet -- premature abstraction otherwise.
- **Automating "set Ghostty as default GNOME terminal".** Modifies GUI desktop state outside the declarative scope of dotfiles. Documented as a one-liner in the runbook (`gsettings set org.gnome.desktop.default-applications.terminal exec ghostty`) for the user to run consciously.
- **Nerd Font automation.** Fonts live in `~/.local/share/fonts/`. The setup script will NOT fetch/unzip a Nerd Font release at install time -- font fetching is heavyweight, version-fragile, and not in scope of a terminal config bootstrap. Runbook documents the manual fetch.
- **Windows side.** WingGhostty (community native fork, v1.3.110 first release 2026-05-09) is too immature (<1 week post-release) for inclusion in `setup-windows.ps1`. Windows continues to use Windows Terminal (native MS, no setup change). A separate TERM-002 may evaluate WingGhostty in 3-6 months.
- **GhostInTheWSL / WSL bridge.** Not relevant: user is on native Linux.
- **Replacing tmux.** tmux stays for shell sessions and `sshmux`. Ghostty native splits/tabs are recommended for opencode TUI only (PR #38 already gives truecolor passthrough so tmux + ghostty + claude/shell remains fine).

## Risks / open questions

- **Docker integration test container:** `apt install ghostty` in the integration test (`tests/Dockerfile.integration`, clean Ubuntu 24.04 -- but Ghostty needs 26.04 LTS for `universe`). If `ghostty` is not in the 24.04 universe archive, the install step will fail loudly. Mitigation: the install block must use the same "warn-not-fail" pattern as `xclip` (log_warning, do not abort the script). Then bats tests inside the container can `skip` when ghostty is absent. **Decision implicit in this proposal:** detect-and-act with warn-not-fail; container without ghostty stays valid.
- **Ubuntu version drift:** Ghostty 1.3.0 is in 26.04 universe today. If the user pins `GHOSTTY_VERSION=1.3.0` and later Ubuntu ships 1.3.1, the version check in healthcheck would warn (not fail). Bump is a one-line PR. Acceptable.
- **Apt requires sudo:** the install step needs `sudo apt install`. The rest of `setup-linux.sh` already uses sudo for things like `apt update` and `apt install` of dev tools, so no new precedent. Still, document in PR description.
- **GPU requirement:** Ghostty 1.3.0 needs OpenGL 4.3+ for GPU rendering. Falls back to software rendering otherwise. Healthcheck does NOT verify GPU capability -- if a user runs on hardware without OpenGL 4.3, ghostty will still work, just slower. Out of scope to detect.
- **`ghostty --version` parser format:** output is `Ghostty 1.3.0-dev+0000000` on the Ubuntu universe build (development tip channel). The version-match check in healthcheck must tolerate the `-dev+HASH` suffix when matching against `versions.conf` `GHOSTTY_VERSION=1.3.0`. Solution: strip suffix when comparing (`echo "$installed" | sed 's/-.*//'`).

## Acceptance criteria

Observable outcomes after merge + a fresh `./setup-linux.sh` run on a clean Ubuntu 26.04 box:

- [ ] `command -v ghostty` resolves to `/usr/bin/ghostty` (or distro-specific path).
- [ ] `ghostty --version | head -1` reports a version whose stripped form matches `GHOSTTY_VERSION` from `versions.conf`.
- [ ] `cat ~/.config/ghostty/config` shows the deployed canonical config (font-family JetBrainsMono Nerd Font Mono, theme `Catppuccin Mocha`, `confirm-close-surface = true`).
- [ ] `ghostty +validate-config` exits 0 (no theme typos, no schema errors).
- [ ] `~/.config/ghostty/config` is a regular file with identical content to `~/.dotfiles/terminal/ghostty/config` (two-tier sync verified by `cmp`).
- [ ] `scripts/healthcheck.sh` reports a new section (likely 12/12) "Ghostty" with three OK lines: binary present, version match, config deployed.
- [ ] `tests/ghostty.bats` passes with at least 6 assertions: config exists, config has font-family line, config has theme line, config has confirm-close-surface line, parses cleanly with `ghostty +validate-config` (skip if binary absent), version reported matches versions.conf (skip if binary absent).
- [ ] `bats tests/` full suite still green (no regression in existing 200+ tests).
- [ ] CI `lint`, `lint-powershell`, `test`, `integration` all green. `integration` container without ghostty available emits a clean warning, no failure.
- [ ] Runbook `40-runbooks/guide-ghostty-setup.md` exists in the vault with three sections: (a) GUI one-time steps (Nerd Font, set-as-default), (b) SSH terminfo workaround, (c) recommended workflow alongside tmux.
- [ ] Backlog entry in `11-tasks.md` updated to mark TERM-001 as `[x]` with link to merge commit; AI-015 umbrella unchanged.

## Design decisions locked (2026-05-17 review)

| # | Decision | Choice | Rationale |
|---|---|---|---|
| 1 | Version pinning | **Pin in `versions.conf`** (`GHOSTTY_VERSION=1.3.0`) | Coherent with `UV_VERSION`, `BATS_VERSION`, etc. Healthcheck can assert exact match. Reproducible cross-machine. Single source of truth. |
| 2 | Distro coverage | **Ubuntu only** (`apt install ghostty`) | KISS. Only distro in user's current fleet. Cross-distro adapter is premature abstraction; cheap to add later if needed. |
| 3 | Set-as-default GNOME terminal | **Manual, documented in runbook** | Modifying GUI desktop state is outside the declarative dotfiles contract. Reversible only by the user knowing the gsettings key. Belongs in runbook. |
| 4 (implicit) | Install gating | **Detect-and-act with warn-not-fail** | Matches BUG-001 pattern (try-everything-available, log warning on missing pieces, no flags for non-heavy installs). `apt install ghostty` is ~30MB with GTK4 deps already present on GNOME desktops -- not heavy enough for an opt-in flag like `-WithOllama`. |

## References

- Vault backlog: `10_projects/dotfiles/11-tasks.md` -> `TERM-001-ghostty-bootstrap`
- Vault runbook (existing): `10_projects/dotfiles/40-runbooks/guide-opencode-go-setup.md` -- already documents Ghostty + opencode workflow alongside tmux; the new ghostty-setup runbook will link back here for the recommended workflow section.
- Related ADR: `30-architecture/adr-009-multi-agent-runtime` -- multi-agent runtime + Ghostty as host for opencode TUI.
- Related pattern: `00_meta/patterns/pattern-setup-script-idempotence` -- the reconcile-not-skip and two-tier deploy patterns this PR will follow verbatim.
- Predecessor merged PRs in this loop: #38 (tmux truecolor for ghostty), #39 (AI-011 archive), #40 (BUG-001 Copilot detect-and-act), #41 (oclog alias).
- Umbrella: `AI-015-harness-parity-across-agents` (not blocking, but TERM-001 closes the Ghostty surface for the AI-015 audit).
