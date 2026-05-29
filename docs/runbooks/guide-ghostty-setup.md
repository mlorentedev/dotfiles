---
id: guide-ghostty-setup
type: runbook
status: active
created: "2026-05-17"
---

# Ghostty Setup (Linux)

> **Goal:** install Ghostty as the daily-driver terminal emulator and host for the opencode TUI (which suffers visible render lag under tmux). The dotfiles bootstrap (`setup-linux.sh`) handles the config deploy and version pin check; this runbook covers the GUI/manual steps that cannot or should not be automated.
>
> **Audience:** future-me on a clean Ubuntu 26.04+ box. Cross-OS Windows side defers to WingGhostty maturity -- see [`guide-opencode-go-setup.md`](guide-opencode-go-setup.md) for the recommended-workflow section that pairs with this.

## Prerequisites

| Item | Why | Check |
|---|---|---|
| Ubuntu 26.04+ with `universe` enabled | Ghostty 1.3.0 lives in the universe archive | `apt-cache policy ghostty | head -3` |
| OpenGL 4.3+ GPU | Ghostty is GPU-accelerated; falls back to software rendering otherwise | `glxinfo \| grep "OpenGL version"` |
| `setup-linux.sh` already ran | Deploys `terminal/ghostty/config` -> `~/.config/ghostty/config` + sets `GHOSTTY_VERSION` pin check in healthcheck | Healthcheck section 11/12 reports ok |

If GPU OpenGL is below 4.3, install drivers (`sudo ubuntu-drivers autoinstall` for NVIDIA; AMD/Intel usually fine out of the box on Ubuntu 26.04) before chasing slower-than-expected rendering.

## Step 1 -- install the binary (one-shot, requires sudo)

The dotfiles setup script avoids sudo for tool installs (same convention as tmux/xclip). Run this once per machine:

```bash
sudo apt update
sudo apt install -y ghostty
ghostty --version
```

Expected: `Ghostty 1.3.0-dev+...` (the universe build is the development tip; the `-dev+HASH` suffix is functionally equivalent to a 1.3.0 release for our purposes -- healthcheck strips the suffix when comparing against `GHOSTTY_VERSION` in `versions.conf`).

If `apt` cannot find `ghostty`: `sudo add-apt-repository universe && sudo apt update` and retry.

## Step 2 -- install JetBrainsMono Nerd Font (one-shot, no sudo)

The deployed Ghostty config declares `font-family = JetBrainsMono Nerd Font Mono`. The font is not auto-fetched by `setup-linux.sh` -- Nerd Font tarballs are heavyweight and version-fragile, so we keep this manual:

```bash
mkdir -p ~/.local/share/fonts
cd /tmp
wget -q https://github.com/ryanoasis/nerd-fonts/releases/download/v3.2.1/JetBrainsMono.zip
unzip -q -o JetBrainsMono.zip -d ~/.local/share/fonts/JetBrainsMonoNerd
fc-cache -fv >/dev/null 2>&1
fc-list | grep -i "jetbrainsmono nerd" | wc -l
```

Expected: a number > 0 (typically 40-50 variants). If 0, the wget likely failed silently -- check network.

If you skip this step: Ghostty still launches, but glyphs for the opencode TUI status icons render as boxes `?`. Cosmetic, not functional.

## Step 3 -- first launch + sanity checks

Open Ghostty from the GNOME launcher (Super -> type "ghostty") or `ghostty &` from any terminal. Inside the new window:

```bash
echo $0                # -zsh or /usr/bin/zsh (Ghostty respects $SHELL)
echo $TERM             # xterm-ghostty (Ghostty's terminfo identifier)
echo $COLORTERM        # truecolor (24-bit advertised)
printf '\033[38;2;255;100;0mgradient test\033[0m\n'   # should be smooth orange, not flat
```

All four signals must be sane. If `$TERM` is not `xterm-ghostty`, the deployed config is overriding it -- check `~/.config/ghostty/config` for a stray `term =` line (default is fine).

## Step 4 -- (optional) set Ghostty as the GNOME default terminal

By design, this is NOT automated by `setup-linux.sh` -- modifying `gsettings` state belongs to the user's per-machine choice. One-liner if you want it:

```bash
gsettings set org.gnome.desktop.default-applications.terminal exec 'ghostty'
gsettings set org.gnome.desktop.default-applications.terminal exec-arg ''
```

Reverse:

```bash
gsettings reset org.gnome.desktop.default-applications.terminal exec
```

Note: this only affects the "default terminal" used when other apps spawn a terminal (rare). Launching Ghostty manually from the GNOME launcher works regardless.

## Step 5 -- SSH terminfo workaround for remote hosts

Ghostty advertises `TERM=xterm-ghostty`, a custom terminfo not present on most remote machines. When you `ssh somehost` from Ghostty, the remote may log `WARNING: terminal is not fully functional` or mis-render keys.

**Per-host fix (run once per remote host):**

```bash
infocmp -x xterm-ghostty | ssh somehost -- tic -x -
```

This installs the local Ghostty terminfo entry on the remote. Subsequent ssh sessions from Ghostty to that host render correctly.

**Quick fallback if you do not want to install the terminfo on every remote:** alias `ssh` to force a portable TERM:

```bash
alias ssh='TERM=xterm-256color ssh'
```

Trade-off: you lose some Ghostty-specific terminfo capabilities, but everything still works.

## Recommended workflow (alongside opencode + tmux)

This is the empirical workflow from AI-011-validation (2026-05-17):

