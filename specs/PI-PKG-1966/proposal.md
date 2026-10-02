---
id: "PI-PKG-1966"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-10-02"
issue: "mlorentedev/dotfiles#1966"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "Fast-track hotfix for broken pi installation (15 active, limit 10, 2026-10-02)"
---

# PI-PKG-1966

## Why

<!-- from issue #1966: fix: resolve pi extension loading conflicts (typebox/mcp) -->

Antigravity (pi) 2.0 introduces a native `builtin:mcp` extension and enforces strict extension loading (requiring `typebox` to be a `peerDependency`). The existing `packages.json` manifest pinned older versions of these extensions (`pi-web-access@0.24.2`, `pi-subagents@0.56.0`) and included the now-redundant `pi-mcp-client@0.8.0`, which causes the agent to crash on startup due to conflicts.

## What

Update `ai/pi/packages.json` to bump the versions of `pi-web-access` and `pi-subagents` to their latest releases which correct the `typebox` peer dependency issue. Additionally, remove `pi-mcp-client` from the active packages array and move it to the `retire` array so it is cleanly uninstalled by `dotf pi packages apply`, allowing the native `builtin:mcp` extension to function properly.

## Out of scope

- Updates to other tools or core `pi` configuration.
- Changing the default `mcp.json` servers.

## Risks / open questions

- None. The fix has already been tested locally and verified to work correctly.

## Acceptance criteria

- [x] The `pi-mcp-client` package is removed and added to `retire`.
- [x] The `pi-web-access` and `pi-subagents` packages are updated to versions that correctly declare `typebox` as a `peerDependency`.
- [x] Running `pi -v` completes without emitting `Extension package` warnings.

## References

- Bitácora board: mlorentedev/dotfiles#1966
