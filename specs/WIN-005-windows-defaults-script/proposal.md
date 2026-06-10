---
id: "WIN-005-windows-defaults-script"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
---

# WIN-005-windows-defaults-script

> **Naming**: file lives at `<repo>/specs/WIN-005-windows-defaults-script/proposal.md`. `WIN-005-windows-defaults-script` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: New `scripts/windows-defaults.ps1` analog to mathiasbynens' `.macos` — HKCU registry tweaks for sensible engineering defaults (show extensions, show hidden files, disable advertising-ID, disable Cortana/Bing-in-search, taskbar alignment, dark mode). No admin required. Effort: M, ~200-300 LOC. Anti-scope: NO personal preferences; NO `HKLM:` writes. -->

mathiasbynens' `.macos` ships sensible engineering defaults for Mac in one runnable script — show file extensions, disable annoyance prompts, etc. The repo's prior survey (`research/dotfiles-survey.md`) classified `.macos` as "macOS-only, doesn't port" — but that misses the philosophical analog: the same idea on Windows is HKCU registry tweaks. Windows is first-class here; new contributors lose minutes per machine setting Explorer to show file extensions and hidden files through the GUI. A scripted equivalent removes that friction.

## What

New `scripts/windows-defaults.ps1` (HKCU only, no admin) that idempotently sets ~15–20 universally-engineered defaults via `Set-ItemProperty`. Categories:

- **Explorer**: show file extensions, show hidden files, "This PC" as default open, full path in title bar.
- **Taskbar**: left-align (Win11), smaller icons, disable widgets, disable news/interests.
- **Privacy**: disable advertising ID, disable telemetry consent prompts (the screen, not telemetry itself).
- **Search**: disable Bing in Start, disable Cortana web search.
- **Theme**: dark mode.

Invoked at the end of `setup-windows.ps1` behind a new `-WithDefaults` flag (off by default; opt-in).

## Out of scope

- **HKLM writes** — would require admin; opt-in admin-required script is a separate WIN-XXX.
- **Personal preferences** (wallpaper, screensaver, sounds, mouse speed, key repeat).
- **Windows Defender / Firewall changes** — security implications, separate ticket.
- **Restarting `explorer.exe`** to apply settings — document for user, don't auto-restart.

## Risks / open questions

- **R1**: Some HKCU keys require Explorer restart to take effect. Document in script output; don't auto-restart.
- **R2**: Registry paths may shift across Windows updates. Keep all paths as named constants at top of file for easy patching.
- **R3**: `Cortana`/`Search` keys differ between Win10 and Win11. Branch on OS build.
- **R4**: Idempotency — `Set-ItemProperty` is idempotent by nature, but reading current value before writing produces cleaner logs (`[INFO] X already set` vs blind overwrite).
- **R5**: Default OFF for `-WithDefaults`. A user-opinionated script that mass-sets defaults without explicit opt-in violates user autonomy. Off by default; documented in README.

## Acceptance criteria

- [ ] `scripts/windows-defaults.ps1` exists, idempotent, no admin required.
- [ ] `setup-windows.ps1` accepts `-WithDefaults` switch and invokes the script when present.
- [ ] Flag is OFF by default — users opt in explicitly.
- [ ] All registry writes target `HKCU:\` only (verified via grep / structural test).
- [ ] `tests/windows-defaults.bats` (or `.ps1` equivalent) verifies HKCU-only invariant + idempotency.
- [ ] README documents the opt-in flag.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` § "Session 2026-05-27 — fresh-eyes audit" → WIN-005.
- Inspiration: mathiasbynens/dotfiles `.macos` (~1200 LOC of `defaults write`).
- Prior research note: `research/dotfiles-survey.md` § "Validación: cosas que el usuario YA hace mejor" — this proposal contests the survey's "doesn't port" framing.
