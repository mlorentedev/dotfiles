---
id: lesson-234-orchestrating-orca-ade-declarative-configuration-and-bi
type: lesson
status: active
created: "2026-08-27"
owner: manu
tags: [lesson, dotfiles, orca, cli, deploy]
---

# Lesson 234: Orchestrating Orca ADE declarative configuration and bidirectional settings capture

**Context**: CLI-051/#1273 — Orca ADE (Stably AI) settings were split between shell tuning scripts (`scripts/orca-tune.sh`), untracked user directories (`~/.orca/keybindings.json`), and the primary data store (`~/.config/orca/orca-data.json`).

**Problem**: `orca-data.json` combines declarative user settings (`settings` object) with dynamic session state (active worktrees, PTY leases, GitHub caches, and temporary cookies). Symlinking the entire directory or committing `orca-data.json` into git risks polluting repository history with private ephemeral data. Furthermore, applying settings to `orca-data.json` while Orca is actively running causes silent clobbering when the Electron app flushes its in-memory state on exit.

**Solution**:
1. Separate keybindings into `ai/orca/keybindings.json` and deploy them idempotently using the declarative `dotf deploy` engine (`ai/deploy.json`).
2. Implement `dotf orca export` in Go to extract the isolated `settings` JSON map from `orca-data.json` and dump it into `ai/orca/settings.json` while skipping all ephemeral top-level runtime keys.
3. Port baseline tuning into `dotf orca tune` with process guards (`pgrep` matching `orca-ide` and `AppImage`) and timestamped pre-write backups.

**Rule**: When managing desktop ADEs or Electron-based tools whose state files mix persistent user preferences with ephemeral runtime cache, never symlink or commit the raw state file. Provide an export command that strips runtime noise and a guarded tuning command that prevents write collisions against active processes.
