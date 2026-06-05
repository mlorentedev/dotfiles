---
id: "DX-004-opencode-tui-config"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-05"
tags: [spec, proposal]
template_version: "1.0"
---

# DX-004-opencode-tui-config

> **Naming**: file lives at `<repo>/specs/DX-004-opencode-tui-config/proposal.md`.

## Why

<!-- from 11-tasks.md: (GH #225): opencode reasoning visibility + TUI config. (1) Per-model `interleaved: {field: "reasoning_content"}` in opencode.jsonc (4 NaN models) — maps NaN's non-standard reasoning field into a reasoning part opencode renders. (2) New SSOT `ai/opencode/tui.json` (theme=opencode, keybind `display_thinking`=ctrl+o) + cross-platform deploy in setup-{linux.sh,windows.ps1} + bats. Solves the invisible-reasoning "is it hung?" gap (1.15.13 schema has `interleaved`; SDK stream emission needs empirical TUI verify). Spec: specs/DX-004-opencode-tui-config/. -->

opencode looks frozen while a NaN model reasons: NaN streams its chain in the
non-standard `reasoning_content` field, which `@ai-sdk/openai-compatible` does
not map to a renderable reasoning part, so the TUI shows nothing during the
silent reasoning phase. Separately, opencode's TUI config (theme, keybinds) is
not under version control, so a fresh machine has no reproducible TUI setup and
the reasoning-visibility toggle (`display_thinking`, unbound by default) is not
wired. This feature makes the reasoning visible and puts the TUI config in the
dotfiles SSOT.

## What

After this PR:

- Each NaN model in `opencode.jsonc` carries `interleaved: { field: "reasoning_content" }`,
  so opencode ingests NaN's reasoning chain and can render it as a reasoning block.
- A new versioned SSOT `ai/opencode/tui.json` sets `theme: "opencode"` and binds
  `keybinds.display_thinking` to `ctrl+o`, so the user toggles reasoning
  visibility in the TUI (also available via the `/thinking` command).
- `setup-linux.sh` and `setup-windows.ps1` deploy `tui.json` to
  `~/.config/opencode/tui.json` (plain copy — no secret substitution, unlike
  `opencode.jsonc`).
- The stale comment in `opencode.jsonc` (claiming opencode renders neither NaN
  reasoning field) is corrected to reflect the `interleaved` fix on 1.15.13.

## Out of scope

- Disabling thinking / changing the default model — handled by #223 (default → qwen3.6, relaxed timeouts).
- The DX bundle keys (share/autoupdate/disabled_providers/tool_output/agent.plan) — handled by #224.
- An in-TUI elapsed-time / token counter — upstream feature requests (opencode #5872, #12028), not configurable today.
- `reasoningEffort` / `reasoningSummary` / `options.reasoning` on NaN — native-only; openai-compatible providers reject them (opencode #13546).
- A healthcheck drift assertion for `tui.json` — noted as an optional follow-up (the file is a plain copy; drift detection can be added later if it matters).

## Risks / open questions

- **SDK emission unverified.** `interleaved` is present in the 1.15.13 JSON schema, but it is not byte-confirmed that the bundled AI SDK actually emits the reasoning part on stream. Mitigation: empirical in-TUI verification (AC6); fallback is the existing `dbg` / `qf` wrappers and the qwen3.6 fast default from #223.
- NaN's other reasoning field, `provider_specific_fields.reasoning`, is NOT covered by the `interleaved` enum (only `reasoning_content` / `reasoning_details`) — so we map `reasoning_content`.
- `ctrl+o` could collide with an existing opencode bind — verify in the TUI; `leader+t` or `/thinking`-only are fallbacks.
- `tui.json` uses a different schema URL (`https://opencode.ai/tui.json`, not `config.json`) — get the `$schema` right.
- Windows: `setup-windows.ps1` must deploy as a plain copy and stay ASCII-only (PSScriptAnalyzer fails CI on non-ASCII without BOM — `pattern-powershell-ascii-only`).

## Acceptance criteria

- [ ] AC1: `ai/opencode/opencode.jsonc` has `interleaved: { field: "reasoning_content" }` on all 4 NaN chat models (deepseek-v4-flash, qwen3.6, gemma4, mimo-v2.5).
- [ ] AC2: `ai/opencode/tui.json` exists, is valid JSON, with `theme: "opencode"` and `keybinds.display_thinking: "ctrl+o"`.
- [ ] AC3: `setup-linux.sh` deploys `tui.json` to `~/.config/opencode/tui.json` as a plain copy (no `substitute_env_placeholders`).
- [ ] AC4: `setup-windows.ps1` deploys `tui.json` (cross-OS parity) and is ASCII-only.
- [ ] AC5: the stale `opencode.jsonc` reasoning comment is updated to reflect the `interleaved` fix and version 1.15.13.
- [ ] AC6 (empirical, user-run): in the TUI, the NaN reasoning chain becomes visible after pressing `ctrl+o` (or `/thinking`). Evidence recorded in `verification.md`.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` (DX-004 entry)
- GH issue: #225
- Related PRs: #223 (SSE timeouts + default model), #224 (DX config bundle)
- opencode schema: `https://opencode.ai/config.json` (`interleaved` enum), `https://opencode.ai/tui.json`
- opencode docs: `/docs/keybinds` (`display_thinking`), `/docs/tui` (`/thinking`)
- Pattern: `00_meta/patterns/pattern-powershell-ascii-only.md`
