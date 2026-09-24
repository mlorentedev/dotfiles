---
id: "CLI-078-secrets-layout-drift"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-09-21"
issue: "mlorentedev/dotfiles#1596"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-078-secrets-layout-drift

## Why

`secrets/registry.yaml` has been the mapping SSOT since ADR-028, declaring an
item, a field and a folder for every secret — and **nothing has ever compared
that declaration to the vault**. Measured 2026-09-21 against the live store:
every declared folder name was absent under that spelling (the declarations said
`apps`, the vault holds `Dotfiles/apps`), three declared items did not exist at
all, and one item sat outside the folder it was declared into. None of it was
reported anywhere, because a declaration nobody checks is not a declaration; it
is a comment that happens to be YAML.

Two of those cost something real. `BWPut.ResolveFolder` matches a folder by exact
name and **creates** one when it finds none, so the next `dotf secrets set`
provisioning an app-plane secret would have minted a second folder called `apps`
beside `Dotfiles/apps` and split the managed items across both. And
`GITHUB_PERSONAL_ACCESS_TOKEN` declared `github-cli-pat`, an item that had never
been created — invisible while the entry still read from age, whose blob held a
token GitHub now answers **401** to. `dotf secrets verify` reported OK throughout,
because resolving and working are different claims.

## What

`dotf secrets drift` compares the registry's declared layout against the vault's
actual shape and reports each disagreement, ordered by the sequence they must be
fixed in — folder, then item, then field. It exits non-zero on any finding, so a
hook or CI can gate on it, and it syncs the store before reading it, so what a
gate reads is the server's state rather than the daemon's cache.

It reads **shapes only**: item names, folder names, field names, and three
presence booleans. No value is read, and the guarantee is structural rather than
procedural — see the design note below.

## Out of scope

- **Converging the store.** `reconcile` is the next slice. This one has no writer,
  no `--fix`, and is never handed one: you cannot safely converge what you cannot
  see, and a report that quietly repaired credentials would make the repair
  unreviewable.
- **Rotation age and CI staleness.** Both are in #1596 and both need this
  inventory to exist first. `ItemSummary.Revised` is carried and documented as a
  lower bound on a credential's age, ready for that slice; nothing reads it yet.
- **The unmanaged items** (164 of 187 on 2026-09-23: items whose name no
  declaration uses). A personal vault legitimately holds what this repo
  does not manage. They are counted, never reported as findings.
- **Creating the three missing items.** That is a vault write, and it belongs to
  `reconcile` rather than to a hand-run command — the standing order is that no
  change to a remote system is ad hoc.

## Risks / open questions

- **Resolved — where the `Dotfiles/` prefix lives.** The declaration now carries
  it, rather than the code prepending it. Prepending would have made the declared
  value differ from the real one, so the report would have to undo the
  translation to avoid lying about what it found. Decided with the operator;
  `validBWFolders` and 26 registry lines moved together.
- **Resolved — the projection cannot be enforced by review.** See below.
- **Accepted — `hasField` mirrors `fieldFromItem`.** Mirroring a resolver is
  normally the defect (lesson 279). The alternative here is worse: that dispatch
  is private, and exporting a "would this resolve?" predicate would put a
  value-resolving path one refactor away from a report that must never hold one.
  The duplication is small, named, and pinned by a test that fails if the two
  diverge.
- **Open — `field: password` is answered from login presence.** `itemWire` never
  names `password`, so an item with a login block but an empty password reads as
  present. No registry entry declares `password` today. The weaker answer is the
  deliberate price of the projection.

## The design note that matters

`bw serve` has no projection: `/list/object/items` answers with every item's
complete plaintext. So the narrowing happens at the producer and is enforced by
the **type system** — `itemWire` does not name `login.password`, a field's
`value`, `card`, `identity`, `sshKey` or `passwordHistory`, and `encoding/json`
discards what the target struct does not name. A password is dropped during
decoding and never exists as a Go value. There is no line to forget to redact and
no filter that can be reordered past it.

Two members are read as content and reduced to booleans on the next line, because
their questions cannot be asked otherwise: `notes` and `login.username`. Both are
named in the type's doc comment as the exceptions they are.

## Acceptance criteria

- [x] AC1 — a declared folder name that no folder carries is reported, and the
      finding says `dotf secrets set` would CREATE rather than reuse.
- [x] AC2 — a declared item absent from the vault is reported, including for a
      **dormant** declaration on an age-backed secret.
- [x] AC3 — an item in a folder other than the declared one is reported, naming
      both places. A declared folder belongs to the declaration's plane: the
      registry refuses a folder on a plane the taxonomy gives none (`floor`,
      `personal`), so no declaration can send an item into another plane's
      folder *(added after review round 4)*.
- [x] AC4 — a declared field the item does not carry is reported, and so is a
      declaration that names no field (the reader refuses an empty field). Field
      presence agrees with `fieldFromItem` on whether the field yields a usable,
      non-empty value, for `notes`, `username`, `password` and custom fields, with
      two stated exceptions: an empty `password` or custom-field value reads as
      present, because the value-free projection never decodes those values.
      *(Amended after review round 3: it said "matches `fieldFromItem` exactly",
      which was false for empty values and untested for the agreement itself.)*
- [x] AC5 — one finding per problem, not per env var: seven vars naming one
      absent or misfiled item produce one finding. The registry refuses one item
      declared in two folders, so keying the misfiled dedupe on the item cannot
      hide a disagreement *(added after review round 3)*.
- [x] AC6 — no secret value can cross the projection, proven by marshalling the
      result and asserting distinctive planted values are absent: one for every
      member the projection reads, and for the values it must drop. *(Amended
      after review round 4: the username, a member it reads, was not planted.)*
- [x] AC7 — an unparseable inventory reports a byte count and never the body.
- [x] AC8 — the command never writes, and exits non-zero on any finding. The
      check is derived from the writer interface, so it cannot fall behind it
      *(amended after review round 4: a literal list of four writer methods
      missed the fifth)*.
- [x] AC9 — the command syncs the store before reading it and refuses a store it
      cannot sync. An item filed in a folder the folder list does not carry is
      reported as `item-folder-unknown`, never as unfoldered, and `reconcile`
      blocks on it instead of moving it *(added after review round 4)*.

## References

- Bitácora board: `mlorentedev/dotfiles#1596`
- **#586** — the curation this makes enforceable: folder taxonomy, the GitHub
  8-in-1 split (#321), naming drift. That issue is the outcome; this is the
  mechanism that reports and (next slice) converges toward it.
- ADR-028 §2 (registry as mapping SSOT), §6 (curation & naming)
- `cli/internal/secrets/bw.go` — `fieldFromItem`, the resolver `hasField` mirrors
