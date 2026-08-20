---
tags: [spec, tasks]
created: "2026-06-03"
---

# Tasks - AI-022-hive-daemon-activation

> One task = one focused commit. Tick as you go.

## Setup

- [x] Branch (worktree) created from main: `feat/hive-daemon-activation`
- [x] `proposal.md` complete; acceptance criteria testable
- [x] No open questions left (the skip-if-present migration + old-hive-hang risks are resolved in-design)

## Implementation

- [x] `mcp-servers.json`: hive `args` `uvx hive-vault` -> `hive client` (+ `_history` note)
- [x] `setup-linux.sh`: post-MCP-loop activation block — version-gate (>=1.32.0 via `uv tool list` + `sort -V`), migrate stale `uvx hive-vault` entry (snapshot/restore-wrapped), `hive service install` (non-fatal)
- [x] `setup-windows.ps1`: mirror block — version-gate via `[version]`, migrate via `Backup-AndRestoreClaudeJson`, `hive service install` (non-fatal)
- [x] `bash -n` + shellcheck clean on the added block

## Closing

- [x] Each acceptance criterion covered by a `features.json` entry
- [x] Lint passes (shellcheck adds no new findings in the block)
- [x] No unrelated changes (diff = 3 files, 65 insertions)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder
- [ ] (post-merge) Multi-machine validation by the maintainer, then tick hive #176

## Follow-on: Windows-safe upgrade orchestration (ADR-015 / hive#176)

The first real Windows rollout validation (2026-06-04) found the daemon's
restart-on-upgrade broken on Windows three ways: `uv tool upgrade` cannot replace
a running exe (os error 32), Task Scheduler `RestartOnFailure` ignores the exit-75
drift stop, and the task showed a console window every logon. Fixed across hive
(PR #207 — S4U principal + supervisor loop) and this repo:

- [x] `windows/hive-upgrade.ps1`: orchestrated stop-before-upgrade — only-if-newer
      -> defer-if-locked -> stop daemon -> `uv tool upgrade` (never `--reinstall`)
      -> start daemon. Runs from PowerShell, so it holds no lock on the install.
- [x] `setup-windows.ps1`: `DotfilesHiveUpgrade` runs the orchestration script on a
      15-min cadence (Linux `OnCalendar=*:0/15` parity, replacing the daily-9:00
      trigger whose ~24h lag was observed live). Idempotent on Execute/Args drift.
- [x] `tests/hive-upgrade-timer.bats`: orchestration contract + 15-min wiring (24/24).
- [x] Validated on real hardware: a genuine `1.32.4 -> 1.33.0` release ran clean.
- [ ] (post-merge) activate the Windows daemon, then tick hive #176.

Decision: ADR-015 (hive) — A1 orchestrated stop-before-upgrade, NOT A3 junction-swap
(over-engineering for a low-ROI Windows daemon). The residual OS limit (replacing a
loaded native module while a session holds it) is tracked upstream in uv
(astral-sh/uv#8528, #11930, #11134).

> **SUPERSEDED 2026-08-07 — hive chose A3 and shipped it.** The decision above was
> recorded here on 2026-06-05. Three weeks later hive ran the A3 spike it had
> deferred and it **passed** on real non-admin Windows (2026-06-24,
> `hive/specs/HIVE-267-upgrade-swap/verification.md`): a junction was created
> without admin, and `current` was repointed while the old version was locked with
> no "Access is denied". hive then implemented A3 (`src/hive/_runtime.py`,
> `hive self-upgrade`) and published it in **1.43.0**.
>
> So the "over-engineering" judgement was overtaken by evidence, and neither repo
> noticed: hive shipped A3 while this repo kept running A1, and the two specs
> contradicted each other in writing for six weeks. That drift is itself the
> lesson — a decision recorded in repo A about a mechanism owned by repo B needs a
> pointer back, or it silently becomes fiction.
>
> **Current state:** A3 is the Windows mechanism. Linux keeps `uv tool` (see below).
> The migration of this repo's three install sites is **AI-028** (dotfiles#791);
> its PR2 is gated on hive#328, because `self_upgrade` builds the versioned layout
> but installs no launcher on PATH.
>
> **Linux decision (AI-028 AC6), recorded explicitly so it is a choice and not an
> omission:** Linux **keeps `uv tool install --upgrade`**. A3 exists to work around
> Windows' inability to replace an in-use executable; POSIX has no such constraint,
> so the versioned-dir + junction machinery would buy nothing and add a second
> install model to maintain. `ai/hermes/setup.sh:83` and `setup-linux.sh` are
> unchanged by AI-028.

## Machine-readable features

See sibling `features.json`. Structural/static checks (JSON shape, syntax, lint,
block presence); the dynamic migration + service-install behavior is validated by
the hive runbook `docs/runbooks/daemon-activation.md` (one machine done) and the
maintainer's multi-machine rollout. The agent must NOT set `state: passing` —
only the harness may, after a clean run.
