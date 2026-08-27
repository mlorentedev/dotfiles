---
id: "WIN-007-harness-mirror"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-08-27"
issue: "mlorentedev/dotfiles#1288"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# WIN-007-harness-mirror

> **Naming**: file lives at `<repo>/specs/WIN-007-harness-mirror/proposal.md`. `WIN-007-harness-mirror` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #1288: WIN-007: setup-windows.ps1 never mirrors harness/ into ~/.dotfiles, so doctor fails on model-map and model-pins and its printed remedy can never clear -->

Measured 2026-08-27 on the Windows work box: after a clean `.\setup-windows.ps1`, `dotf doctor` FAILs the routing registry and the model-pin check because `~/.dotfiles/harness/` does not exist, and both messages print *"re-run setup to mirror it"* — a remedy that cannot clear them, because `setup-windows.ps1` never mirrors `harness/`. Linux carries a 60-line bash+jq block for exactly this (with a zsh word-split hazard and a "jq not on PATH yet" race it documents itself); Windows has nothing; and the Go doctor records the false belief that Windows has no mirror at all and early-returns its orphan check on that belief. Two shell implementations of one mirror is the drift class ADR-020 exists to remove, and the first slice of the CLI-026/CLI-035 engine port that pays for itself today.

## What

- `dotf harness mirror` (Go, `cli/internal/harness/mirror.go`): copies the whole `harness/` tree and every file `harness/manifest.json` declares in `.targets[].file` from the checkout into `$DOTFILES_DIR`, preserving relative paths. Idempotent — a file whose bytes match is left untouched, mtime included — and it reports `N updated, M unchanged` so a setup run has its `changed=0` evidence (#1266). It never prunes (`dotf doctor --fix` owns orphans, #802). A declared target the checkout lacks is named on stderr and the command exits 1 **after** mirroring everything else.
- Both setup scripts call it, at the position the bash block occupied on Linux (after `compile-harness.sh --refresh`) and beside the other `dotf` calls on Windows. The bash+jq block in `setup-linux.sh` is deleted, not duplicated.
- `dotf doctor`: `checkHarnessMirrorOrphans` stops special-casing Windows (its stale comment goes with it), and the pi-packages count reads `ai/pi/packages.json` checkout-first like every other registry read (it reported "0 packages declared" as a PASS on Windows).

## Out of scope

- `dotf harness {refresh,deploy,check}` — the engine port is CLI-026 (#495) / CLI-035 (#909); this is the mirror slice only.
- Pruning the mirror — `dotf doctor --fix` (#802 semantic).
- The copilot catalog CRLF writer (WIN-008) and the non-recursive `sensitive/` copy (WIN-011): their PowerShell halves ship in the PowerShell hygiene PR.
- Mirroring `ai/` wholesale: nothing reads it from the mirror once the pi-packages count is checkout-first.

## Risks / open questions

- On Linux `~/.local/bin` may not be on PATH in the setup process (the jq comment in the old block). Resolved: the call uses the same `command -v dotf` guard as `dotf tools install` two hundred lines earlier, degrades with a warning naming the consequence, and `tests/verify-setup.bats` ("every harness manifest target exists in the deploy dir") fails on the resulting gap in the integration job.
- Windows CI never runs `dotf` today (TEST-003/#1298), so the Windows call site is exercised end-to-end only after that lands. Accepted: the behaviour is covered by the Go tests, which run on the `cli` workflow's Windows matrix, and by the `test-windows` bats source assertion on the call site.
- Order on Linux matters: the mirror must run after `--refresh` so the snapshot matches the refreshed repo. Resolved: the call replaces the block in place.

## Acceptance criteria

- [ ] AC1 — `dotf harness mirror` copies `harness/` and every `.targets[].file` into the deploy dir; a re-run reports `0 updated` and rewrites nothing (mtime unchanged).
- [ ] AC2 — a declared target the checkout lacks is named on stderr and the command exits 1, with everything else still mirrored.
- [ ] AC3 — a file only the deploy dir has survives a mirror run (no pruning).
- [ ] AC4 — both `setup-linux.sh` and `setup-windows.ps1` call `dotf harness mirror`, and `setup-linux.sh` no longer carries the bash+jq mirror block.
- [ ] AC5 — `dotf doctor` reports harness mirror orphans on Windows as on Linux, and the pi-packages count reads the checkout first.

## References

- Bitácora: #1288 (WIN-007); parent port: #495 (CLI-026), #909 (CLI-035); Linux precedent: #1200 (the mirror evaluated a target setup never copied).
- ADR-020 (strangler-fig on contact), ADR-030 (checkout-first precedence), #802 (doctor --fix prunes, setup only copies).
- `00_meta/patterns/pattern-setup-script-idempotence.md`.
