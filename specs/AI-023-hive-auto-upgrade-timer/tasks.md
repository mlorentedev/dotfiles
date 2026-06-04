---
tags: [spec, tasks]
created: "2026-06-03"
---

# Tasks - AI-023-hive-auto-upgrade-timer

> One task = one focused commit. Tick as you go.

## Setup

- [x] Branch (worktree) created from main: `feat/hive-auto-update-timer`
- [x] `proposal.md` complete; acceptance criteria testable
- [x] Timer scheduling semantics decided with the maintainer: wall-clock
  (`OnCalendar=*:0/15`) + `Persistent=true` catch-up

## Implementation

- [x] `systemd/hive-upgrade.service`: `Type=oneshot`, absolute `%h/.local/bin/uv tool upgrade hive-vault`
- [x] `systemd/hive-upgrade.timer`: `OnCalendar=*:0/15`, `Persistent=true`, `RandomizedDelaySec=60`, `WantedBy=timers.target`
- [x] `setup-linux.sh`: inside the >=1.32.0 gate, deploy units to `~/.config/systemd/user/`, `daemon-reload`, `enable --now hive-upgrade.timer` (non-fatal), strip legacy `uv tool upgrade hive-vault` cron on success
- [x] `setup-windows.ps1`: inside the >=1.32.0 gate, register/self-heal daily `DotfilesHiveUpgrade` Scheduled Task running `uv tool upgrade hive-vault`
- [x] `bash -n` + shellcheck clean on the added block; `setup-windows.ps1` ASCII-only

## Closing

- [x] Each acceptance criterion covered by a `features.json` entry
- [x] `tests/hive-upgrade-timer.bats` added (units + both setup scripts + parity)
- [x] No unrelated changes
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder
- [ ] (post-merge) Multi-machine rollout by the maintainer, then tick hive #176

## Machine-readable features

See sibling `features.json`. Static/structural checks (unit file shape, setup
wiring, version gate, cron strip, ASCII). The dynamic timer firing + real upgrade
is validated by the maintainer's rollout (the systemd journal + `uv tool list`).
The agent must NOT set `state: passing` — only the harness may, after a clean run.
