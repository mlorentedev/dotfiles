---
id: "CLI-080-secrets-reconcile"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-22"
issue: "mlorentedev/dotfiles#1596"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-080-secrets-reconcile

## Why

`dotf secrets drift` (CLI-078) made the registry's declared Bitwarden layout
comparable to the live store, and the first comparison found the store did not
match: an item outside its declared folder, and two declared items — the
per-purpose GitHub tokens #321 and #586 call for — that had never been created.
Meanwhile both GitHub entries still read an age blob GitHub answers **401** to,
while the working tokens sit as two of nine fields inside one unfoldered `GitHub`
item. Every one of those gaps can today be closed only by hand in the Bitwarden app,
which is the ad-hoc change to a remote system the standing orders forbid, and which
leaves no reviewable record of what moved where.

## What

`dotf secrets reconcile` converges the store's **layout** toward the registry:

- **Plan by default.** It prints the operations it would perform and changes
  nothing. `--apply` performs them. A second run after a successful apply plans
  zero operations.
- **Four operations**, each derived from a drift finding:
  `create-folder`, `move-item` (into its declared folder), `create-item` and
  `add-field` (the last two only from a declared source — below).
- **A declared source, `bw.from: { item, field }`.** Reconcile never invents a
  value. Where a declared item or field is absent, it can be created only by
  copying the value from the source a registry entry names — the Terraform `moved`
  idea applied to credentials: the refactor is declared in code, reviewed in a PR,
  and applied by the tool. The value is read and written inside the process and
  never printed.
- **Never overwrites a value.** A destination field that already exists is left
  alone whatever it holds; changing a value is `set`/`rotate`'s job. Once the
  destination exists, a `from:` is satisfied and reconcile reports it as removable.
- **Fails closed.** A finding it cannot plan (an absent item with no `from:`, a
  `from:` whose source is absent or empty) is reported with its remediation and
  makes the plan non-zero; `--apply` refuses to run a plan that contains one.

First use, in this PR: `GITHUB_PERSONAL_ACCESS_TOKEN` and `RELEASE_TOKEN` declare
`from:` the two fields of the `GitHub` item and flip to `backend: bw`, which is the
#321 split for those two tokens and retires their dependence on the revoked blob.
Their items are created with the field named after the env var
(`GITHUB_PERSONAL_ACCESS_TOKEN`, `RELEASE_TOKEN`), per the 2026-09-21 naming
decision: the rename slice exists for items that already carry another name, and
these do not exist yet, so minting them under a name already rejected would only
feed that slice.

`RELEASE_TOKEN`'s consumer is CI (`ci:mlorentedev/dotfiles`), so resolving locally
does not fix it. After the apply, `dotf secrets sync` for that scope pushes the
live value to the repository secret, and the release workflow is what proves it.

## Out of scope

- **Renaming fields to the env-var name** (the naming decision of 2026-09-21).
  The same `add-field` operation carries it, but adopting it across ~20 items needs
  its own transition design: if the registry renames a field before `--apply`
  runs, reads break between the merge and the apply. Separate slice.
- **Deleting the source.** The copied fields stay in the `GitHub` item. Copying is
  additive and reversible; deleting a credential is not, and belongs after the
  consumers are verified by consequence. The other seven credentials in that item
  are also untouched.
- **Rotation age.** See the decision below; this slice writes no timestamp.
- **Age-backed absent items** (`zoho`). Their source is the age store and their
  tool is `migrate`; reconcile names that remediation rather than duplicating it.
- **Unmanaged items.** The ~164 items no declaration names are never touched.
- **Retiring `sensitive/github.token.secret.age`.** Unreferenced after the flip;
  retiring age files is its own step.

## Risks / open questions

