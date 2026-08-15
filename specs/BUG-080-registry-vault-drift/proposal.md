---
id: "BUG-080-registry-vault-drift"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-15"
issue: "mlorentedev/dotfiles#985"
tags: [spec, proposal, secrets, doctor, bitwarden]
template_version: "1.0"
---

# BUG-080-registry-vault-drift

<!-- from issue #985: dotf secrets run without --only is dead for every consumer — DOCKERHUB_TOKEN maps to an item that does not exist -->

## Why

`secrets/registry.yaml` is the mapping SSOT; Bitwarden is the store it maps into.
Nothing checks that the two agree. On 2026-08-15 they did not: the registry names
a vault item `dockerhub`, the vault holds `DockerHub`, and item lookup is an
exact-name match.

That single mismatch is not local. `dotf secrets run` with no `--only` resolves
the **whole** registry and fails fast on the first unresolvable entry — by
design, so a child never launches with a partially-populated secret set. So one
stale name takes down every unscoped run: the `pi` shell wrapper, and
`dotf spec review`, whose launcher builds an unscoped run. The adversarial-review
gate was therefore unrunnable **for every spec in every repo**, which in turn
blocks `dotf spec archive`, which blocks `spec-gate` on any PR closing a spec's
issue. Two PRs were stuck behind it before anyone identified the cause.

The failure gave no usable signal. The review launcher printed `[OK] Review
running detached` over a process that had already died, leaving a 0-byte
transcript; one session diagnosed a locked vault and was heading for a fix that
would have changed nothing. A mapping error surfaced as "the reviewer produced
nothing".

## What

Two changes: one correctness fix in the registry, and the guard that makes this
class visible instead of inferred.

**1. `DOCKERHUB_TOKEN` maps to the item's `PAT` field, not `password`.** Found
while reading the item that the registry could not resolve. The entry pointed at
the **account password** while a scoped personal access token sat beside it in
the same item. Verified live: both authenticate against the Hub API with HTTP
200, which is exactly why nothing ever surfaced it — the wrong credential worked.
This is a blast-radius fix, not the repair of a broken credential.

**2. A `dotf doctor` section asserting the registry→vault mapping.** For every
`backend: bw` entry, the item it names must exist. Name-only: it lists item names
through the `bw serve` daemon and compares sets, never reading a field, resolving
a secret, or seeing a value — so it is cheap enough for the full sweep and safe
anywhere. It reports which item is missing, which secret ids named it, and what
breaks, because the symptom never points back here.

The vault-side rename (`DockerHub` → `dockerhub`, matching ADR-028's kebab
`<service>-<purpose>` convention and the other 16 managed items) is the operator
action that clears the current instance. It is not in this PR: it mutates a live
password vault and belongs to its own authorised step.

## Out of scope

- **Making `dotf secrets run` tolerant of an unresolvable secret the caller never
  asked for.** This is the deeper fix, and it is a deliberate reversal of the
  fail-fast contract added in #612 A1 — a decision, not a bug fix. Until it is
  taken, the next registry/vault drift breaks the same things; this spec makes
  that drift *visible before* it does, which is the part that can be shipped
  without re-opening a settled design.
- **The review launcher announcing success over a dead child.** Filed as #989.
  Scoping or fixing the mapping changes the current cause of death; it does not
  teach the launcher to tell a live review from a corpse.
- **The `Dotfiles/` folder-prefix rename.** PR #982, another session.
- **The broader vault deduplication** (manual items shadowing managed ones,
  literal duplicates, the 11-field `GitHub` item). Its own spec, unstarted, and
  it requires the CR overlay because it mutates a live vault.

## Risks / open questions

- **The check needs a reachable, unlocked vault.** When the daemon is absent or
  locked it SKIPs rather than failing — `checkBitwardenReach` owns that severity,
  and reporting it twice inflates the failure count and trains the reader to
  ignore the specific section. Consequence, stated: on a locked machine this
  check proves nothing, which is correct but worth knowing.
- **It cannot run in CI**, for the same reason. It is a local-operator check, and
  the drift it catches is between a checkout and a personal vault — a pairing CI
  never has.
- **`dotf secrets backup` is currently unusable**, so the sanctioned DR escrow
  could not be taken before any vault work: `bw export` goes through the CLI's
  own session, which is locked, while the daemon holds a separate unlocked one.
  Not caused by this change, but it is the reason the vault rename is deferred
  rather than done here, and it deserves its own ticket.
- **Two other PRs touch `secrets/registry.yaml`** (#982 folders, #984 the
  `validate:` marker). Conflicts are line-local and trivial; noted so whoever
  merges second expects them.

## Acceptance criteria

- [ ] **AC1** `DOCKERHUB_TOKEN` resolves the item's `PAT` field, and the change is
      recorded with the evidence that both credentials authenticate — so a later
      reader does not "fix" it back on the assumption the old one was broken.
- [ ] **AC2** A doctor section FAILs when a `backend: bw` entry names an item the
      vault does not hold, naming the item, every secret id that named it, and the
      consequence (unscoped `dotf secrets run` fails, including `dotf spec review`).
- [ ] **AC3** The check never reads a field or a value — names only.
- [ ] **AC4** An unavailable vault (locked, absent daemon, transport error) SKIPs
      and produces no failure; a registry with no `bw` entries SKIPs without
      touching the vault at all, and does not emit a PASS implying it checked.
- [ ] **AC5** Observed failing against the **real** vault state, not only a
      fixture: the live run reports the `dockerhub` drift.

## References

- Bitácora board: mlorentedev/dotfiles#985
- #989 — the launcher reporting success over a dead review
- #612 A1 — the fail-fast contract that turns one bad entry into a total outage
- `docs/adr/adr-028-secrets-two-tier-bitwarden-age` — the mapping SSOT and the naming convention
- #982 (folder prefix), #984 (`validate:` marker) — the other two live edits to the registry
