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
2. The `deepseek-chat` model limits defined in `models.json` did not match the provider catalog `dotf doctor` compares against, so it emitted limit drift warnings during setup.

## What

1. Sets `HideWindow: true` (SW_HIDE through STARTF_USESHOWWINDOW) next to the existing `CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS` creation flags in `bwServeDetachAttr()`, `cli/internal/secrets/bwserve_windows.go`, so the Scoop shim that launches `bw serve` renders no visible console window. `CREATE_NO_WINDOW` (0x08000000) was the first plan and is deliberately not used: Windows ignores it when combined with `DETACHED_PROCESS` (lesson-333). Amended 2026-10-03 after the independent review found the proposal still named the flag the code had rejected.
2. Corrects the `contextWindow` and `maxTokens` of `deepseek/deepseek-chat` in `ai/pi/models.json` to the provider catalog's values, 163840 and 16384. The first attempt used 128000 and 16000, which sit below the catalog and still made `dotf doctor` warn (found by the independent review, 2026-10-03).

## Out of scope

- Updating the older `hive` binary (should be handled via external update).
- Automatically removing duplicated winged/scoop `opencode` PATH entries.

## Risks / open questions

- The effect needs a rebuilt CLI before it is seen (`dotf tools install` after the release).
- Whether `HideWindow` suppresses the Scoop shim's window can only be observed on a Windows box; CI can show that the child has no console, not that no window appeared.

## Acceptance criteria

- [x] Console window is suppressed for `bw serve` via process attributes.
- [x] Deepseek limits conform to OpenRouter's API limits.
- [ ] Next run of `dotf doctor` on Windows is clean.

## References

- Bitácora board: mlorentedev/dotfiles#1968
