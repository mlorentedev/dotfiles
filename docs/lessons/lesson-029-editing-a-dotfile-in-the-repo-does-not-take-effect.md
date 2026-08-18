---
id: lesson-029-editing-a-dotfile-in-the-repo-does-not-take-effect
type: lesson
status: active
created: "2026-05-11"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 029: Editing a dotfile in the repo does not take effect until `setup-linux.sh` runs

**Context**: After editing `tmux.conf` in `~/Projects/dotfiles/` to add clipboard bindings (`copy-pipe-and-cancel` piped to `xclip`), tmux still behaved like the old config. Mouse selection produced nothing in the system clipboard.

**Problem**: The dotfiles repo uses a **two-tier deploy**, not a direct symlink to the repo:

```
~/Projects/dotfiles/<file>   ← canonical, git-tracked (you edit here)
        │ safe_copy in setup-linux.sh
        ▼
~/.dotfiles/<file>           ← deploy-dir intermediate (what the symlink resolves to)
        ▲ ln -sf in setup-linux.sh
~/.<file>                    ← active symlink in $HOME
```

The symlink `~/.tmux.conf → ~/.dotfiles/tmux.conf` was intact, but `~/.dotfiles/tmux.conf` was stale — `grep -c "copy-pipe-and-cancel" ~/.tmux.conf` returned `0` despite the repo file having the bindings. tmux was reading the old version of the file.

This applies to **every** dotfile deployed via `safe_copy` in `setup-linux.sh`: `tmux.conf`, `.gitconfig`, `.zshrc`, `.bashrc`, etc.

**Solution**:

```sh
cd ~/Projects/dotfiles
./setup-linux.sh                  # refreshes ~/.dotfiles/<file> from the repo
tmux source-file ~/.tmux.conf     # for tmux: reload running session
```

For other dotfiles, the appropriate reload command (e.g. `exec zsh`, `source ~/.zshrc`) after the redeploy.

**Rule**: After editing **any** file in `~/Projects/dotfiles/` that's tracked by `setup-linux.sh`, the change is not live until both:
1. `./setup-linux.sh` runs (refreshes the deploy-dir middle layer).
2. The relevant tool reloads its config.

Verification check before claiming the change is live:

```sh
grep -c "<new content>" ~/.<file>   # should match what's in the repo file
```

Cross-ref: [`runbooks/guide-tmux.md`](runbooks/guide-tmux.md) documents the same flow under "How the config gets deployed".
