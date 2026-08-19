---
id: "OPS-032-dr-drift-detection"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-19"
issue: "mlorentedev/dotfiles#1077"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# OPS-032-dr-drift-detection

> **Naming**: file lives at `<repo>/specs/OPS-032-dr-drift-detection/proposal.md`. `OPS-032-dr-drift-detection` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #1077 -->

The disaster-recovery chain has three properties and the repository can currently
observe one and a half of them. **Decryptable** is proven — `Backup` round-trips the
artifact it writes and deletes it on mismatch. **Restorable** is only ever proven by a
drill, and `doctor` tracks when one last happened. **Fresh** is measured in days, which
is a cheaper question than the one anyone has: *does the escrow still describe the
vault?* Those two answers diverge in exactly the case that loses data — a deleted item
disappears from `bw list`, every survivor can predate the escrow, and the age check
reports healthy about a vault that has lost a secret.

There is a second unobserved property beside it. The age root now has a declared home
(#937) and, as of 2026-08-19, an offline copy the operator holds on a USB — so losing
the Bitwarden account is survivable. Nothing checks that the key on this disk is still
*that* key. A root replaced, truncated, or restored from the wrong backup is
indistinguishable from a healthy one until the day it is needed, which is the class
this repository keeps rediscovering.

## What

**Two independent checks, one PR, two commits, in this order.**

**A. The root has not drifted (#1000 AC3).** `secrets/registry.yaml` gains an optional
`recipient:` on a `file-authority` secret — the age *public* recipient, which is public
by construction. `verify` derives the local key's recipient with `age-keygen -y` and
compares. A mismatch is FAILED and names both. A secret with no `recipient:` behaves
exactly as it does today, so the field is additive and nothing else in the registry
moves.

This deliberately does NOT compare against the offline USB copy. That medium is not
plugged in, and a check that cannot see its subject must not report on it — the USB is
verified by hand at drill time.

**B. The escrow still describes the vault (#1077).** `dotf secrets backup` writes a
companion manifest recording `count`, `max_revision`, and a SHA-256 over the sorted
`"<id>:<revisionDate>"` lines — no name, no field, no value. A later check compares the
live vault to it and reports what changed, catching all three mutations where a
timestamp catches two:

| Mutation | Detected by |
|---|---|
| Item added | `count` + `digest` |
| Item modified | `revisionDate` moves the `digest` |
| **Item deleted** | `count` + `digest` — the case the timestamp cannot see |

**C. The runbook names where the offline copy lives (#1000 AC1)** — the location, never
the value.

## Out of scope

- **The recovery drill (#1000 AC4).** Executing the chain end to end from the USB is a
  physical act and the largest remaining DR risk. Nothing here substitutes for it, and
  the PR references #1000 without a closing keyword for that reason.
- **Comparing the offline USB copy automatically.** It is not plugged in.
- **`dotf secrets backup` acquiring its own session (#1008).** The manifest inherits
  whatever session model `backup` has; changing that model is its own ticket.
- **Attachments and trashed items.** `bw export` never carried attachments and
  `bw list` does not see the trash, so the escrow's blind spots are the manifest's.
  Bounded and documented, not silently inherited.

## Risks / open questions

- **Bootstrapping, and it is the one that breaks machines.** Every escrow that exists
  today predates the manifest. A comparison that treats "no manifest" as FAILED would
  turn `doctor` red everywhere until `backup` is re-run — the same deploy-skew shape
  measured on #992 twice this week, in a third direction. Absent manifest must be
  SKIPPED with the remediation named, never FAILED and never a silent pass.
- **The comparison needs a Bitwarden session**, so it must SKIP with a reason when
  there is none. An unchecked escrow reported as fresh is this ticket's own defect
  arriving through its fix.
- **`age-keygen -y` reads the private key** and prints the public recipient. Only the
  derived public value may reach a log, an error message or a fixture.
- **Portability.** `GOOS=windows go vet ./...` is part of the loop now (#1075). Any
  new check gets the not-applicable-versus-not-checked question that the key-mode
  check had to answer: JSON and mtimes are portable; anything touching permissions or
  sessions is not assumed to be.
- **The `.gitignore` un-ignore must be an exact name** — `!escrow-manifest.json`, never
  `!*.json`. `sensitive/dr/.gitignore` is deny-by-default precisely so a plaintext
  `bw export` can never be staged; a glob reopens the door the guard exists to hold.

## Acceptance criteria

- [x] AC1 — `recipient:` parses on a `file-authority` secret, is rejected on any other
      backend, and a secret without it verifies exactly as it does today.
- [x] AC2 — `dotf secrets verify` reports FAILED when the local key's derived recipient
      differs from the declared one, naming both; observed by pointing the entry at a
      key with a different recipient, mutation confirmed present first.
- [x] AC3 — with `age-keygen` absent or unreadable the check reports its own failure
      rather than passing; the pre-existing 34 entries do not regress.
- [x] AC4 — `dotf secrets backup` writes `escrow-manifest.json` beside the escrow with
      `count`, `max_revision` and `digest`, containing no item name, field or value;
      `sensitive/dr/.gitignore` un-ignores it by exact name.
- [x] AC5 — the freshness check reports the three mutations distinctly on a synthetic
      vault: an addition, a modification, and **a deletion whose survivors all predate
      the escrow** — the case a timestamp comparison passes.
- [x] AC6 — an escrow with **no** manifest is SKIPPED with the remediation named, never
      FAILED; and with no Bitwarden session the comparison is SKIPPED with a reason,
      never OK.
- [x] AC7 — the runbook names where the offline copy of the age root lives, and never
      its value.
- [x] AC8 — `GOOS=windows go vet ./...` clean, and any property that is not applicable
      on a platform says so in its own file rather than being inferred.

## References

- Bitácora board: `mlorentedev/dotfiles#1077` (see the `issue:` frontmatter field)
- `#1000` — AC1 and AC3 are delivered here; AC4, the drill, is not and the PR says so
- `#1008` — the session model `backup` inherits
- `#992` — the deploy-skew class the bootstrapping risk belongs to
- Related ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md` §5
- Related patterns: `00_meta/patterns/pattern-verification-fails-toward-unproven.md`

<!-- archived 2026-08-19 — PR: https://github.com/mlorentedev/dotfiles/pull/1079 -->
