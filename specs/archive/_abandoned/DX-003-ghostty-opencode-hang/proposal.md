---
id: "DX-003-ghostty-opencode-hang"
type: spec
status: abandoned # draft | implementing | verifying | archived
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
---

# DX-003-ghostty-opencode-hang

> **Naming**: file lives at `<repo>/specs/DX-003-ghostty-opencode-hang/proposal.md`. `DX-003-ghostty-opencode-hang` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: (formalises Phase 2.4): Ghostty/Linux opencode tool-resolution hang investigation. Needs Linux machine. Test plain `opencode` in (a) Ghostty, (b) gnome-terminal/xterm, (c) ssh session. If hang only in Ghostty → file upstream issue, drop `--pure` from `.bashrc` + `.zsh/aliases.zsh`, restore strict cross-OS parity. -->

`opencode` invocation hangs on Linux under Ghostty (the user's preferred terminal emulator) during tool resolution. The repo currently works around it by passing `--pure` flag (from `.bashrc` + `.zsh/aliases.zsh`) — but that's a workaround, not a fix, and it diverges from Windows where no `--pure` is needed. Strict cross-OS parity (Standing Order #1: SSOT) requires this be either fixed at the boundary (Ghostty / opencode upstream) or proven to be intentional.

## What

A two-step investigation, no code changes in the dotfiles itself:

1. **Reproduction matrix on a Linux machine**: launch plain `opencode` in (a) Ghostty, (b) gnome-terminal / xterm, (c) ssh session, (d) tmux. Log: does it hang? Where? Stack trace via `strace -p <pid>` if hung.
2. **Decision branch**:
   - If hang reproduces only in Ghostty → file upstream issue (Ghostty or opencode side, whichever the trace fingerprints). Remove `--pure` from the repo aliases and replace with a `terminal != ghostty` guard or a one-line README warning.
   - If hang reproduces in multiple terminals → upstream opencode issue with stronger repro; keep `--pure` until fixed.
   - If hang doesn't reproduce → the workaround can be removed entirely; vault entry ticks done.

Deliverable is the investigation log + decision + (if upstream filed) the issue URL captured in a vault troubleshooting note.

## Out of scope

- **Fixing Ghostty or opencode upstream** — this spec investigates and files; upstream owns the fix.
- **Replacing Ghostty** — the user picked Ghostty deliberately (TERM-001). Don't downgrade because of a workaround.
- **Auto-detection of Ghostty in setup-linux.sh** — pure investigation, no auto-fix logic.

## Risks / open questions

- **R1**: Needs a Linux machine. The author's daily-driver is Windows; the spec sits pending until a Linux session lands.
- **R2**: `--pure` flag's semantic — what env vars does it actually purge? Read opencode's source or docs before drawing conclusions about hang cause.
- **R3**: Ghostty version pinning. The hang may be specific to a Ghostty release; capture exact version (`ghostty --version`) at repro time.

## Acceptance criteria

- [ ] Reproduction matrix run on Linux (Ghostty + gnome-terminal/xterm + ssh + tmux).
- [ ] Strace / dmesg / stack-trace evidence captured for any hang.
- [ ] Decision documented: upstream-bug / workaround-stays / can-remove.
- [ ] If upstream-bug: GH issue filed on Ghostty or opencode (URL captured).
- [ ] If can-remove: `--pure` removed from `.bashrc` + `.zsh/aliases.zsh`, cross-OS parity restored, drift detector confirms.
- [ ] Vault troubleshooting note `50-troubleshooting/ghostty-opencode-hang.md` exists with the findings.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` → DX-003 (formalises Phase 2.4).
- Workaround sites: `.bashrc`, `.zsh/aliases.zsh` (look for `opencode --pure`).
- Companion: TERM-001 (Ghostty bootstrap, already shipped).
