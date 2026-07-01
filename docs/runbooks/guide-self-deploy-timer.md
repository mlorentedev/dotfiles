---
id: "guide-self-deploy-timer"
type: runbook
status: active
tags: [runbook, self-deploy, systemd, scheduled-task, ops, opt-in]
created: "2026-06-09"
---

# Runbook: dotfiles self-deploy timer (OPS-001)

> Opt-in automation that keeps a machine converged to `origin/main` by pulling the dotfiles repo
> (fast-forward only) and re-running the idempotent setup. Design: [ADR-019](../adr/adr-019-self-deploy-fast-forward-only.md).
> Issue: [#295](https://github.com/mlorentedev/dotfiles/issues/295).

## What it does

Once a day the timer runs `dotf update` (the Go port of the former
`dotfiles-selfupdate` twins, CLI-027), which:

1. Skips if the repo worktree is **dirty** (uncommitted changes).
2. `git fetch`es; skips on a network failure.
3. Fast-forwards `main` **only** if it can; skips on diverged/non-ff history.
4. Re-runs `setup` **only if `HEAD` actually moved**.

Every skip exits 0 (benign). Only a setup command that runs and fails exits non-zero.

## Enable / disable

The mechanism is opt-in and default OFF — a normal `setup` run never installs it.

### Linux (systemd `--user`)

```bash
# Enable: deploy the units + enable the daily timer
DOTFILES_AUTODEPLOY=1 ./setup-linux.sh

# Disable: stop, disable, and remove the units
DOTFILES_AUTODEPLOY=0 ./setup-linux.sh
```

### Windows (Scheduled Task)

```powershell
# Enable
$env:DOTFILES_AUTODEPLOY = "1"; .\setup-windows.ps1
# Disable
$env:DOTFILES_AUTODEPLOY = "0"; .\setup-windows.ps1
```

## Verify it is active

```bash
# Linux: timer listed and next run scheduled
systemctl --user list-timers dotfiles-selfupdate.timer
systemctl --user status dotfiles-selfupdate.service   # last run result

# Run it once by hand (safe; no-ops on a clean, current repo)
dotf update
```

```powershell
# Windows
Get-ScheduledTask -TaskName DotfilesSelfUpdate
Get-ScheduledTaskInfo -TaskName DotfilesSelfUpdate   # LastRunTime / LastTaskResult
```

## Diagnose

| Symptom | Likely cause | Action |
|---|---|---|
| `[skip] Dirty worktree` in the journal | uncommitted changes in the repo | commit or stash; the timer resumes next slot |
| `[skip] ... diverged ... (non fast-forward)` | local commits not on `origin/main` | push or reconcile your branch; non-ff is never auto-resolved |
| `[skip] git fetch failed (network?)` | offline / remote unreachable | transient; next slot retries |
| `[skip] No upstream configured` | current branch has no `@{u}` | `git branch --set-upstream-to=origin/main` |
| service shows **failed** | `setup` itself errored (not a skip) | read `journalctl --user -u dotfiles-selfupdate.service` |

```bash
# Linux logs
journalctl --user -u dotfiles-selfupdate.service --since today
```

## Override the target repo

`dotf update` resolves the repo via the ADR-025 seam, defaulting to `$HOME/Projects/dotfiles`;
override with `DOTFILES_REPO_DIR`. The setup command it re-runs can be overridden with
`DOTFILES_SELFUPDATE_SETUP_CMD`.
