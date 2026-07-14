---
id: "BUG-031-windows-project-key"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-07-14"
issue: "mlorentedev/dotfiles#689"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, memlink, project-key, powershell, adr-020, fresh-machine]
template_version: "1.0"
---

# BUG-031-windows-project-key

Make Go the single source of the Claude auto-memory project-key encoding, have
the Windows PowerShell twins call it (with a corrected fallback), and fix the
`--all` binding so the weekly vault-maintenance task actually runs.

## Why

The Claude auto-memory junction encoding **drifted** between the Go layer and its
PowerShell twins. `memlink.ClaudeProjectKey` (`cli/internal/memlink/memlink.go:105`,
the #551/HARNESS-040 fix) maps `/`, `\` **and** the drive `:` each to `-`,
producing Claude Code's real key — `C:\Users\me\p` -> `C--Users-me-p`. Two
PowerShell sites still map only `\`->`-` and **delete** the `:`
(`.Replace('\','-').Replace(':','')`), producing `C-Users-me-p` — a key Claude
never reads. On Windows this means:

- `setup-windows.ps1:871` builds the junction at the wrong path and prints a
  misleading `Linked auto-memory` — junk junctions Claude ignores (self-healed
  later by `dotf mem session-start`, which uses the correct key).
- `knowledge-crystallize.ps1:57` (`Get-EncodedPath`) resolves a nonexistent path
  in single-project mode and warns `No MEMORY.md found` for every project.

Separately, `vault-maintenance-weekly.ps1:23` invokes the crystallizer with
`--all` (a POSIX double-dash). PowerShell binds the literal `--all` positionally
to `$ProjectDir`; the declared `[switch]$All` stays `$false`, so the Sunday
`DotfilesVaultMaintenance` scheduled task has **never** processed a project —
it single-project-resolves `--all` and throws.

Root cause of the encoding half: **one datum (the project-key encoding) duplicated
across Go + 2 PowerShell sites**, which drifted when #551 fixed only Go. Re-syncing
three copies would reintroduce the same drift class; per ADR-020 (Go owns tooling
logic) and Standing Order #2 (SSOT), the fix is to let Go own the encoding and have
the twins call it. This is the #551 fix the setup twin never received.

Source: issue #689 (audit codebase-audit-2026-07-06 findings C16, C11, C5).

## What

- **Go owns the datum.** New `dotf mem project-key <path>` subcommand prints
  `memlink.ClaudeProjectKey(path)`. The encoding now has one implementation.
- **PowerShell calls Go, with a corrected fallback.** `setup-windows.ps1` and
  `knowledge-crystallize.ps1` obtain the key via `dotf mem project-key`, guarded on
  `dotf` being on PATH (the existing `if (Get-Command dotf ...)` pattern at
  `setup-windows.ps1:605/619/645`). When `dotf` is absent the inline fallback uses
  the **corrected** encoding `.Replace('\','-').Replace(':','-')` (matches Go), so
  bootstrap never hard-depends on the CLI yet never re-emits the buggy key.
- **Decoder follows the encoder.** `knowledge-crystallize.ps1`'s `Get-DecodedPath`
  is updated from the single-dash assumption (`^([A-Za-z])-`) to the real
  double-dash key (`^([A-Za-z])--`), so `--all` discovery maps real
  `~/.claude/projects/*` dirs back to filesystem paths. The wrong `Get-EncodedPath`
  comment ("strip the ':'") is corrected.
- **`--all` -> `-All`.** `vault-maintenance-weekly.ps1` invokes the crystallizer
  with the PowerShell switch `-All`, so `$All` is true and the weekly task fans out
  over every project.
- **Anti-drift guard (incident -> guard).** A Pester test asserts the PS fallback
  encoder and `dotf mem project-key` both yield `C--Users-me-p` for `C:\Users\me\p`
  (== Go `ClaudeProjectKey`), so the twin can never silently re-drift.

## Out of scope

- **Porting the scripts wholesale to Go.** Bootstrap stays shell (ADR-020 C7); the
  `knowledge-crystallize` port is tracked separately (CLI-021 #490). This PR ports
  only the *encoding datum*, not the scripts.
- **The other fresh-machine bugs** (#691, #690, #688) — their own issues.
- **Cleaning junk junctions already deposited** by prior buggy runs on this machine.
  `dotf mem session-start` self-heals the correct key on next session; a one-time
  sweep of stale `C-Users-*` dirs is a possible follow-up, not this diff.

## Risks / open questions

- **`dotf` on PATH at the setup memory-loop.** Verified: `dotf` is installed at
  `setup-windows.ps1:533`, before the memory loop at :857. The `Get-Command dotf`
  guard degrades a missing CLI to the corrected inline encoder rather than failing —
  same tolerance the adjacent env/secrets steps already use.
- **Standalone scheduled runs.** `vault-maintenance-weekly` -> `knowledge-crystallize`
  rely on `dotf` on the Task Scheduler PATH; the fallback covers absence.
- **Atomicity.** The decoder change is only correct if the encoder change ships in
  the same PR — both live in this diff.
- **#734.** The Linux integration container can't exercise a new `dotf` subcommand
  until release (and there is no Windows integration container). Coverage here is Go
  unit tests for the subcommand + the Pester guard for the fallback.

## Acceptance criteria

- [ ] `dotf mem project-key 'C:\Users\me\p'` prints `C--Users-me-p`; a POSIX path
      `/home/me/p` prints `-home-me-p` (identical to `ClaudeProjectKey`).
- [ ] `setup-windows.ps1` and `knowledge-crystallize.ps1` compute the key via
      `dotf mem project-key`, with a corrected fallback when `dotf` is absent.
- [ ] `knowledge-crystallize.ps1 -All` discovers projects via the double-dash
      decoder (a known key round-trips to its path).
- [ ] `vault-maintenance-weekly.ps1` invokes the crystallizer with `-All`
      (`$All` is `$true`; the scheduled task fans out).
- [ ] Pester guard: PS fallback key == `dotf mem project-key` output == expected Go
      key for `C:\Users\me\p`.
- [ ] `go build`/`vet`/`test ./...` + golangci-lint clean; PSScriptAnalyzer clean
      (ASCII-only `.ps1`); Pester green.

## References

- GH issue: [#689](https://github.com/mlorentedev/dotfiles/issues/689)
- Prior encoding fix: #551 / HARNESS-040 (`ClaudeProjectKey`, Go side only)
- ADR-020 (Go owns user-facing tooling logic; bootstrap stays shell — C7)
- Related port: CLI-021 [#490](https://github.com/mlorentedev/dotfiles/issues/490)
  (`dotf vault` crystallize — the eventual home of `knowledge-crystallize`)
- Sibling fresh-machine bugs: #691, #690, #688
- Pattern: `00_meta/patterns/pattern-powershell-ascii-only.md`
