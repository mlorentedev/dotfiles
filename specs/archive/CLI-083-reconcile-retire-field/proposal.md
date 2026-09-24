---
id: "CLI-083-reconcile-retire-field"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-09-24"
issue: "mlorentedev/dotfiles#1624"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-083-reconcile-retire-field

## Why

The 2026-09-24 dedupe (#1648) removed every legacy copy `reconcile` could verify
equal. Four credentials are still held in two places, and the tool cannot remove
any of them (measured on the live store, value-free):

1. **A dead legacy value.** `Stripe`/"backup-codes" holds a backup code superseded
   by a regeneration. `login.tailscale.com`/"auth-key" holds auth keys past
   Tailscale's 90-day maximum. `Hetzner`/"key" answers 401. `retire: true` removes
   a source only when it equals the canonical copy, which is the right gate for a
   copy, but a dead value equals nothing current. The items themselves must stay,
   because they are the account logins. Today the only way to remove the field is
   a hand edit in the web vault, which the standing orders rule out (#1624,
   comment of 2026-09-24).
2. **A copy of a multi-var secret.** `cloud.nan.builders`/"api-key" is a copy of
   `NAN_API_KEY`, and the registry refuses `bw.from` on it: "bw.from is not
   supported on a multi-var secret (one source cannot fill 2 fields)". Both of
   NAN_API_KEY's variables read the same field, `nan-api-key`/"api-key", so one
   source does fill it. The refusal is stricter than the case needs.

Two defects from the same issue are also still open:

3. **A whitespace-only `retired:` item name is accepted** (CLI-082 review, Minor).
   `checkRetired` tests `r.Item == ""`, so `item: "  "` parses and plans as gone.
4. **`dotf secrets sync ci NAME NAME` uploads the same secret twice** (#1624 item
   2). `scopeUpload` appends once per name given.

## What

- **A `retired:` entry may name a field.** `{item, field, reason}` retires one
  field of an item that stays. `reconcile` plans a `delete-field` for it when the
  vault holds the field. The plan line shows the reason and what the item will
  still hold, so an operator can see the item is not emptied. The operation runs
  last, at the same rank as `delete-item`. A field that is already gone, or an
  item that is gone, is reported satisfied, so the entry can be removed. A name
  that matches several items blocks, as `delete-item` does.
- **The registry validates a field entry.** It refuses:
  - a blank field;
  - `username` and `password`, which are the account's login and not a legacy copy;
  - a field that a declaration reads (any secret's `bw.item` plus a field it
    resolves, or a `bw.from` item plus field);
  - the same item and field listed twice;
  - a field entry for an item that is also retired whole.

  A field on an item that a declaration names is allowed, as long as that
  declaration does not read that field.
- **`bw.from` on a multi-var secret whose variables all resolve to one field.**
  The registry refuses `bw.from` only when the variables resolve to different
  fields. The planner already plans once per secret (`seen[d.Secret]`), so two
  variables over one field plan one operation.
- **Blank names.** `checkRetired` refuses an item name that is empty after
  trimming.
- **`sync ci` names are deduplicated.** A name given twice is uploaded once, in
  first-given order.

## Out of scope

- **Disambiguating the two items named `Hetzner`.** Every operation that acts by
  name blocks on an ambiguous name, by design. So `Hetzner`/"key" stays blocked
  until one item is renamed. A `rename-item` operation belongs to the folder
  taxonomy work (#586, #1596), which needs an ADR-028 §6 decision first.
- **Enforcing escrow freshness in `reconcile`.** The runbook requires a DR escrow
  before any apply. Checking it mechanically (escrow `max_revision` against the
  store's newest revision) is a separate guard.
- **Per-var `ci:` consumers** (#1603, CLI-081).
- **The field retires themselves, applied to the live store.** Each is a data
  change that needs the owner's authorization, made with this tool once it merges.

## Risks / open questions

- **Accepted: `delete-field` has no equality gate.** That is its purpose: the value
  is dead, not equal. What replaces the gate:
  - the entry is declared in a reviewed file, with a mandatory reason;
  - the plan shows the field, the reason, and what the item keeps, before
    anything runs;
  - login credentials are refused;
  - a field a declaration reads is refused.

  Unlike `delete-item`, a removed field does **not** go to Bitwarden's trash (an
  edit is not a deletion), so the only recovery path is the DR escrow. The
  runbook step says so and requires the escrow before the apply.
- **Resolved: a copy of a multi-var secret is compared once.** One secret, one
  field, one verdict, because the planner keys on the secret. A test pins that a
  two-variable secret over one field plans exactly one `retire-source`.
- **Resolved: which name a field retire acts on.** The same lookup as
  `delete-item` and `retire-source`: exactly one item carries the name, or the
  plan blocks.

## Acceptance criteria

- [ ] AC1 — the registry accepts a `retired:` entry with a `field:` and refuses
      each invalid form: a blank item or field (whitespace included), `username`
      or `password`, a field a declaration reads as destination or `bw.from`
      source, a duplicate item and field, and a field entry for an item that is
      also retired whole.
- [ ] AC2 — `reconcile` plans `delete-field` for a retired field the vault holds,
      showing its reason and what the item keeps, at the last rank. It reports a
      field or item already gone as satisfied, and blocks a name that matches
      several items. `--apply` removes the field through the writer seam, and a
      further plan is empty.
- [ ] AC3 — `bw.from` is accepted on a multi-var secret whose variables all resolve
      to one field, and it plans a single operation. It is still refused when they
      resolve to different fields.
- [ ] AC4 — `dotf secrets sync ci` uploads a name given twice only once.
- [ ] AC5 — no secret value reaches the plan or apply output on the new paths,
      proven by planted values.

## References

- Bitácora board: `mlorentedev/dotfiles#1624` (item 2, and the comments of
  2026-09-24: delete-field, multi-var `bw.from`, the whitespace name)
- `specs/CLI-082-reconcile-dedupe/`: `retired:` and `delete-item`, which this extends
- #1648: the dedupe that measured what remains
- `docs/runbooks/guide-secrets-governance.md`: CONVERGE step 8
