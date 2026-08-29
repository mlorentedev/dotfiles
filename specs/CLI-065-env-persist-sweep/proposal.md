---
id: "CLI-065-env-persist-sweep"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-29"
issue: "mlorentedev/dotfiles#1363"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-065-env-persist-sweep

> **Naming**: file lives at `<repo>/specs/CLI-065-env-persist-sweep/proposal.md`. `CLI-065-env-persist-sweep` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

`dotf env persist` (CLI-058, #1324) writes every variable the contract names into
`HKCU\Environment` and never removes one. When a variable is retired from
`env-contract.json` — renamed, dropped, or moved out of the persisted scope — its
User-scope value stays behind on every box that ever ran setup, and the
profile-less processes the scope exists for (Copilot's `pwsh -NoProfile` tool
calls, Scheduled Tasks, Explorer) keep inheriting a value nothing declares any
more. Same class as WIN-013 (#1310): a deployer that adds and never sweeps.
CodeRabbit named it on #1362 and it was deliberately not built there.

## What

- `dotf env persist` records the names it wrote in an **ownership marker** in the
  same store — the value `DOTF_MANAGED_ENV`, the sorted names joined by `;` — and
  on every run deletes each name the marker lists that the contract no longer
  resolves, then rewrites the marker. The sweep is bounded to the marker: a
  variable dotf never wrote is never touched, whatever its name.
- `dotf env persist --check` reports such leftovers as drift (`retired: NAME`)
  and exits non-zero while one remains.
- `dotf doctor`'s *Persisted environment (user scope)* row WARNs on a leftover
  with the remedy `dotf env persist`.
- Off Windows nothing changes: no per-user persistent scope, no-op.

## Out of scope

- Values dotf wrote **before the marker existed**. A box whose last persist
  predates this change has no marker, so names retired before it are not swept:
  there is no record of what dotf wrote, and guessing from the registry is the
  unbounded sweep the ticket forbids. WIN-013 chose an allow-list of seven known
  names for its pre-contract leftovers; here the set is "whatever a past
  contract named", which nothing records.
- `PATH` / `Path` and every non-contract variable: never read, never written,
  never deleted.
- Machine (HKLM) scope, and the process-scope environment of the running shell —
  a deleted value is gone for new processes; the current one keeps its copy.
- The rc-file layer (`paths.ps1` / `paths.sh`): `dotf env generate` rewrites
  those whole, so they carry nothing stale.

## Risks / open questions

- A hand-edited or truncated marker. RESOLVED: the sweep deletes only a name
  that is **both** in the marker **and** absent from the contract; a marker
  naming things that do not exist in the store deletes nothing, and the marker
  is rewritten from the contract on every run, so one bad value cannot persist.
- The marker is a visible store value (Environment Variables dialog). RESOLVED:
  named `DOTF_MANAGED_ENV` so its owner is obvious, and named in `--help`.
- The user re-purposed a name dotf wrote (same name, their value) and the
  contract then retires it: the marker says dotf owns the name, so the sweep
  removes it. RESOLVED, stated rather than guarded: the contract owns the names
  it declares, and a retired name is retired everywhere; `--check` prints
  `retired: NAME` before any write for whoever wants to look first.
- Interface growth: `UserEnvStore` gains `Delete`, which the doctor's read-only
  adapter cannot honestly implement. RESOLVED: split the reader out
  (`UserEnvReader` = `Get`; `UserEnvStore` = reader + `Set` + `Delete`), and
  `Drift` takes the reader — which also deletes the adapter's `Set` no-op, a
  method that existed only to satisfy an interface wider than its caller.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] AC1 — `dotf env persist` writes an ownership marker naming every contract
      variable it persisted; a second run changes nothing, marker included.
- [x] AC2 — retiring a name from the contract and running `dotf env persist`
      again deletes that name from the user scope and **only** it; a variable
      dotf never wrote is never touched. (Fake store: two runs, one name retired
      between them, one foreign name present throughout.)
- [x] AC3 — `dotf env persist --check` names each leftover (`retired: NAME`)
      and exits non-zero while one is persisted; zero once swept.
- [x] AC4 — `dotf doctor` *Persisted environment (user scope)* WARNs on a
      leftover with the remedy `dotf env persist`; PASS after the sweep.
- [x] AC5 — with no marker in the store (a box that persisted before this
      change) nothing is deleted and the marker is written; off Windows the
      command stays a no-op.
- [x] AC6 — on the Windows work box: with a scratch contract that retires one
      name, `dotf env persist` removes it from `HKCU\Environment` (`reg query`
      shows it gone) and the next run reports nothing changed.

## References

- Bitácora board: #1363. The class: WIN-013 (#1310, `specs/archive/WIN-013-scripts-dir-contract/`).
- Prior spec: `specs/archive/CLI-058-env-persist/` (the command this extends).
- Related ADR: `docs/adr/adr-025-cross-machine-paths.md` (the contract and its cascade).
- The rule this applies: *the writer touches only what it owns* — the shared-surface
  pattern the project memory records across six surfaces.
