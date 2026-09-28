---
id: "SEC-006-vault-curation"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-27"
issue: "mlorentedev/dotfiles#1784"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, secrets, bitwarden]
template_version: "1.0"
---

# SEC-006-vault-curation

## Why

<!-- from issue #1784: SEC-006: curate the Bitwarden vault and write its governance protocol -->

The Bitwarden vault is the secrets SSOT (ADR-028). Only the 21 registry-managed
items have any structure. The ~160 personal items grew by autocapture:
- they are unfoldered;
- some are duplicates;
- some are credentials of retired projects;
- some copy values whose authority lives elsewhere;
- some keep secret values in plain-text fields;
- none has reprompt.

The curation is decided: a value-free plan of about 60 operations, held in the private knowledge vault because item names are personal.

`reconcile` cannot apply it. It acts only on what the registry declares, and a personal login is out of the registry's bounds by design. Applying the plan by hand in the web vault means about 60 unreviewable clicks, with nothing to stop a duplicate being deleted with a passkey on it. It is the manual path ADR-028 exists to remove.

## What

`dotf secrets curate --plan <file>` reads a declarative plan, one operation per line, keyed by item id. It shows what each operation would do against the store as it is now, and changes nothing.

`--apply --digest <d>` performs the plan. It runs only if the store still has the state that was shown, and it verifies the result.

The operations:

| op | effect | converged when |
|---|---|---|
| `folder <id> <Folder>` | file the item (creating the folder) | the item is in that folder |
| `delete <id> [<kept>]` | delete the item (trash, 30 days) | the item is absent |
| `merge-delete <id> <keeper> [flags]` | carry the URIs the keeper lacks, then delete the duplicate | the duplicate is absent |
| `delete-field <id>:<field>` | remove one custom field | the field is absent |
| `copy-username <id>:<field> <src>` | store `src`'s username as a TEXT field | the field holds that value |
| `add-uri <id> <uri>` | append a login URI | the URI is present |
| `hide <id>:<field>` | turn a text field hidden | the field is hidden |
| `reprompt <id>` | require the master password to view | reprompt is on |

Every operation has one of these states:
- `apply`;
- `done`, meaning already converged, so the second run of a plan is empty;
- `blocked`, with a reason.

A blocked operation makes the whole plan unappliable.

It is blocked when:
- the item carries a passkey. An edit, a delete or a merge never touches such an item; it may only be a keeper that needs no edit.
- a delete target carries attachments.
- a merge's passwords differ, compared in memory.
- a merge's usernames differ, without the `alias` flag.
- a merge's duplicate carries a TOTP.
- a merge's duplicate carries fields or notes the plan did not name, via `drop=fields` or `drop=notes`. Its URIs are carried to the keeper unless the row names `drop=uris`.
- the item is one the registry declares, and the op is not `hide` or `reprompt`. `reconcile` owns where a declared item lives and which fields it has; it does not govern how they are shown.
- an id prefix matches more than one item.
- the row has a `gate=<name>` that `--cleared <name>` did not clear. A gate is a precondition outside the vault, such as a key revoked at its provider.

## Out of scope

- The plan itself, and the item names. They stay in the private knowledge vault; the repo is public.
- Any registry-declared item or field, except `hide` and `reprompt`. `reconcile` and `retired:` own the rest. `ZOHO_APP_PASSWORDS` is retired through the registry, not through a curate plan.
- The policy check (`--check`: folder taxonomy, hidden secret-named fields, reprompt): SEC-007 (#1792). The governance guide's rewrite: the follow-up PR that closes #600 (#1784 AC3).
- Editing items that carry passkeys: refused by construction, see Risks.

## Risks / open questions

- **Passkeys through an edit.** `bw` rewrites the whole item on every edit, and whether that round trip preserves `login.fido2Credentials` has never been measured here. Resolved by refusal, not by measurement. `curate` refuses every write to an item that carries a passkey, and it counts the vault's passkeys before and after `--apply`, failing loudly on a difference. The live plan touches no passkey item except as a keeper that needs no edit.
- **Review-to-apply drift.** A plan reviewed at 19:00 and applied at 19:30 may meet an item edited in between. The dry run prints a digest over each op, its resolved full id and every touched item's `revisionDate`. `--apply` recomputes it and refuses on a mismatch. This is the Terraform saved-plan guarantee, without a plan file.
- **Values.** The comparisons read values into memory: the passwords of a merge, and the username a copy stores. Nothing prints them: output names ids, ops and states only, and a test asserts that no fixture value appears in any output, error paths included.
- **Id prefixes.** The plan uses 8-character id prefixes (the inventory's key). An ambiguous prefix blocks. An absent one reads as converged for `delete`/`merge-delete` and blocks for every other op, so a typo in an edit row cannot pass as done.

## Acceptance criteria

1. **AC1:** the dry run prints one line per op, as `apply`, `done` or `blocked` with a reason. It changes nothing: the fake store records no write.
2. **AC2:** every blocking rule above blocks, each proved by its own test. A plan with any blocked op refuses `--apply` whole, with no write.
3. **AC3:** `--apply` performs each op through the bw write seam. It then re-plans and fails unless every op is `done`. Applying the same plan a second time plans zero writes.
4. **AC4:** `--apply` refuses a missing or stale `--digest`. Changing a touched item's `revisionDate` after the dry run changes the digest.
5. **AC5:** no output contains a value. A test runs dry-run, apply and every error path over a fixture whose secrets are sentinel strings, and asserts that none appears.
6. **AC6:** `--apply` fails if the vault's passkey count after the run differs from before.

## References

- ADR-028 (Bitwarden SSOT), CLI-080, CLI-082 and CLI-083 (reconcile, the registry-side twin whose shape this mirrors).
- Plan and evidence: knowledge vault `10_projects/dotfiles/research/sec-006-vault-curation/` (private).
