---
id: "lesson-527-an-optional-credential-must-not-gate-a-daemon-s-core-service"
type: lesson
status: active
title: "An optional credential must not gate a daemon's core service"
created: "2026-10-31"
---

# An optional credential must not gate a daemon's core service

## Context

CLI-042 AC7 kept the NaN API key off disk by starting hive's daemon as
`dotf secrets run --only NAN_API_KEY -- hive serve`. The cost was written down in the drop-in: the daemon
cannot start before Bitwarden is unlocked, so it retries every 30 s until it can.

## Problem

Only hive's worker reads that key. The vault tools that every agent session uses need no secret. After each
reboot the vault was down for every agent until the operator remembered to unlock: on msi, 188 failed starts in
96 minutes. A session that started in that window got `CONNECT_TIMEOUT` for hive and stayed without it after the
unlock. The trade was recorded as a cost of AC7. In practice the key was optional, and the wait made it gate
the daemon's main service.

## Solution

`dotf secrets run --skip-locked` runs the child without the selected secrets when the vault is locked, and only
then. A typo, a wrong key or an empty value still refuses. When `dotf` is a systemd unit's main process, it
leaves a trace under `XDG_RUNTIME_DIR`. `dotf secrets unlock` restarts each traced unit with its secrets, and
`dotf doctor` fails the hive backend while the trace exists. A connected `hive client` survived the restart
(measured), so the unlock costs open sessions nothing (#3005).

When a secret blocks a process from starting, ask which part of the process reads it. If the main service does
not, start without the secret and restart when it arrives. A watched trace replaces the silent wait.
