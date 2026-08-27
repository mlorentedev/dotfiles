---
id: "CLI-051-orca-config-sync"
type: spec
status: implementing
created: "2026-08-27"
issue: "mlorentedev/dotfiles#1273"
tags: [spec, proposal, orca, cli, deploy]
template_version: "1.0"
---

# CLI-051: Orca ADE Keybindings Deployment and Bidirectional Settings Capture

## Why

Orca ADE configuration was previously split between shell-based tuning scripts (`scripts/orca-tune.sh`) and untracked user directories (`~/.orca/keybindings.json`). In-app settings and keybindings customizations had no clean export workflow back into git without risking the exposure of dynamic session databases, terminal histories, or ephemeral tokens. This spec formalizes Orca configuration management into the `dotf` CLI ecosystem per ADR-020 (Go-owned tooling).

## What

1. **Declarative Keybindings Deployment**: Add `ai/orca/keybindings.json` and register `orca-keybindings` in `ai/deploy.json` so `dotf deploy` installs it idempotently.
2. **`dotf orca export`**: Extracts clean settings from `~/.config/orca/orca-data.json` to `ai/orca/settings.json` and updates `ai/orca/keybindings.json` from `~/.orca/keybindings.json`.
3. **`dotf orca tune`**: Ports the baseline tuning logic from `orca-tune.sh` into Go, applying memory-saving and telemetry opt-out defaults to `orca-data.json` safely with active process detection and timestamped backups.
4. **Health Diagnostics**: Extends `dotf doctor` checks to verify Orca keybindings deployment.
5. **Durable Documentation**: Updates `ai/orca/ORCA.md` and CLI documentation so future sessions discover the synchronization workflow.

## Out of scope

- Managing ephemeral worktree databases (`orchestration.db`), runtime sockets, or active session cookies.
- Creating a proprietary GUI editor for Orca configuration.

## Risks / open questions

- **Process Collision**: `orca-data.json` can be overwritten by a running Orca instance on exit.
  - *Mitigation*: `dotf orca tune` refuses to write when an active Orca process is running (matching `orca-tune.sh` guard semantics).
- **Schema Drift**: New Orca releases might introduce new top-level keys in `orca-data.json`.
  - *Mitigation*: The export command targets the `settings` object specifically and preserves custom overrides.

## Acceptance criteria

- [x] `ai/orca/keybindings.json` is tracked in git and registered in `ai/deploy.json`.
- [x] `dotf deploy orca-keybindings` deploys `keybindings.json` to `~/.orca/keybindings.json`.
- [x] `dotf orca export` extracts settings and keybindings cleanly.
- [x] `dotf orca tune` supports `--dry-run` and applies the tuned baseline safely with process guards.
- [x] `dotf doctor` validates Orca keybindings and configuration state.
- [x] Unit tests cover `cli/internal/orca` and pass `go test ./...`.

## References

- Issue: [#1273](https://github.com/mlorentedev/dotfiles/issues/1273)
- ADR-020: Shell keeps thin bootstrap, Go owns tooling logic.
- ADR-025: Environment variable contract resolution in deploy.
- CLI-039: Declarative `dotf deploy` framework (`ai/deploy.json`).
