---
id: "CLI-024-secrets-file-migrate"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-08-14"
issue: "mlorentedev/dotfiles#964"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-024-secrets-file-migrate

> **Naming**: file lives at `<repo>/specs/CLI-024-secrets-file-migrate/proposal.md`. `CLI-024-secrets-file-migrate` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #964: dotf secrets migrate: file-secret path (6 secrets stuck on age) -->

`dotf secrets migrate` (#585, #627) only handles the single-scalar env-var shape; `assertMigratable`/`migrateGuard` explicitly refuse any `expose: { file: ... }` secret. Six registered secrets (kubeconfig, four backup/recovery codes) are stuck permanently on `backend: age` as a result — not because migrating them is hard, but because nobody has designed the write path for the case where the value is file content instead of an env token. Without this, the #585 age→bw cutover can never reach 100% for this registry; it stalls at 22/28 migratable secrets forever.

## What

`dotf secrets migrate <id>` accepts a file secret (`expose: { file: ... }`) and cuts it over exactly like an env secret does today: read the age plaintext byte-exact (no transformation — file secrets are already byte-exact by convention, see `normalizeValue(value, isFile=true)`), write it to the secret's declared `bw:` target, parity-gate the read-back, then flip the registry. `SetBackendBW`'s line-surgery already handles this shape blindly; the change is relaxing `assertMigratable`'s guard once the write side is ready, plus a byte-exact (not `ageValue`'s trailing-newline-trim) resolver for the migrate path. Also fixes `registry_write.go`'s stale "#612 M3/M6" comment (that milestone numbering doesn't exist in #612, which uses A/B/C phase letters) in the same change, since it directly describes the guard this PR touches.

## Out of scope

- `ZOHO_RECOVERY_CODE` migrating for real in this PR — it shares the ambiguous `zoho` Bitwarden item tracked separately in #962, so it stays on `backend: age` even after this ships; it rides the same new code path once #962 resolves the item mapping.
- `dotf secrets set`/`add` gaining a new file-write mode of its own — `applySet` already supports `isFile=true` (used by `secrets set` for file secrets today, per the "TestSetBackendBW_Guards / file secret" fixture); this spec only wires `migrate`'s read side and guard to use it, not touch its write side.
- Retire/backup of the six `.secret.age` files after migration — separate front, #612 C6.
- Repointing `STRIPE_BACKUP_CODE` onto the `stripe-api-key` item — decided (below): ships as-declared, own item.

## Risks / open questions

- **STRIPE_BACKUP_CODE grouping — decided.** Its registry entry carried an inline `# GROUP?: own item, or a field on stripe-api-key's item` comment. Resolved: keep the registry's current declared target (`bw: { item: stripe-backup-code, field: notes }`, a dedicated item) — zero registry change, ships in this PR. Consolidating onto `stripe-api-key` (one Stripe item total) is left as a future follow-up if wanted, not part of this spec.
- **Byte-exact contract must hold end-to-end (THEORETICAL, no incident yet):** file secrets get zero transformation on write (`isFile=true` → no trim at all, unlike env secrets' single trailing-newline trim). The migrate-side age reader must match exactly, or a KUBECONFIG's trailing newline (or lack of one) silently diverges from what's on disk today. Needs a dedicated byte-exact multi-line test through `migrateExec`, not reuse of the env-secret trim-based test.
- **KUBECONFIG's `mode: "0600"` field:** unrelated to the write path (`materialize`'s Mode threading is #612 B2, a read-side concern) — noted here only so it is not confused with this spec's scope.

## Acceptance criteria

- [x] AC1: `dotf secrets migrate KUBECONFIG --yes` (and the other four non-Zoho file secrets) succeeds: writes byte-exact age plaintext to the declared bw target, parity-gates the read-back, flips the registry to `backend: bw`.
- [x] AC2: `dotf secrets migrate ZOHO_RECOVERY_CODE` fails, now surfacing the pre-existing `zoho` item-name ambiguity from #962 (previously masked by the blanket file-secret guard this PR removes) — a real blocker correctly exposed, not a bug this PR introduced.
- [x] AC3: A byte-exact end-to-end test (via `migrateExec`, multi-line fixture with no trailing newline trim) proves file-secret migration preserves content exactly — the analog of `TestSecretsMigrate_PreservesInteriorNewlines` but for the `isFile=true` no-trim contract.
- [x] AC4: `registry_write.go`'s stale "#612 M3/M6" comment is corrected to describe the real guard, not a nonexistent milestone.
- [x] AC5: `dotf secrets verify` reports 33/33 OK after migrating the five, with `ZOHO_RECOVERY_CODE` still correctly reporting its pre-existing failure mode.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related ADR: `<repo>/docs/adr/adr-XXX.md` (if any)
- Related patterns: `00_meta/patterns/<pattern>.md` (if any)
