---
id: lesson-028-tmux-clipboard-needs-an-external-bridge-and-that-b
type: lesson
status: active
created: "2026-05-11"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 028: tmux clipboard needs an external bridge — and that bridge is display-server-specific

**Context**: After enabling `set -g mouse on` and `mode-keys vi`, selections inside tmux still did not appear in the system clipboard. `Ctrl+V` outside tmux pasted stale content.

**Problem**: tmux's `copy-pipe` writes to its own internal buffer, not to the OS clipboard. The buffer is only exposed externally if you pipe the selection through an out-of-process tool. That tool is also display-server-specific:

- X11 → `xclip` (or `xsel`)
- Wayland → `wl-copy` (from `wl-clipboard`)
- macOS → `pbcopy` (stdlib, but irrelevant for this Linux-only setup)

`xclip` does not work on a pure Wayland session. `wl-copy` does not work on X11.

**Solution**: Pipe selections to `xclip` via `copy-pipe-and-cancel` and install `xclip` as a system package (added a warning block to `setup-linux.sh`, matching the existing `tmux` pattern — no sudo from the script). Bindings live in `tmux.conf`:

```tmux
bind -T copy-mode-vi y                 send-keys -X copy-pipe-and-cancel 'xclip -selection clipboard -in'
bind -T copy-mode-vi MouseDragEnd1Pane send-keys -X copy-pipe-and-cancel 'xclip -selection clipboard -in'
```

**Rule**: When migrating display server, the clipboard bridge must change in lockstep:
- X11 → keep `xclip`.
- Wayland → swap to `wl-copy` in `tmux.conf` **and** install `wl-clipboard` instead of `xclip`.

Check current server with `echo $XDG_SESSION_TYPE`. Full operational walkthrough lives in [`runbooks/guide-tmux.md`](../runbooks/guide-tmux.md) under the "Copy / paste" section.
