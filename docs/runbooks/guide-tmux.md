---
id: guide-tmux
type: runbook
status: active
created: "2026-05-11"
---

# tmux Quick Reference

> Linux-only. Config lives in the dotfiles repo at `tmux.conf`, deployed to `~/.tmux.conf` by `setup-linux.sh`. Prefix: default `C-b` (preserves muscle memory across machines).

## Install

tmux is a system package — `setup-linux.sh` deliberately does not `sudo`, so install it once per machine:

```sh
sudo apt install -y tmux
tmux -V   # expect: tmux 3.4 or newer
```

After install, run `./setup-linux.sh` from the dotfiles repo to deploy `~/.tmux.conf` (copy of the repo's `tmux.conf`), then verify with `dotf doctor` (tmux section).

## How the config gets deployed

Two-tier copy pattern (same as `.zshrc`, `.bashrc`, `.gitconfig`):

```
~/Projects/dotfiles/tmux.conf    (canonical, git tracked — edit here)
        │ safe_copy in setup-linux.sh
        ▼
~/.dotfiles/tmux.conf            (deploy dir — do not edit)
        │ ln -sf in setup-linux.sh
        ▼
~/.tmux.conf                     (symlink resolves to deploy dir)
```

To update: edit the repo file, commit, re-run `setup-linux.sh`. The symlink stays the same; the deploy-dir target gets refreshed.

## Daily commands (shell aliases)

| Action | Command |
|---|---|
| Start/attach session by name | `tx <name>` |
| List sessions | `txl` |
| Attach to most recent | `txa` |
| Kill named session | `txk <name>` |
| SSH + remote tmux attach-or-create | `sshmux <host> [session]` |

## Inside tmux (prefix `C-b`)

| Action | Keys |
|---|---|
| Detach (session keeps running) | `C-b d` |
| Reload `~/.tmux.conf` | `C-b r` |
| Vertical split | `C-b %` |
| Horizontal split | `C-b "` |
| Move between panes (vim style) | `C-b h/j/k/l` |
| Zoom current pane | `C-b z` |
| Close pane | `C-b x` |

## Remote SSH persistence

`sshmux <host>` is the killer feature for this workflow. With ~16 SSH targets in `ssh/config`, dropped connections used to mean lost work. Now:

```sh
sshmux rpi4              # ssh in, attach-or-create "main" tmux session
# wifi drops — remote session keeps running
sshmux rpi4              # reattach, output preserved
```

Requires tmux installed on the remote host. For kubelab-managed hosts (ace1, ace2, rpi4, rpi3, beelink, vps, aws1), this is provisioned by the `base_system` Ansible role — see ticket ANSIBLE-021 in the kubelab project (maintainer's knowledge store). For ad-hoc hosts: `ssh <host> 'sudo apt install -y tmux'`. Default session name is `main`; override with `sshmux <host> <name>`.

## Project workflow

- One named session per project: `tx dotfiles`, `tx kubelab`, `tx hydra3d`
- `C-b d` to detach when switching projects, `tx <name>` to come back
- Pane layout for AI pair-programming: editor left (`%`), aider right, `bats -w` bottom (`"`)
- Sessions survive terminal-window close — re-attach with `txa` or `tx <name>`

## Verification

| Check | How |
|---|---|
| Binary installed | `dotf doctor` (tmux section, or `tmux -V`) |
| Config deployed | `dotf doctor` (verifies `~/.tmux.conf` matches the repo source) |
| Config parses | `tmux -f tmux.conf -L test new-session -d -s s 'sleep 0.5' && tmux -L test kill-server` |
| Integration test | `tests/Dockerfile.integration` builds and verifies in clean Ubuntu 24.04 |

## Copy / paste

tmux ships with `set -g mouse on` and `mode-keys vi`. The config also pipes any selection to the system clipboard via `xclip`, so what you select inside tmux is immediately pasteable in any other app.

### Required dependency

`xclip` is the bridge between tmux's internal buffer and the X11 system clipboard. Install once per machine — `setup-linux.sh` warns if missing but does not `sudo`:

```sh
sudo apt install -y xclip
xclip -version   # sanity
```

After install, reload the config inside a running tmux: `C-b r`.

### Mouse workflow

| Action | What happens |
|---|---|
| Drag to select → release | Selection copied to **both** tmux buffer and system clipboard |
| Double-click | Selects word, copies it |
| Triple-click | Selects line, copies it |
| Scroll wheel up in a pane | Enters copy mode automatically |
| `Shift` + drag | Bypasses tmux, uses terminal's native selection (useful for selections that cross splits) |

### Keyboard workflow (vi-mode)

1. `C-b [` — enter copy mode (indicator appears top-right).
2. Navigate: `h j k l`, `w b`, `0 $`, `gg G`, `/text` (search forward), `?text` (backward), `n` / `N`.
3. `Space` — start selection.
4. Move to extend.
5. `y` — copy and exit copy mode. Lands in clipboard **and** tmux buffer.
6. `q` or `Esc` — exit without copying.

### Pasting

| Where | How |
|---|---|
| Outside tmux (browser, GUI app) | `Ctrl+V` (system clipboard) |
| Inside a tmux pane | `Ctrl+Shift+V` (terminal paste) or `C-b ]` (tmux buffer paste) |
| Pick from previous copies | `C-b =` opens buffer list, `Enter` to paste selected |

### Wayland (when migrating from X11)

`xclip` only talks to X11. On a Wayland session (`echo $XDG_SESSION_TYPE` shows `wayland`):

1. Install Wayland clipboard tools: `sudo apt install -y wl-clipboard`.
2. Edit `tmux.conf` in the dotfiles repo: replace every `xclip -selection clipboard -in` with `wl-copy`.
3. Re-run `./setup-linux.sh` to redeploy, then `C-b r` inside tmux.

The config keeps a comment marker on the binding line. If both servers matter (e.g. XWayland mixed setups), the alternative is a dispatcher script that prefers `wl-copy` and falls back to `xclip`.

### Troubleshooting

| Symptom | Cause / fix |
|---|---|
| Selection copies to tmux buffer but `Ctrl+V` outside tmux pastes old content | `xclip` not installed, or config not reloaded — `sudo apt install -y xclip && tmux source-file ~/.tmux.conf` |
| Mouse drag does nothing visible | Mouse mode off — confirm `set -g mouse on` is in the deployed config (`grep mouse ~/.tmux.conf`) |
| Selection vanishes immediately on release | Working as intended — `copy-pipe-and-cancel` exits copy mode after copying. Use `Ctrl+V` to paste. |
| Want to copy across splits | Hold `Shift` while dragging to bypass tmux's pane awareness |
| `xclip` hangs / process visible in `ps` | Known xclip quirk; harmless. To silence: pipe through `xclip -selection clipboard -in -loops 1` (already the default for `copy-pipe-and-cancel`) |

## See also

- `tmux.conf` in dotfiles repo root — the versioned config
- `.zsh/aliases.zsh` — `tx`/`txl`/`txa`/`txk` aliases
- `.zsh/functions.zsh` — `sshmux` function
- `tests/tmux.bats` — config parse + content assertions
- `dotf doctor` (tmux section) — runtime verification
