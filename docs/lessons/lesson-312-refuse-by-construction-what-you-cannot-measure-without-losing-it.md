---
id: lesson-312
type: lesson
status: active
created: "2026-09-27"
owner: manu
tags: [lesson, secrets, bitwarden, passkey, curate, idempotence]
---

# 312 — Refuse by construction what you cannot measure without losing it

## What happened

SEC-006 needed a tool to curate about 90 personal items in the Bitwarden vault: file them into folders, hide secret-named text fields, turn reprompt on, and merge duplicates. Every `bw` edit is a whole-item read-modify-write. Nobody had measured whether that round trip keeps `login.fido2Credentials` intact, and 12 items carry a passkey.

Measuring it meant editing a live passkey item and checking the result afterwards. If the round trip dropped the credential, the escrow could not undo that: a passkey is registered at the website it signs in to, and restoring the item's JSON does not re-register it. The measurement risked the very loss it was meant to rule out.

So `dotf secrets curate` does not measure it. It refuses every write to an item that carries a passkey. The check sits in one place: `applyMutation` runs it before looking at the mutation kind, and every curate edit goes through `applyMutation`. As a second, independent check, `--apply` counts the vault's passkeys before and after the run and fails loudly on any difference.

The live plan still needed a merge whose keeper carries a passkey. The duplicate's only extra URI was the `www.` host of the keeper's own URI, and both URIs use the default base-domain match. The fix was a declared `drop=uris` flag on that row, not an exception to the passkey rule. Nothing is carried, so the keeper needs no edit.

The first live dry run also caught an item that had gone to the trash between the inventory and the plan. An edit on an absent item blocks rather than reading as converged, so the two rows for that item surfaced instead of passing as `done`.

## The rule

When the only way to learn whether an operation is safe would itself cause a loss you cannot recover, do not run the experiment. Make the unsafe case unreachable, put that check where every path must pass through it, and add a separate before/after invariant that would catch the loss if the check were ever bypassed.

When a plan needs to reach the refused case, make the plan state what it gives up (`drop=uris`, `drop=notes`). Do not widen the exception. A declared loss can be reviewed; a widened exception is one nobody reads again.

"Absent" means converged only for a delete. For any other operation, an absent target means the plan no longer describes the store.

Refs: SEC-006 (#1784), ADR-028.
