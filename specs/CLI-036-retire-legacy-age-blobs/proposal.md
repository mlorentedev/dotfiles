---
id: "CLI-036-retire-legacy-age-blobs"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-09-25"
issue: "mlorentedev/dotfiles#938"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, secrets, age, doctor]
template_version: "1.0"
---

# CLI-036: retire the legacy age blobs

## Why

<!-- from issue #938: CLI-036: no retire path, so migrated secrets keep a live age copy forever -->

`sensitive/` still commits 32 `*.secret.age` blobs, and the registry claims one of them
(`id_ed25519`, the `SSH_KEY` floor). The other 31 are what ADR-002's per-secret age store left
behind when every app, infra and personal secret moved to Bitwarden. `migrate` dropped their
`age:` pointers (#971). No code reads them: the resolver only opens a blob that an age-backed
entry names, and no shell loads them any more. They are decryptable copies of values that were
current before the rotations; the Hetzner token among them answers `401`. Setup copies them into
`~/.dotfiles/sensitive/` on every machine and never removes one (#802). Doctor sees them and only
WARNs, on the belief that they are "the ADR-028 floor". ADR-028 §5 says otherwise: the floor is
the escrow, a verified, age-encrypted export of the **whole** vault (`sensitive/dr/`, 183 items at
`f42c212`), and §3 gates age-file retirement on exactly that verified round-trip.

The repository is public. Removing the blobs from `HEAD` leaves them in history, decryptable with
the age key, so this is hygiene, not remediation. The mitigation for a blob that held a live value
is rotation, and the protection of the key is #1000.

## What

| Before | After |
|---|---|
| 32 blobs committed, 31 claimed by no entry | 1 blob committed (`id_ed25519`), claimed by `SSH_KEY` |
| Doctor: one WARN "N blobs claimed by no entry … do not bulk-delete" once any secret has migrated | Doctor: one FAIL per unclaimed blob in the mirror, naming it, whatever has migrated |
| Nothing removes a blob from `~/.dotfiles/sensitive/` | `dotf doctor --fix` prunes a mirror blob when no entry claims it, the checkout no longer has it, and the checkout holds a DR escrow |
| ADR-028 is read (by the code comment) as making per-secret blobs part of the floor | ADR-028 amendment: the escrow is the floor; per-secret blobs exist only for `age-offline` entries |

The prune follows the semantic decided on #802 (2026-08-07): **`doctor --fix` prunes, setup only
copies.** Setup is not edited.

## Out of scope

- A `dotf secrets retire <id>` subcommand, the shape #938 first proposed. Nothing is left to retire
  through it: no age-backed entry remains outside the floor, and a future migration leaves a blob
  that doctor now FAILs on by name. Deletion over construction.
- Restoring `age:` pointers on migrated entries (#971's proposal). It would re-document 31 blobs
  this change deletes; #971 closes as superseded by the ADR amendment.
- The other copy-only mirror surfaces #802 measured (`scripts/`, `harness/`). `harness/` is already
  pruned by `checkHarnessMirrorOrphans`; `scripts/` belongs to the CLI drive.
- History rewriting. The repo is public and forked copies exist; the answer is rotation and #1000.
- `docs/secrets-inventory.md`, which describes itself as the migration's record, not a mirror.

## Risks / open questions

- **The checkout resolver can resolve the wrong repo.** `resolveRepoDir` proves "a git checkout",
  not "the dotfiles checkout" (the lesson recorded at `checkHarnessMirrorOrphans`). Against an
  unrelated repo, every mirror blob would look absent. The prune therefore refuses unless the
  checkout has `secrets/registry.yaml` **and** `sensitive/dr/bitwarden-export.age`; that pair is
  also the ADR-028 §3 gate.
- **The mirror is the checkout.** Then nothing is pruned (the same guard as the harness arm), and
  the FAIL tells the operator to `git rm` the blob.
- **A blob still in the checkout** is never pruned from the mirror: setup would copy it back on the
  next run. The FAIL names the checkout path to remove it from.
- **Machine-local files in the mirror** (`env-mapping.conf`, `README.md`, `dr/`) are never
  candidates: only top-level regular files named `*.secret.age`, plus the shell-era partial writes
  `*.secret.age.tmp.*` (one on msi, from 2026-05-11, age-encrypted).
- **The Windows box** runs the same Go check. Its mirror is pruned when `dotf` there is at a release
  that contains this change, run with `--fix`.

## Acceptance criteria

- [ ] AC1 — `git ls-files 'sensitive/*.secret.age'` lists exactly the blobs an age-backed registry entry names (today: `sensitive/id_ed25519.secret.age`), and a test fails if a committed blob is claimed by no entry.
- [ ] AC2 — `dotf doctor` reports each unclaimed `*.secret.age` in the mirror as a FAIL naming the file, with a registry that has bw-migrated entries. The WARN "claimed by no registry entry … Tracked as #971" no longer exists.
- [ ] AC3 — `dotf doctor --fix` removes an unclaimed mirror blob that the checkout lacks, reports one FIX line per file, and a second run reports nothing to fix.
- [ ] AC4 — `--fix` removes nothing when (a) the checkout still has the blob, (b) the checkout lacks `secrets/registry.yaml` or the DR escrow, (c) the mirror is the checkout, or (d) the file is not `*.secret.age` / `*.secret.age.tmp.*`. Each case is a Go test.
- [ ] AC5 — ADR-028 carries an amendment stating that the escrow is the DR floor and per-secret blobs exist only for `age-offline` entries; the comment at `reportUnreferencedBlobs` no longer claims otherwise.
- [ ] AC6 — On msi, after `go run ./cmd/dotf doctor --fix` from the worktree: the mirror holds `id_ed25519.secret.age` as its only `*.secret.age*` file, and `dotf secrets drift` still reports 0 findings.

## References

- Issues: #938 (this spec's gate), #971 (superseded), #802 (the prune semantic, decided 2026-08-07), #1000 (the offline key).
- ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md` §3, §5.
- Precedent: `checkHarnessMirrorOrphans` in `cli/internal/doctor/checks_deploy.go`.
- Patterns: `00_meta/patterns/pattern-spec-driven-development.md`.
