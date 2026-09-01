---
id: "CLI-055-owner-only-acl"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-29"
issue: "mlorentedev/dotfiles#1302"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-055-owner-only-acl

> **Naming**: file lives at `<repo>/specs/CLI-055-owner-only-acl/proposal.md`. `CLI-055-owner-only-acl` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

`ai/deploy.json` declares `"mode": "0600"` for pi's `models.json` (*declared, never
inferred*, #987), and `secrets/registry.yaml` exposes three credential files at
`0600`. All of them reach disk through `os.Chmod` — `deploy.stage`, `deploy.commit`,
`secrets.AtomicWriteMode` — and on Windows `os.Chmod` can only toggle the
read-only attribute. A rendered credential file under `%USERPROFILE%` is then
protected by whatever ACL its directory hands down, which on a shared or
domain-joined box is not "owner only". The manifest promises a mode the OS
silently does not deliver.

## What

- One package, `cli/internal/fsmode`, owns "apply this mode to this file":
  `fsmode.Apply(path, mode)`. On POSIX it is `os.Chmod`. On Windows it is
  `os.Chmod` (the read-only bit, as before) **plus**, when the mode grants
  nothing to group or other (`mode & 0o077 == 0`), an owner-only DACL: the
  calling user's SID with full access, `LocalSystem` with full access (backup,
  Defender, the services that already read every profile), inheritance from
  the directory cut (`PROTECTED_DACL`). A mode with group/other bits leaves the
  inherited ACL alone — there is nothing owner-only to express.
- The three writers call it instead of `os.Chmod`; nothing else changes shape.
- A Windows-only test proves the consequence with the tool an administrator
  would use: `icacls` on a file written at `0600` lists exactly the user and
  SYSTEM and no inherited entries, and `GetNamedSecurityInfo` reports
  `SE_DACL_PROTECTED`; a `0644` file keeps its inherited entries.

## Out of scope

- Directories: the writers create parent directories at `0755`; a secret's
  directory is the user's own and stays inherited.
- Ownership transfer, auditing (SACL), or Administrators: an administrator can
  take ownership regardless; expressing that in the DACL would be theatre.
- Reading the ACL back in `dotf doctor`: verified here by test and on the box;
  a doctor row is a follow-up if drift is ever observed (the ticket's option 2
  was the floor, option 1 is built).

## Risks / open questions

- `SetNamedSecurityInfo` on a path the user does not own (a symlink into a
  system location). RESOLVED: the writers only ever write under the user's
  home; an error is returned, never swallowed, naming the path.
- Removing inherited ACEs strips Administrators' entry. RESOLVED, deliberate:
  that is what 0600 means, and Administrators keep `SeTakeOwnershipPrivilege`.
- CI's Windows leg runs as a service account: the test must resolve the
  current user's SID from the token, never from a username, so it holds for
  any account. RESOLVED by construction (`OpenCurrentProcessToken` +
  `GetTokenUser`).

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] AC1 — `fsmode.Apply(path, 0o600)` on Windows leaves the file with exactly two ACEs — the calling user and SYSTEM — and a protected DACL; on POSIX it is `os.Chmod`.
- [x] AC2 — `fsmode.Apply(path, 0o644)` on Windows keeps the inherited ACL (no DACL rewrite) and sets the read-only attribute as `os.Chmod` did.
- [x] AC3 — `dotf deploy` of a `mode: 0600` entry and `secrets.AtomicWriteMode(…, 0o600)` both go through `fsmode.Apply`; no `os.Chmod` call remains in the writers.
- [x] AC4 — on the Windows work box: after `dotf deploy`, `icacls ~/.pi/agent/models.json` lists the user and `NT AUTHORITY\SYSTEM` only, with no `(I)` entries; a `0644` neighbour keeps its `(I)` entries.
- [x] AC5 — CI's Windows leg runs the Windows-only test and passes under its service account.

## References

- Bitácora board: #1302. Related: #987 (declared modes), CLI-039.
- `cli/internal/deploy/deploy.go` (`stage`, `commit`), `cli/internal/secrets/render.go` (`AtomicWriteMode`), `golang.org/x/sys/windows` (already a direct dependency).
