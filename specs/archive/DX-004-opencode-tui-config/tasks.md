---
tags: [spec, tasks, templates]
created: "2026-06-05"
---

# Tasks - DX-004-opencode-tui-config

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `spec/dx-004-opencode-tui`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] No open questions left unresolved (caveats captured under Risks)

## Implementation

- [x] AC1: add `interleaved: { field: "reasoning_content" }` to the 4 NaN models in `ai/opencode/opencode.jsonc`
- [x] AC5: update the stale reasoning comment in `opencode.jsonc` (1.15.10 → 1.15.13 + `interleaved`)
- [x] AC2: create `ai/opencode/tui.json` SSOT (theme=opencode, keybinds.display_thinking=ctrl+o, correct `$schema`)
- [x] AC3: add `tui.json` plain-copy deploy block to `setup-linux.sh` (reconcile-not-skip; NO secret substitution)
- [x] AC4: add `tui.json` deploy block to `setup-windows.ps1` (ASCII-only, parity)
- [x] Guard tests: extend `tests/opencode.bats` (AC1, AC2, AC3) + `tests/setup-windows.bats` (AC4)
- [x] `features.json` emitted with executable verification per AC

## Closing

- [x] Every AC (1-5) covered by at least one test; AC6 is user-empirical
- [x] JSONC/JSON validated; shellcheck on setup-linux.sh
- [x] `verification.md` filled with evidence (test output + AC6 user-confirmed 2026-06-05)
- [x] PR opened referencing this spec folder (PR #226, commit `9e31ae3`)
