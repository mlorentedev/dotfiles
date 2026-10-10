---
id: "lesson-380-a-check-keyed-on-the-supervisor-skips-the-host-without-one"
type: lesson
status: active
title: "A check keyed on the supervisor skips the host without one"
created: "2026-10-10"
---

# A check keyed on the supervisor skips the host without one

## Context
Every agent registers hive as `hive client`, a stdio shim that fails outright when no `hive serve`
daemon answers. Doctor's only hive check read the daemon's systemd unit, and it opened with "no
`systemctl` here, skip".

## The Trap
On the Mac mini no daemon ran, because macOS has no hive supervisor yet. Every Claude session's hive
MCP was `CONNECTION_CLOSED`, and `dotf doctor` printed `[INFO] skipped: no systemctl on this host`
(#2231). The check asked the supervisor whether the service was fine. On a host with no supervisor,
nobody could answer that question, so the check answered "not applicable". But the thing that
depended on the daemon, the MCP registration, applied everywhere.

## The Solution
A second check gates on the dependency, not the supervisor: `mcp-servers.json` declares
`hive client` and its prerequisite is installed. It then asks the service itself, through hive's own
`hive service status`, which proves the listener against the daemon's pinned certificate on every
OS. Its verdict line has three outcomes: healthy, a named failure state, or no verdict. "No verdict"
is reported as unknown, never as down. The fix it names depends on the OS, and on macOS it says that
there is no supervisor instead of naming a setup script.

## Takeaways
- **Gate a liveness check on who depends on the service, not on what supervises it.** "No supervisor
  here" is the most likely reason the service is down, so it is the worst reason to skip the check.
- **Probe through the service's own health command when it has one.** Re-deriving its port and
  certificate pin in the checker is logic that drifts away from the service's.
- Same class as BUG-114 (#2224): a guarantee whose only writer does not run on darwin.