| Task | Use |
|---|---|
| opencode TUI session (cargo-load, slow under tmux) | **Ghostty native split/tab** (`Ctrl+Shift+E` for split, `Ctrl+Shift+T` for tab) |
| Shell session, dev work, editing, anything not opencode | **tmux** (`tx <name>` to attach-or-create) |
| Persistent SSH session | **tmux on the remote** via `sshmux` |
| Live tail of opencode log while debugging "thinking..." stalls | **`oclog` alias** in a parallel Ghostty split |

PR #38 already enabled truecolor passthrough for tmux running inside Ghostty (`xterm-ghostty:Tc` override in `tmux.conf`), so claude / shell sessions inside tmux inside Ghostty still get full 24-bit color. The reason to drop tmux specifically for opencode is the TUI's aggressive full-screen refreshes that tmux's ANSI re-emission cannot keep up with -- not a color or capability issue.

## Theme + font choices (rationale)

- `theme = Catppuccin Mocha` -- dark, pastel, contrast around 7:1. Lower-fatigue than GitHub Dark Default or Dracula. Family alternatives if Mocha is too contrasty: `Catppuccin Frappe`, `Catppuccin Macchiato`. Cross-family low-fatigue contenders: `Gruvbox Material Dark` (warmer earth tones), `Everforest Dark Hard` (green-soft), `Rose Pine Moon` (muted pastel). List all 463 themes Ghostty ships with: `ghostty +list-themes`. **Theme name format is literal capitalization with spaces, NOT kebab-case** -- `Catppuccin Mocha`, not `catppuccin-mocha`.
- `font-family = JetBrainsMono Nerd Font Mono` -- monospace ligatures-friendly + Nerd Font glyphs for opencode/claude-code status icons. Common alternatives: `FiraCode Nerd Font Mono`, `Hack Nerd Font Mono`, `IosevkaTerm Nerd Font Mono`.
- `font-size = 12` -- adjust to 13-14 if reading at >50cm from screen. Larger font reduces accommodation effort, often more impactful on fatigue than the theme.
- `confirm-close-surface = true` -- guards against killing tmux sessions or live opencode contexts with an accidental Ctrl+Shift+W.

## Bigger eye-fatigue levers than theme

Honest ranking from biggest impact to smallest:

1. **GNOME Night Light** (Settings -> Display -> Night Light). Enable, schedule auto sunset-to-sunrise OR a custom 24/7 manual range, temperature ~3700K (warmer than the 4000K default). This recorta blue light system-wide -- more impact than any terminal theme.
2. **Monitor brightness 40-60% in dim rooms.** Brightness at 100% at night fatigues 2x more than the highest-contrast theme.
3. **20-20-20 rule.** Every 20 minutes, look 20 seconds at something 20 feet (6m) away. Relaxes the ciliary muscle.
4. **Font size.** Increase if you find yourself leaning toward the screen.
5. **Theme choice.** Marginal compared to the four above. Going from `GitHub Dark High Contrast` to `Catppuccin Mocha` helps; going from Mocha to Gruvbox is a delta.

## Troubleshooting

### `command -v ghostty` fails after `setup-linux.sh`

`setup-linux.sh` does NOT auto-install ghostty (avoids sudo). Manually: `sudo apt install -y ghostty`. The script will warn rather than fail when ghostty is absent.

### `theme "<name>" not found` on `ghostty +validate-config`

Wrong theme name format. Ghostty uses capitalized-with-space (`Catppuccin Mocha`), not kebab-case. List + grep: `ghostty +list-themes | grep -i <fragment>`. Empirical finding 2026-05-17.

### Glyphs render as boxes / question marks in opencode TUI

JetBrainsMono Nerd Font not installed or not selected. Re-run Step 2. Verify with `fc-list | grep -i "jetbrainsmono nerd"`.

### `~/.tmux.conf` truecolor under Ghostty stops working

The override `xterm-ghostty:Tc` in `tmux.conf:13` (PR #38) is what makes this work. Check: `grep xterm-ghostty ~/.tmux.conf`. If absent, re-run `./setup-linux.sh` to redeploy.

### opencode TUI noticeably slow specifically under tmux

Known limitation, not a bug. See [`guide-opencode-go-setup.md`](guide-opencode-go-setup.md) -> "TUI feels noticeably slower than Claude Code, especially under tmux". Use Ghostty native splits/tabs for opencode TUI; keep tmux for shell/SSH.

### `ghostty --version` reports a `-dev+HASH` suffix and healthcheck warns about version drift

The Ubuntu universe build of ghostty 1.3.0 ships with the `-dev+0000000` tip-channel suffix. Healthcheck strips the suffix when comparing against `GHOSTTY_VERSION` in `versions.conf` -- a true drift means the `1.3.0` part differs. Bump `GHOSTTY_VERSION=1.3.0` in `versions.conf` if the upstream package moves to `1.3.1`.

### Want to reset Ghostty defaults

`gsettings reset` of the GNOME default; `rm ~/.config/ghostty/config` then re-run `./setup-linux.sh` to redeploy the canonical config from the repo.

## References

- Spec: `~/Projects/dotfiles/specs/TERM-001-ghostty-bootstrap/`
- Related runbook: [`guide-opencode-go-setup.md`](guide-opencode-go-setup.md) (the opencode + tmux + Ghostty stack)
- Related ADR: [`../adr/adr-009-multi-agent-runtime.md`](../adr/adr-009-multi-agent-runtime.md)
- Related pattern: `pattern-setup-script-idempotence` (maintainer's cross-project knowledge store)
- Upstream: <https://ghostty.org/docs/>
- WingGhostty (community native Windows port, evaluated for future TERM-002): <https://winghostty.com/>
