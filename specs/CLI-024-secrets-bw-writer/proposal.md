---
id: "CLI-024-secrets-bw-writer"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-26"
issue: "mlorentedev/dotfiles#612"
tags: [spec, proposal, secrets, cli, go, bitwarden]
template_version: "1.0"
---

# CLI-024-secrets-bw-writer

## Why

The age→bw migration (#612) must be fully CLI-driven — no manual Bitwarden GUI
clicks. The read seam (`BWReader`/`BWGet`) and the registry mutation (#617
`SetBackendBW`) already exist; the missing half of #612 C2 is the **write** seam:
put a value into a Bitwarden item field. It must be **read-modify-write** so a sibling
field is never clobbered — critical once a multi-field item holds several values (the
x-twitter 7-in-1, dockerhub 2-in-1 cases). This is what `dotf secrets set` / `migrate`
(C3/C4) will call to write the secret into Bitwarden.

## What

1. **`BWWriter` interface** — `SetField(item, field, value string) error`, the write
   analog of `BWReader`. Mock-testable; the command layer injects a fake (no real
   Bitwarden in CI), exactly like `BWReader`/`bwReader`.
2. **`setItemField(itemJSON, field, value)` — pure read-modify-write.** Returns the
   item JSON with `field` set to `value`, **preserving every other key of the item**
   (id, type, name, folderId, other fields…): `password`/`username` set the typed
   login (created if the item has none); `notes` sets the note; any other name updates
   the matching custom field or appends a new hidden one. This is the testable core,
   unit-tested with no bw.
3. **`BWPut` — the production `BWWriter`** (shell-out, the analog of `BWGet`): fetch
   the item (`bw get item`), `setItemField`, base64-encode in Go, `bw edit item <id>`.
   Thin I/O, `--nointeraction`, verified by a live smoke with the operator's session —
   not CI (same coverage shape as `BWGet`/`AgeDecrypt`).

Scope: **update an existing item.** Creating an absent item is deferred to `set`
(C3) — distinguishing "item not found" from "vault locked" needs care, and creating on
the wrong signal would spawn duplicate items; `SetField` therefore errors clearly when
the item is absent, and the create path lands with the command that can prompt/confirm.

## Out of scope

- Creating a brand-new item (`set --create` / `EnsureItem`) — #612 C3.
- The `set`/`migrate`/`rotate` commands that consume this seam — #612 C3/C4/C7.
- `bw serve` write path (perf) — behind the same seam later.

## Risks / open questions

- **Read-modify-write preserves the whole item.** `setItemField` works on a generic
  `map[string]any`, not the narrow `bwItem` struct, so unknown keys (id, type, name,
  reprompt, …) survive the round-trip. Tested explicitly (assert an unrelated key is
  unchanged + a sibling field is untouched).
- **No accidental create.** `SetField` only edits; a missing item is a clear error, not
  a silent create — the create-vs-locked ambiguity is handled by the command (C3).
- **No plaintext to disk.** The value lives in memory + is passed to `bw` via stdin
  (base64); never written to a temp file.
- **Live-only verification of `BWPut`.** The shell orchestration is verified by a smoke
  against the operator's unlocked vault (the canary, #612 C8); CI tests `setItemField`.

## Acceptance criteria

- [ ] **AC1** — `setItemField` sets a typed login field (creating `login` if absent),
  a note, an existing custom field, and appends a new custom field — each preserving
  all other item keys and sibling fields. *Verify:* table-driven Go test (assert the
  target changed, an unrelated key + a sibling field unchanged, valid JSON out).
- [ ] **AC2** — `setItemField` errors on malformed JSON; a `BWWriter` mock round-trips
  through the interface. *Verify:* Go test.
- [ ] **AC3** — `BWPut` is wired (compiles, update path: get→setItemField→encode→edit;
  absent item → clear error). *Verify:* build + the structural shape; live smoke
  deferred (documented).
- [ ] **AC4** — `go test ./internal/secrets && go vet && gofmt && go build` clean;
  additive (no existing behaviour changed).

## References

- Issue / backlog: `mlorentedev/dotfiles#612` (Phase C, C2 — write half).
- Mirrors: `cli/internal/secrets/bw.go` (`BWReader`/`BWGet`/`fieldFromItem` — the read
  seam this parallels).
- Composes with: #617 `SetBackendBW` (registry mutation) — `migrate` (C4) calls both.
- ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md`.