- **Resolved — rotation age and `revisionDate`.** Bitwarden bumps `revisionDate`
  on every edit, so every `move-item` resets the only age signal the store has.
  `now − revisionDate` is a **lower** bound on a credential's age (the value
  existed at or before the last edit): a warning built on it never fires falsely
  but goes silent after any edit, reconcile's included. Decision: rotation age
  comes from an explicit `rotated-at` field written by `set`/`rotate` in the
  rotation slice, with `revisionDate` as the fallback lower bound. Reconcile writes
  no `rotated-at`, because it does not know when a copied value was minted.
- **Resolved — stale cache.** The daemon answers from its cache, and a plan
  computed on a stale inventory would create a duplicate of an item made
  elsewhere since the last sync. Reconcile syncs before it reads.
- **Resolved — apply before merge.** The registry is read from the invoking
  worktree (BUG-072), so the reviewed branch can be applied before it merges, the
  way Atlantis applies. Safe because every operation is additive: nothing existing
  is overwritten, moved out of a declared place, or deleted.
- **Accepted — concurrent writers.** `ResolveFolder` is not safe against a racing
  process (OPS-028); reconcile inherits that. `dotf secrets` has no
  concurrent-writer story anywhere.
- **Depends on — CLI-078 defects found by the first live run (#1600).** Unfoldered
  items read as filed in `No Folder`, file-exposed secrets were never declared, and
  `folder: ""` meant "must be unfoldered". Reconcile plans from that inventory, so
  they are fixed first. The third matters most here: `""` now means placement is not
  governed, so reconcile never plans a `move-item` for an item declared with no
  folder, and never unfiles what the operator filed by hand.
- **Resolved — an older binary reading the new registry.** `ParseRegistry` is not
  strict about unknown keys, so a stale `dotf` ignores `from:` rather than failing.
- **Resolved — the test pinning the GitHub entries on age.**
  `registry_write_test.go` fixes `GITHUB_PERSONAL_ACCESS_TOKEN` on age "until C9"
  (`migrate --split`). This PR is that split, done by a different mechanism, so the
  fixture changes with its reason stated.

## Acceptance criteria

- [ ] AC1 — with no flag, reconcile writes nothing (a fake store records zero
      mutating requests) and prints one line per planned operation.
- [ ] AC2 — `--apply` performs `create-folder`, `move-item`, `create-item` and
      `add-field`, in that order, and a second run plans zero operations.
- [ ] AC3 — `create-item`/`add-field` copy the value from the declared `from:`
      source; the value appears in no output, proven by planting a distinctive
      value and asserting its absence.
- [ ] AC4 — an existing destination field is never overwritten, and a satisfied
      `from:` is reported as removable.
- [ ] AC5 — an absent item or field without a `from:` on a live declaration, and a
      `from:` whose source is absent or ambiguous, are reported with a remediation,
      exit non-zero, and make `--apply` refuse the whole plan. A dormant declaration
      with no source is deferred to `migrate` and blocks nothing. An EMPTY source —
      which the plan cannot see, since it reads no value — is refused by `--apply`
      before its first write.
- [ ] AC6 — the store is synced before the inventory is read.
- [ ] AC7 — the registry rejects a malformed `from:` (empty item or field, source
      equal to destination, or declared on a multi-var secret).
- [ ] AC8 — `BWPut` and `BWServeWriter` produce byte-identical item JSON for a
      folder move, sharing one pure core as they already do for `setItemField`.
- [ ] AC9 — applied to the live store: `GITHUB_PERSONAL_ACCESS_TOKEN` and
      `RELEASE_TOKEN` resolve from their new items and the GitHub API answers 200
      to each; a re-run plans zero operations; `dotf secrets sync` carries
      `RELEASE_TOKEN` to its CI consumer.

## References

- Bitácora board: `mlorentedev/dotfiles#1596` (mechanism); `#586` (outcome: folder
  taxonomy, per-purpose split); `#321` (the split's rationale)
- CLI-078 — `dotf secrets drift`, the inventory this plans from
- ADR-028 §2 (registry as mapping SSOT), §6 (curation & naming)
- `cli/internal/secrets/bwserve_writer.go` — the write seams reused
