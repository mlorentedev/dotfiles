---
id: "AI-032-pi-settings-field-sync"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-27"
issue: "mlorentedev/dotfiles#1247"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# AI-032-pi-settings-field-sync

> **Naming**: file lives at `<repo>/specs/AI-032-pi-settings-field-sync/proposal.md`. `AI-032-pi-settings-field-sync` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #1247: sync enabledModels into an existing ~/.pi/agent/settings.json without clobbering pi's own runtime fields -->

`ai/pi/settings.json` is seed-if-missing (lesson-150, #754/#756): pi rewrites `theme`,
`lastChangelogVersion` and the TUI-picked `defaultModel` at runtime, so a whole-file sync
was a real bug the seed-if-missing fix correctly closed. The side effect: `enabledModels`
is genuinely dotfiles-owned (nothing in pi's own runtime mutates it), but it rides along
in that seed-once file, so a catalog addition in the repo (e.g. AI-033's qwen3.8-flash /
glm5.3-flash, #1254) never reaches a machine that already has `~/.pi/agent/settings.json`
— which is nearly every machine past first setup. The only current recourse is deleting
the live file and re-running setup, which also discards the user's real theme/model
choice. This is exactly the deploy-time gap #1256's `checkModelPins` doctor check found
adjacent to (drift in the OTHER direction — stale/invalid entries) and deliberately left
unfixed, "asked, not defaulted," because removing a live entry is a judgment call. Adding
a repo-curated entry is not: it can never surprise or destroy anything, which is the
distinction that makes automating it safe.

## What

On every `setup-linux.sh` / `setup-windows.ps1` run, if `~/.pi/agent/settings.json`
already exists, its `enabledModels` array is overwritten to match the repo's
`ai/pi/settings.json` exactly — nothing else in the file is touched. A pre-existing
`theme`, `lastChangelogVersion`, `defaultModel`, or any other field survives value-for-value
(both `jq` and `ConvertTo-Json` re-serialize the whole file, so indentation/whitespace can
change; the values pi actually reads do not).
If the two arrays already match, nothing is written (idempotent, `changed=0`).

## Out of scope

- Removing/repairing invalid or stale entries already in a deployed `enabledModels`
  (e.g. a retired snapshot id) — that is #1256's `checkModelPins` doctor check, which
  deliberately does not write, for the opposite reason this change safely can
- Any change to `defaultModel`, `theme`, or `lastChangelogVersion` — these remain
  exclusively pi's own runtime state, never written by setup
- A `dotf doctor` visibility check for this same drift (so `dotf doctor` alone, without a
  full setup re-run, could report a stale picker) — complementary, not required for this
  fix, and not built here to keep scope to the actual gap

## Risks / open questions

- The merge is naive array replacement, not a set union — if a user had ever hand-edited
  `enabledModels` in the live file (nothing in this repo does that; the array is
  documented as dotfiles-owned), that edit would be overwritten. Accepted: matches how
  `packages.json`'s reconcile already treats the neighbouring `packages` array in the
  same file.
- PowerShell's `ConvertTo-Json` unwraps a single-element array when piped, which would
  make a 1-model `enabledModels` compare unequal to itself after a round-trip. Resolved
  by using `-InputObject` (parameter binding, not a pipe) for every comparison and write.

## Acceptance criteria

- [ ] `setup-linux.sh`: on a pre-existing `~/.pi/agent/settings.json` with a stale
      `enabledModels`, the array converges to the repo's list on the next run
- [ ] Same run's `theme` / `lastChangelogVersion` / `defaultModel` are provably untouched
- [ ] A second consecutive run makes no further change (idempotent)
- [ ] `setup-windows.ps1` carries the same guarantees (Linux parity)
- [ ] `tests/pi-config.bats` exercises the real block from both scripts (not a
      reimplementation), so the test cannot drift from the shipped behavior

## References

- Bitácora board: #1247 (this spec's work-gate issue)
- Related: #1254 / AI-033 (the catalog addition this unblocks for existing machines),
  #1256 / HARNESS-067 (`checkModelPins`, the adjacent stale-entry detector this
  deliberately does not overlap with)
- `docs/lessons/lesson-150-a-config-file-the-tool-itself-rewrites-must-be-see.md` — why
  `ai/pi/settings.json` is seed-if-missing, the constraint this change works within
