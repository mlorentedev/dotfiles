---
id: "AI-022-hive-daemon-activation"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-06-03"
tags: [spec, proposal]
template_version: "1.0"
---

# AI-022-hive-daemon-activation

> Cross-repo implementation arm of hive Phase C activation. Tracking SSOT is
> `mlorentedev/hive` issue #176 + spec `specs/HIVE-118-phase-c-daemon-model/`.
> This dotfiles change is the fleet rollout the hive spec gates on.

## Why

Hive shipped the Phase C daemon model (single-owner `hive serve` + thin
`hive client` stdio shim + per-user supervised service + restart-on-upgrade) in
v1.27.0–v1.32.0, but every machine still connects via the per-session
`uvx hive-vault` stdio server. Building the daemon is pointless if no machine
uses it. This change flips the fleet to the daemon so the maintainer gets
single-owner lifecycle, zero cold-starts, and cross-session observability — with
the in-process `hive client` fallback as the safety net (a missing/failed daemon
degrades, never breaks a session).

## What

After `setup-linux.sh` / `setup-windows.ps1` runs, the machine's Claude Code
`hive` MCP entry is `hive client` (daemon proxy) instead of `uvx hive-vault`, and
`hive serve` is installed as a supervised per-user service (systemd `--user` on
Linux, Task Scheduler on Windows) that auto-starts at login and restarts on
failure. Existing machines (already registered as `uvx hive-vault`) are migrated,
not just newly-provisioned ones.

## Out of scope

- macOS / launchd activation (hive ships it as a non-zero stub; follow-up).
- Any change to hive itself (the daemon, CLI, and service installer already shipped on PyPI ≥ 1.32.0).
- Telemetry/observability surface beyond what hive already exposes (`/health`, `/status`).

## Risks / open questions

- **Skip-if-present idempotence** — the MCP registration loop skips a server that
  is already registered, so an existing `uvx hive-vault` entry is left untouched.
  Resolved: a dedicated post-loop activation block migrates it (`mcp remove` +
  `mcp add … hive client`, snapshot/restore-wrapped per BUG-011 / #59870).
- **Old hive hangs on `hive service`** — a hive < 1.32.0 routes an unknown
  subcommand to the blocking stdio server. Resolved: the block gates on the
  installed package version (`uv tool list` + `sort -V` / `[version]`), never by
  probing `hive service`.
- **No systemd `--user` / Task Scheduler** (headless/CI) — `hive service install`
  fails; resolved by treating it as non-fatal (warning), client still works via
  fallback.
- **Blast radius** — changes the daily Claude↔hive connection on every machine.
  Mitigated by the H1 in-process fallback and a one-machine manual validation
  (hive `docs/runbooks/daemon-activation.md`) before this rollout.

## Acceptance criteria

- [ ] `mcp-servers.json` hive `args` is `hive client` (fresh installs register the daemon proxy).
- [ ] On a machine still registered as `uvx hive-vault`, setup migrates the entry to `hive client`.
- [ ] Setup installs + enables the supervised `hive serve` service (Linux systemd `--user`, Windows Scheduled Task) when hive ≥ 1.32.0.
- [ ] A hive < 1.32.0, or a missing systemd/Task Scheduler, leaves the stdio entry intact with a warning (setup never fails on it).
- [ ] `setup-linux.sh` passes `bash -n` + shellcheck; the activation block adds no new findings.

## References

- hive issue: [mlorentedev/hive#176](https://github.com/mlorentedev/hive/issues/176)
- hive spec: `specs/HIVE-118-phase-c-daemon-model/` (in the hive repo)
- hive runbook: `docs/runbooks/daemon-activation.md` (the manual procedure this automates)
- BUG-011 / anthropics/claude-code#59870 (the `.claude.json` truncation guard this reuses)
