---
id: "AI-023-hive-auto-upgrade-timer"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-03"
tags: [spec, proposal]
template_version: "1.0"
---

# AI-023-hive-auto-upgrade-timer

> The policy half of the Phase C daemon model. Sibling of
> `specs/AI-022-hive-daemon-activation/` (which activated the daemon + migrated
> the MCP entry). Tracking SSOT: `mlorentedev/hive` issue #176. This is the last
> dotfiles piece before the multi-machine rollout.

## Why

hive shipped the *mechanism* for zero-downtime upgrades in v1.32.2:
`hive serve` polls its installed version and exits 75 under
`Restart=on-failure`, so the supervisor relaunches it into new code within
`HIVE_UPGRADE_POLL_S`. AI-022 then flipped the fleet onto that daemon
(`hive client` proxy + `hive service install`). But nothing *feeds* the
mechanism: a machine only gets new hive code when someone manually runs
`uv tool upgrade hive-vault`. Today that is a hand-rolled `0 9 * * *` crontab
line on one machine — invisible, per-machine, and a second owner of a policy the
daemon model should own centrally.

This change adds the periodic upgrade so every machine always runs the latest
hive-vault with zero added client-launch latency, and makes that schedule the
**single owner** of the upgrade policy.

## What

After `setup-linux.sh` / `setup-windows.ps1` runs on a machine with
hive-vault >= 1.32.0:

- **Linux:** a systemd `--user` `hive-upgrade.timer` (every 15 min, wall-clock
  slots with catch-up after suspend/boot) triggers a `hive-upgrade.service`
  oneshot that runs `uv tool upgrade hive-vault`. Units ship as static files in
  `systemd/` (SSOT), deployed to `~/.config/systemd/user/` and enabled inside the
  same version gate as `hive service install`.
- **Windows:** a daily `DotfilesHiveUpgrade` Scheduled Task running
  `uv tool upgrade hive-vault`, self-healing on action drift, parity with the
  existing `DotfilesVaultMaintenance` task.
- **SSOT cleanup:** on Linux, once the timer is enabled, any legacy
  `uv tool upgrade hive-vault` crontab line is removed so the timer is the single
  owner of the upgrade policy.

## Out of scope

- macOS / launchd timer (hive's daemon model is non-zero stub there; follow-up).
- Any change to hive itself (the daemon, CLI, restart-on-upgrade already shipped
  on PyPI >= 1.32.2).
- Changing the upgrade cadence per-machine / making it configurable (single fixed
  policy for now; revisit only if a machine needs a different window).

## Risks / open questions

- **`uv` not on the systemd service PATH** — a `--user` oneshot gets a minimal
  environment. Resolved: `ExecStart` uses the absolute `%h/.local/bin/uv`, exactly
  where `setup-linux.sh` installs uv.
- **No systemd `--user` / Task Scheduler** (headless/CI, or logged-out without
  linger) — the timer/task cannot be enabled. Resolved: non-fatal (warning); the
  setup `prerequisite_command` (`uv tool install --upgrade hive-vault`) still
  upgrades hive on every run, so the machine never goes stale silently.
- **Two owners of the upgrade policy** — the manual 9am cron and the timer would
  both run `uv tool upgrade`. Resolved: the timer-install path strips the legacy
  crontab line (only after a successful enable, so an old-hive machine that skips
  the gate keeps its cron).
- **Old hive (< 1.32.0)** — gate skips both service and timer. In practice the
  `prerequisite_command` already bumped hive to latest earlier in the same setup
  run, so the gate passes; the skip is only a safety net for the offline /
  failed-upgrade case.

## Acceptance criteria

- [ ] `systemd/hive-upgrade.timer` fires every 15 min wall-clock (`OnCalendar=*:0/15`)
  with `Persistent=true` (catch-up) and installs to `timers.target`.
- [ ] `systemd/hive-upgrade.service` is a `Type=oneshot` running
  `uv tool upgrade hive-vault` via the absolute `%h/.local/bin/uv`.
- [ ] `setup-linux.sh` deploys the units to `~/.config/systemd/user/` and enables
  the timer inside the existing hive-vault >= 1.32.0 version gate (never on an old
  hive); non-fatal if systemd `--user` is unavailable.
- [ ] `setup-linux.sh` removes a legacy `uv tool upgrade hive-vault` crontab line
  once the timer is enabled (single owner).
- [ ] `setup-windows.ps1` registers a daily `DotfilesHiveUpgrade` Scheduled Task
  running `uv tool upgrade hive-vault`, gated the same way; self-heals on drift.
- [ ] `setup-linux.sh` passes `bash -n` + shellcheck; `setup-windows.ps1` stays
  ASCII-only (PSScriptAnalyzer); `make test` / bats stay green.

## References

- hive issue: [mlorentedev/hive#176](https://github.com/mlorentedev/hive/issues/176)
- sibling spec: `specs/AI-022-hive-daemon-activation/` (daemon activation + MCP migration)
- restart-on-upgrade mechanism: hive v1.32.2 (`hive serve` exits 75 under `Restart=on-failure`)
- systemd timer reference: `man systemd.timer` (`OnCalendar`, `Persistent`, `RandomizedDelaySec`)
