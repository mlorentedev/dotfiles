---
id: "HERDR-001-yazi-explorer"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-10-18"
issue: "mlorentedev/dotfiles#2814"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "12 active specs pre-existed (limit 10); #2814 is the new explorer work and nothing else was archivable without the owner (12 active, limit 10, 2026-10-18)"
---

# HERDR-001-yazi-explorer

> **Naming**: file lives at `<repo>/specs/HERDR-001-yazi-explorer/proposal.md`. `HERDR-001-yazi-explorer` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

herdr's sidebar shows agents and spaces only, so answering "what is in this
worktree" means leaving the cockpit or typing a listing. The owner asked for the
project's files visible on the right and accepted one new dependency to get it.
The same change also lands the agent-detection fix (#2813) that makes pi and
opencode appear at all, because the explorer is useless next to a pane the
sidebar still calls `unknown`.

## What

- `prefix+e` opens yazi as a session-modal popup over the focused pane, in that
  pane's directory.
- `dotf herdr explorer` toggles a real split on the right running yazi in the
  pane's directory: the second call closes the explorer instead of stacking a
  second one.
- `YAZI_VERSION` in `versions.conf` provisions yazi through mise on every OS; no
  manual install step exists.
- The docked binding (`prefix+Shift+e` -> `dotf herdr explorer`) lands in a
  follow-up PR, once a released `dotf` carries the subcommand: `DOTF_VERSION`
  today is `0.75.0`, and a binding that names a tree-only subcommand deploys
  cleanly in CI and fails on every new machine (lesson 373).
- `ui.agent_panel_sort = "priority"`, so a blocked or done agent floats to the
  top of the sidebar.

## Out of scope

- The docked keybinding itself, for the reason above (tracked on #2814).
- Windows verification. herdr on Windows is preview and the PowerShell profile is
  untouched on purpose: it launches the real binary in-process, so `pi` is
  already the foreground child and needs no `HERDR_AGENT` hint.
- Editor integration beyond yazi's own opener, and any plugin from the herdr
  marketplace: the vendor's install path has no pin a repo file can express, so
  it conflicts with reproducible provisioning.

## Risks / open questions

- yazi previews images only where the outer terminal speaks the Kitty graphics
  protocol (herdr's `terminal.kitty_graphics` defaults to true); `ffmpeg`, `7z`
  and `poppler` are not provisioned, so video and PDF previews are unavailable
  rather than broken. Accepted: text, code and images are the daily case.
- The docked explorer takes columns from the agent's terminal while it is open.
  The toggle bounds the cost; it does not remove it.
- yazi renders whatever is on disk, secrets included. Same exposure as `cat`, but
  one keypress away; `experimental.pane_history` stays `false`, so it is not
  persisted.

## Acceptance criteria

- [x] `herdr config check` accepts `ai/herdr/config.toml` and the deployed copy.
- [x] `dotf herdr explorer` splits the active pane to the right and runs yazi in that pane's directory.
- [x] A second `dotf herdr explorer` in the same tab closes that pane and opens nothing.
- [x] A machine without yazi gets a refusal naming `dotf tools sync`, not an empty pane.
- [x] `YAZI_VERSION` reaches the rendered mise config from the checkout, and a second `dotf tools sync` reports nothing to do.
- [x] `HERDR_AGENT` is declared per wrapper with its own value on both shells, and herdr reports the pane as that agent.

## References

- Bitácora board: mlorentedev/dotfiles#2814 (explorer), #2813 (agent detection)
- Related ADR: `docs/adr/adr-020-tooling-cli-go-convergence.md` (new tooling is Go)
- Related patterns: lesson 373 (a record the released binary reads waits for the
  release), lesson 493 (a wrapper hides the foreground process), lesson 494
  (`tools sync` reads the deploy dir's `versions.conf` outside the checkout)
