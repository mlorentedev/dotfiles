---
id: "CLI-082-reconcile-dedupe"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-09-23"
issue: "mlorentedev/dotfiles#1624"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-082-reconcile-dedupe

## Why

The Bitwarden store still holds a second copy of about a dozen managed
credentials. They sit in the legacy items they were first saved in, beside the
canonical items the registry declares (measured 2026-09-23, value-free):
`pypi.org`/"API token" next to `pypi-token`, `login.tailscale.com`/"auth-key"
next to `tailscale-auth-key`, and `cloud.nan.builders`/"api-key" next to
`nan-api-key`. Some legacy items hold nothing but the copy, like the notes-only
`POLLEX_API_KEY` and `OPEN ROUTER API KEY`. Two copies of a credential are two
places to rotate and two places to leak, and the one nobody rotates is the one
that goes stale. The bitácora token showed it: its two copies held **different**
live tokens, and the repositories held four between them.

`reconcile` can retire a source field (CLI-080), but two gaps stop it deduplicating:

1. **Its plan cannot say what the apply will do.** Whether a retire can proceed
   depends on the source and destination values being equal, and only `--apply`
   compares them. With a dozen pairs, an operator finds the mismatches one refused
   run at a time. When one is refused, the message says "settle it" and does not
   say how (#1624, item 1).
2. **It cannot remove an item.** Retiring the only field of a notes-only item
   leaves an empty item behind (#1624, item 3). An item whose registry entry was
   retired stays in the vault holding a revoked credential; `github-cli-pat` is
   one, since #1632. Today the only way to remove either is a hand edit in the
   web vault, which the standing orders rule out.

## What

- **The plan gives a verdict for every retire.** `dotf secrets reconcile`, with or
  without `--apply`, reads each planned retire's source and destination values
  into memory and compares them. The plan line ends with one verdict: `equal`,
  `differs`, `destination empty`, or `unreadable`. No value, length or
  fingerprint is printed. A retire that is not `equal` becomes a **blocker**, so
  the plan is unappliable as a whole, and the blocker names both ways to settle
  it. `--apply` still verifies again before any write, because the store can
  change between a plan and an apply.
- **Items are retired by declaration.** The registry gains a top-level `retired:`
  list; each entry names an item and a reason. `reconcile` plans a `delete-item`
  for each listed item the vault holds, and shows the item's shape (field names,
  notes, login) so the operator reads what goes before it goes. It runs after
  every other operation. The registry refuses the entry while a declaration
  still names the item (`bw.item` or `bw.from.item`), and the plan blocks it when
  several items carry the name. A listed
  item that is already gone is reported satisfied, so its entry can be removed.

The dedupe of a pair is then two declared steps, each shown by the plan before it
runs:

1. `bw.from` with `retire: true` removes the legacy field, and only if it equals
   the canonical copy.
2. If that left the legacy item empty, the `bw.from` record is dropped and the
   item is listed under `retired:`.

## Out of scope

- **Deleting an emptied item automatically.** It is a deletion, and this
  codebase declares deletions rather than implying them: `retire` exists for
  that reason. The plan's shape line makes an emptied item visible; the operator
  declares it.
- **The folder taxonomy, renames, and `field` = env var name.** #586 and #1596
  disagree on the target (`Dotfiles/floor` against `identity/` + `finance/`).
  That needs an ADR-028 §6 amendment first.
- **Splitting the rest of the `GitHub` item** (runner tokens, the dispatch token,
  the evalkit key). Their consumers are kubelab's.
- **Rotation age and stale CI consumers** (#1596).
- **#1624 item 2** (`sync ci` uploading a name given twice). It is unrelated to
  dedupe and stays on #1624.
- **The dedupe itself, applied to the live store.** It is a data change that
  needs the owner's authorization per write, made with this tool once it is
  reviewed and merged. It is tracked in `tasks.md`, not as a criterion here.

## Risks / open questions

- **Resolved: the plan now reads values.** Until now only `--apply` read a secret
  value; the plan read shapes. A verdict cannot be computed from shapes. The plan
  reads through the same pinned reader, in the same process, as the apply that
  would follow it, and it prints a verdict and nothing derived from the value.
  The exposure is the one `--apply` already has, moved earlier. Deliberate, and
  stated in the command's help.
- **Resolved: which state is "current".** The tool cannot know, and does not
  guess. A value that `differs` blocks, and the blocker names the exits:
  `dotf secrets rotate <id>` when the canonical copy is the stale one, or drop
  `retire:` to keep both while it is settled by hand.
- **Resolved: deleting an item that holds more than the verified copy.** A
  login item such as `pypi.org` is an account login; it must keep its login and
  lose only the API field. So `delete-item` is never inferred. It is declared,
  refused while any declaration names the item, and shown with its shape.
  Bitwarden moves a deleted item to its trash, which is recoverable for 30 days,
  and the runbook takes a DR escrow before any apply.
- **Accepted: time of check to time of use.** Values can change between the plan
  and the apply. The apply re-verifies every retire before its first write, as it
  does today, so a stale plan fails closed.

## Acceptance criteria

- [ ] AC1 — the plan compares each planned retire's source and destination
      values in memory and prints one verdict per retire (`equal`, `differs`,
      `destination empty`, `unreadable`), and never a value.
- [ ] AC2 — a retire whose verdict is not `equal` is blocked, making the plan
      unappliable. For `differs` and `destination empty` the blocker names both
      exits (`dotf secrets rotate <id>`, or drop `retire:`); for `unreadable` it
      names the side that could not be read. `--apply` re-verifies before any
      write, and its refusal names the same exits.
- [ ] AC3 — the registry accepts a top-level `retired:` list whose entries each
      name an item and a non-empty reason. It refuses an entry missing either, an
      item listed twice, and an item that a declaration still names as `bw.item`
      or `bw.from.item`. That last one is static, so CI catches it rather than a
      plan.
- [ ] AC4 — `reconcile` plans `delete-item` for each retired item the vault holds,
      showing its shape, after every other operation; a retired item already
      absent is reported satisfied; `--apply` deletes it and a further plan is empty.
- [ ] AC5 — `delete-item` is blocked when the name matches several items, since
      deleting an arbitrary one could delete the wrong credential.
- [ ] AC6 — no secret value reaches the plan or apply output on any new path,
      proven by planted values.

## References

- Bitácora board: `mlorentedev/dotfiles#1624` (items 1 and 3)
- `specs/archive/CLI-080-secrets-reconcile/`: the reconcile this extends
- `specs/archive/CLI-078-secrets-layout-drift/`: the value-free projection the plan
  shape line reuses
- #586, #1596: the curation this makes reachable
- `docs/runbooks/guide-secrets-governance.md`: CONVERGE
