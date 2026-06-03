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

## Machine-readable features

See sibling `features.json`. Structural/static checks (JSON shape, syntax, lint,
block presence); the dynamic migration + service-install behavior is validated by
the hive runbook `docs/runbooks/daemon-activation.md` (one machine done) and the
maintainer's multi-machine rollout. The agent must NOT set `state: passing` —
only the harness may, after a clean run.
