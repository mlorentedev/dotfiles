---
id: "PI-002-windows-fixes"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-10-02"
issue: "mlorentedev/dotfiles#1968"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "Fast-track windows stability (15 active, limit 10, 2026-10-02)"
---

# PI-002-windows-fixes

## Why

<!-- from issue #1968: fix: prevent bw serve window popup on Windows and fix deepseek limits -->

The Windows experience had two regressions:
1. `dotf secrets unlock` spawns a background `bw serve` process, but because it relies on a scoop shim, a visible console window was popping up and hanging in the background, annoying the user and getting killed when closed manually.
2. The `deepseek-chat` model limits defined in `models.json` were incorrectly configured beyond OpenRouter's limits (128k context, 16k output), which causes `dotf doctor` to emit limit drift warnings during setup.

## What

1. Adds the Windows `CREATE_NO_WINDOW` (0x08000000) flag alongside `HideWindow: true` in the `bwServeDetachAttr()` syscall properties in `cli/internal/secrets/bwserve_windows.go`. This guarantees the Node/Scoop shim processes execute headlessly without rendering a console host.
2. Corrects the `contextWindow` and `maxTokens` constraints for `deepseek-chat` in `ai/pi/models.json` to 128000 and 16000 respectively.

## Out of scope

- Updating the older `hive` binary (should be handled via external update).
- Automatically removing duplicated winged/scoop `opencode` PATH entries.

## Risks / open questions

- `CREATE_NO_WINDOW` is standard and safe, but requires the CLI to be rebuilt before the effect is seen (handled by user running `dotf tools install` after merge).

## Acceptance criteria

- [x] Console window is suppressed for `bw serve` via process attributes.
- [x] Deepseek limits conform to OpenRouter's API limits.
- [ ] Next run of `dotf doctor` on Windows is clean.

## References

- Bitácora board: mlorentedev/dotfiles#1968
