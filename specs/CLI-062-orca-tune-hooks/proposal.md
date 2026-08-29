---
id: "CLI-062-orca-tune-hooks"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-29"
issue: "mlorentedev/dotfiles#1338"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-062-orca-tune-hooks

> **Naming**: file lives at `<repo>/specs/CLI-062-orca-tune-hooks/proposal.md`. `CLI-062-orca-tune-hooks` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

`scripts/orca-tune.sh` (101 LOC) has no callers, is deployed by neither setup,
has no test, and its port — `dotf orca tune` — landed in #1274; ADR-020 §5
says the shell pair goes in the same PR, and it did not. Its misnamed twin,
`scripts/orca-hook-tune.ps1`, is a *different* function: the DX-006 repair of
Orca's generated Copilot hooks (bump `timeoutSec` in `~/.copilot/hooks/orca.json`,
swap the slow `Invoke-WebRequest` POST in `~/.orca/agent-hooks/copilot-hook.ps1`
for `HttpWebRequest`), which Orca re-breaks on every upgrade and
`setup-windows.ps1` re-applies. `dotf doctor` already detects both signals and
`--fix` already applies the timeout half in Go; the script half still lives
only in PowerShell, and doctor's remedy text points at the script.

## What

- `dotf orca tune-hooks [--check] [--timeout-sec 30] [--hook-config <path>]
  [--hook-script <path>]`: the whole DX-006 repair in Go. Both files backed up
  beside themselves (`.bak.<stamp>`) and written atomically; a missing pair is
  "nothing to do" (exit 0); `--check` reports drift and exits non-zero while
  any remains; an unrecognised POST line is reported and left alone.
- `setup-windows.ps1` runs `dotf orca tune-hooks` where it ran the script, and
  sweeps the script from the deployed scripts directory (WIN-013's retired list).
- `dotf doctor --fix` applies the script half too, through the same package,
  and both remedy lines name the command.
- Deleted: `scripts/orca-hook-tune.ps1`, `scripts/orca-tune.sh`,
  `tests/orca-hook-tune-ps1.bats`, `tests/orca-hook-tune.Tests.ps1`.
  `docs/adr/audit-007` rows amended to "ported".
- The Pester cases become Go tests; the setup-windows bats asserts the call.

## Out of scope

- `dotf orca tune` (orca-data.json baseline) — unchanged.
- Linux: Orca's Copilot hooks are a Windows artefact; the command is
  OS-agnostic (paths are flags) but nothing on Linux invokes it.

## Risks / open questions

- The `HttpWebRequest` replacement block must be byte-for-byte what the script
  wrote, so a box already tuned reads as clean. RESOLVED: the block is the
  script's, indentation captured from the matched line, pinned by a test
  against a copilot-hook.ps1 fixture.
- Orca's upgrade timer re-generates both files. RESOLVED: unchanged from the
  script — setup re-applies; doctor detects; nothing here changes the cadence.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] AC1 — `dotf orca tune-hooks` bumps every `timeoutSec` below the floor in the hook config and swaps the POST in the hook script, with a timestamped backup beside each file it changes; a second run changes nothing.
- [x] AC2 — `--check` exits non-zero and names the drift while either file drifts, zero once clean; a missing pair exits zero with "nothing to do"; a generous timeout is left as is.
- [x] AC3 — `dotf doctor --fix` repairs both signals, and both FAIL lines name `dotf orca tune-hooks`.
- [x] AC4 — `setup-windows.ps1` invokes `dotf orca tune-hooks` and lists `orca-hook-tune.ps1` among the retired scripts; the four files are deleted; audit-007 says ported.
- [x] AC5 — on the Windows work box: `dotf orca tune-hooks --check` against the real files exits 0 (already tuned), and against scratch copies with `timeoutSec: 5` + `Invoke-WebRequest` the command tunes them and `--check` then passes.

## References

- Bitácora board: #1338. #1274 (`dotf orca tune`), #442, DX-006 (lesson 111), ADR-020 §5, `docs/adr/audit-007-cli-convergence-state.md:109`.
