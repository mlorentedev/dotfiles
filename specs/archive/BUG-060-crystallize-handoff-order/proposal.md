---
id: "BUG-060-crystallize-handoff-order"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-08"
issue: "mlorentedev/dotfiles#850"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# BUG-060-crystallize-handoff-order

> **Written after the fix, deliberately.** The defect was found and repaired inside a live
> incident (a corrupted `MEMORY.md` on `web`), and the production diff measured 58 LOC — over
> the gate's 50-LOC threshold, under every other trigger. This spec is the gate's answer:
> honest documentation of a bugfix, not a pre-implementation design. Noted so nobody reads the
> ordering as a process claim it does not make.

## Why

`knowledge-crystallize` corrupts the file it is supposed to maintain. Both twins append their
stamps (`# currentDate`, `## Last Crystallized:`) to end-of-file whenever the marker is absent,
which places them **after** the `## Session Handoff` block. HARNESS-029 requires that block to be
the last section of `MEMORY.md`: it is rewritten every session, so keeping it out of the
auto-loaded KV-cache prefix is what stops it from busting the provider prompt cache on every new
session. Because crystallize had **never been run**, the bug was latent exactly where it does most
damage — the *first* run on any compliant `MEMORY.md` — and would have hit every project, not one.

## What

`knowledge-crystallize.sh` and `.ps1` insert new sections **before** the `## Session Handoff` block
when it exists, and append only when it does not. Running crystallize on a HARNESS-029-compliant
`MEMORY.md` leaves the handoff block last, on the first run and on every subsequent run.

## Out of scope

- **The port to `dotf`.** #490 (CLI-021) owns it and its acceptance says explicitly "no twin
  deleted in this PR" — AUDIT-007 Phase B is build-beside-then-flip. Folding a ~600-line migration
  into this fix would also delay it reaching the projects being corrupted now. ADR-020 §5 is
  satisfied by sequencing, not skipped.
- **Repointing callers and docs** (`setup-linux.sh`, `setup-windows.ps1`,
  `vault-maintenance-weekly.{sh,ps1}`, vault `crystallize/SKILL.md`, `pattern-ai-memory.md`,
  `pattern-ai-protocol.md`, `CURRENT-STATE.md`). The shell stays canonical until the flip;
  repointing now would aim them at a command that is not canonical yet. That inventory belongs in
  CLI-021's `tasks.md` as the flip checklist.
- **Trimming or restructuring any existing `MEMORY.md`.** Only `web`'s was repaired, because this
  session corrupted it.

## Risks / open questions

- **Other `MEMORY.md` files may already be corrupted** by an earlier crystallize run. Mitigated by
  the evidence that crystallize had never run before today — but unverified across all 22 projects.
  Cheap follow-up: assert `## Session Handoff` is the last heading, vault-wide.
- **The PowerShell twin cannot be executed here** (`pwsh` absent on the Linux dev box), so its
  behaviour rests on the CI `test-windows` job rather than a local run. Stated, not hidden.
- **`Add-SectionBeforeHandoff` uses a regex replace with count 1**; a `MEMORY.md` containing more
  than one `## Session Handoff` heading would only match the first. The handoff skill mandates
  exactly one such block, so this is consistent with the contract rather than defensive against it.

## Acceptance criteria

- [x] Crystallizing a fresh HARNESS-029-compliant `MEMORY.md` leaves `## Session Handoff` as the
      last section, with both stamps above it.
- [x] A second run is idempotent: no duplicated `# currentDate` / `## Last Crystallized:`, block
      still last.
- [x] The regression tests **fail against the pre-fix script** and pass after — proving they test
      the invariant, not the implementation.
- [x] Both twins carry the fix; no bare `Add-Content` / `>>` of a stamp survives in either.
- [x] `web`'s corrupted `MEMORY.md` is repaired in the vault.

## References

- Bitácora board: mlorentedev/dotfiles#850
- Blocked-on-sequencing: #490 (CLI-021, the `dotf vault crystallize` port), #672 (CLI-031, golden
  characterization tests for twin ports)
- ADR-020 §5 (strangler-fig on contact) — satisfied by sequencing through #490
- HARNESS-029 — the last-section invariant, specified in the vault `handoff` SKILL.md §1

<!-- archived 2026-08-08 — PR: https://github.com/mlorentedev/dotfiles/pull/851 -->
