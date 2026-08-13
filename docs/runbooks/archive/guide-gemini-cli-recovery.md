---
id: "guide-gemini-cli-recovery"
type: runbook
status: active
tags: [runbook, dotfiles, gemini-cli, recovery, npm]
created: "2026-05-27"
owner: manu
---

# Gemini CLI Recovery

> ⚠️ **Archived.** Gemini CLI's sunset for Google One / unpaid tiers (originally
> "a few more weeks" from 2026-05-27) has passed. Users migrate to Antigravity CLI —
> see [`guide-antigravity-cli-migration.md`](../guide-antigravity-cli-migration.md).
> Kept for reference in case a `gemini-cli` install still needs this recovery procedure.

Recovery procedure for `@google/gemini-cli` when auto-update corrupts the install.

## Problem

`gemini-cli` self-updates on first launch from v0.28.2 to v0.42.0 and corrupts its own install path:

```
Cannot find module 'C:\Users\Manu\AppData\Roaming\npm\node_modules\@google\gemini-cli\dist\index.js'
```

Post-failure:
- `gemini` binary no longer on PATH (`CommandNotFoundException` on Windows, `command not found` on Linux)
- `npm list -g @google/gemini-cli` may show the package but it's broken

## Recovery Steps

### Linux / macOS

```bash
# 1. Remove the broken installation
npm uninstall -g @google/gemini-cli

# 2. Reinstall fresh
npm install -g @google/gemini-cli

# 3. Verify
gemini --version
```

### Windows (PowerShell)

```powershell
# 1. Remove the broken installation
npm uninstall -g @google/gemini-cli

# 2. Reinstall fresh
npm install -g @google/gemini-cli

# 3. Verify
gemini --version
```

## Root Cause

`@google/gemini-cli` performs a self-update on first launch after a fresh install. The update process replaces files in place while the package manager may still hold references to the old file paths. This leaves the binary in a broken state where `node_modules` references point to non-existent files.

## Mitigation

### Option A: Pin version (recommended for now)

Pin `@google/gemini-cli` to a known-working version in `versions.conf`:

```
GEMINI_CLI_VERSION="0.28.2"
```

Then in setup scripts, install the pinned version:

```bash
npm install -g "@google/gemini-cli@${GEMINI_CLI_VERSION}"
```

### Option B: Healthcheck hint

Add a healthcheck section that detects missing `gemini` binary and suggests the recovery command. See `setup-linux.sh` line ~560 (Obsidian CLI install pattern) for the `command -v` + `npm install -g` pattern.

### Option C: Upstream issue

File an issue against `@google/gemini-cli` for the broken self-update. This is the ideal long-term fix but requires investigation into whether the self-update is intentional (opt-in) or a bug.

## Important Notes

- **Do NOT auto-pin in setup scripts** until upstream behavior is investigated. The self-update may be an intentional security feature (opt-in update), and pinning blocks security patches.
- **gemini-cli sunset**: Google announced that Gemini CLI will stop serving requests for Google One and unpaid tiers starting June 18th. Users should migrate to Antigravity CLI (AI-020). This makes the recovery runbook time-sensitive — it may only be needed for a few more weeks.
- **Anti-scope**: This runbook does NOT modify setup scripts. Any code fix opens a separate atomic PR.

## Related

- [AI Tools](../../troubleshooting/ai-tools.md) — AI tool installation overview
- Task backlog (BUG-019, AI-020) — maintainer's knowledge store
