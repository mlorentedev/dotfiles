---
id: "BUG-084-secrets-write-bw-serve"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-08-15"
issue: "mlorentedev/dotfiles#993"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# BUG-084-secrets-write-bw-serve

> **Naming**: file lives at `<repo>/specs/BUG-084-secrets-write-bw-serve/proposal.md`. `BUG-084-secrets-write-bw-serve` is `AREA-NNN-slug`.

## Why

<!-- from issue #993: BUG-084: dotf secrets set is unusable under the sanctioned unlock model — the write path never moved to bw serve -->

`dotf secrets set` cannot write anything on a machine unlocked the way ADR-028 and
CLI-024 prescribe. CLI-024-secrets-bw-serve moved the **read** path to the `bw serve`
daemon and deliberately deferred the write path as "a natural fast-follow"; that
fast-follow never happened, so `BWPut` still shells out to a `bw` binary that
authenticates from `BW_SESSION` — the ambient variable ADR-028 exists to abolish. Every
write therefore fails with `Vault is locked.` no matter how recently the operator
unlocked, because the daemon holds the unlocked session and the shellout asks a
different subject. This is not a corner case: it is the entire provisioning and rotation
story, and it blocks the Phase 3 migration outright.

## What

The write path gains the same daemon seam the read path already has.

- A `BWServeWriter` implementing `SetField`, `CreateItem` and `ResolveFolder` against
  the daemon's REST API — `PUT /object/item/:id`, `POST /object/item`,
  `GET /list/object/folders` + `POST /object/folder`.
- A `BWFallbackWriter` selecting daemon-vs-shellout, mirroring `BWFallbackReader`.
- `dotf secrets set <ID>` (both the edit-existing and create-new paths), and by
  extension `rotate` and `migrate`, succeed against an unlocked daemon with no
  `BW_SESSION` anywhere in the environment.
- Where a write genuinely cannot proceed, the error names the remediation that works
  (`dotf secrets unlock`), never a bare `Vault is locked.` that sends the operator to
  `bw unlock` — which prints a session key to a shell nobody is reading and does not
  fix the failure.
- `dotf secrets backup` keeps its shellout, and gains an error that says *why* the
  daemon cannot serve it and names the invocation that does work. See Out of scope.

## Out of scope

- **Serve-backed `Export` / `dotf secrets backup`.** Not deferred by preference —
  impossible. Enumerated from the shipped `@bitwarden/cli` 2026.5.0 bundle, `bw serve`
  exposes **no export route**; the sole `"/export"` string in it is
  `apiService.send("GET", "/organizations/" + organizationId + "/export")`, an
  *organization* export against the cloud API. `BWExport` therefore stays a `bw`
  shellout permanently, and `BW_SESSION="$(bw unlock --raw)" dotf secrets backup` is its
  standing invocation, not a workaround with an expiry date. Only its error message
  changes here.
- **Assembling the escrow from `GET /list/object/items` + `/list/object/folders`.** It
  would *probably* yield an importable `{encrypted:false,folders,items}` document, and
  *probably* is not a property a disaster-recovery escrow may have. Proving it requires
  a real `bw import` round-trip into a scratch vault. Shipping it unverified produces an
  escrow that looks like it works and fails only during an actual recovery — the exact
  class of #997, #992 and #906. Filed separately with that round-trip as its acceptance
  criterion.
- **`DELETE /object/item/:id`.** The daemon exposes it; nothing in `dotf secrets` needs
  it, and a delete capability on the write seam is a footgun with no current caller
  (#938's retire path is a separate, deliberate design).
- **#1004 (verify aborts on registry validation)** — same package, adjacent symptom,
  different surface (registry loader, not the bw backend). Parked behind this spec by
  the same session, not bundled.
- **Windows daemon-lifecycle validation.** Cross-platform by construction, live-verified
  on Linux only, per the precedent set by CLI-024-secrets-bw-serve.

## Risks / open questions

- **Read/write backend split-brain.** The failure this bug *is*: two halves of one
  command authenticating through different subjects. `rotate` (#996/#1003) reads and
  writes in a single invocation, making it the sharpest detector — if reads resolve
  through the daemon and writes through a shellout, rotation half-succeeds. AC5 exists
  to make that unrepresentable rather than merely tested.
- **Read-modify-write over a stale daemon cache.** `SetField` is RMW: read item, mutate
  one field, write it back. If the daemon's cache is stale relative to the server, a
  sibling field edited elsewhere is silently clobbered. `BWPut` inherits the same risk
  today via the CLI's own cache, so this is not a regression — but the daemon makes it
  easier to hold an unlocked session open for days, which widens the window. Mitigation:
  `POST /sync` before the read half of an RMW (`BWServeClient.Sync()` already exists).
- **`bw serve`'s REST API is unauthenticated by design.** Already accepted and mitigated
  by the loopback bind invariant (`bwServeHostname`, AC5 of CLI-024-secrets-bw-serve).
  This spec adds *write* capability to that surface, which raises the consequence of a
  bind regression from vault disclosure to vault mutation. No new mitigation proposed;
  the existing bind test is now load-bearing for integrity, not just confidentiality,
  and this spec records that.
- **Live verification requires a real unlocked vault**, which CI does not have. Thin I/O
  is live-smoke-verified only, per the standing convention for `BWGet`/`BWPut`. Any real
  write must target the designated canary item and print fingerprints, never values
  (`docs/lessons.md`, "Redact at the producer").

## Acceptance criteria

- [ ] AC1 — `dotf secrets set <ID>` succeeds against an unlocked `bw serve` daemon with
      no `BW_SESSION` anywhere in the environment, for an **existing** item.
- [ ] AC2 — the same holds for a **new** item, exercising `CreateItem` and
      `ResolveFolder` (an entry declaring a `bw.folder` lands in that folder).
- [ ] AC3 — with no daemon and no session, the error names `dotf secrets unlock`, not a
      bare `Vault is locked.`
- [ ] AC4 — `dotf secrets backup` with no session fails with an error that names
      `BW_SESSION="$(bw unlock --raw)" dotf secrets backup` and states that the daemon
      has no export endpoint.
- [ ] AC5 — a test asserts the read and write fallbacks agree on backend selection for
      the same daemon state, so the next path cannot move alone (the guard this bug's
      own existence proves is needed).
- [ ] AC6 — the failure is observed red against real state before the fix, and green
      after, with both captured in `verification.md`.

## References

- Bitácora board: `mlorentedev/dotfiles#993` (see `issue:` frontmatter)
- `specs/CLI-024-secrets-bw-serve/proposal.md` — the read-path move; names this work as
  its fast-follow in Out of scope
- `specs/archive/CLI-024-secrets-bw-writer/` — the original write seam being extended
- `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md` — why the ambient environment is
  empty by design
- #997 (escrow severity), #992 (parse guard), #1004 (verify scoping) — adjacent, other
  sessions or parked
- `docs/lessons.md`, "Redact at the producer" — binding on any live write verification
