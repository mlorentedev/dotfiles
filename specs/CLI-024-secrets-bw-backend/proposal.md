---
id: "CLI-024-secrets-bw-backend"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-26"
issue: "mlorentedev/dotfiles#585"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, secrets, cli, go, bitwarden]
template_version: "1.0"
---

# CLI-024-secrets-bw-backend

## Why

The registry (`secrets/registry.yaml`) already declares `backend: bw` as a target
(ADR-028 §2), but it is a **no-op today**: `Secret.AgeBacked()` gates bw out of
`Registry.Entries()`, and `showSource` rejects it ("not yet supported, ADR-028
Phase 3"). No secret can resolve from Bitwarden, so the age→bw migration that makes
Bitwarden the SSOT cannot begin.

This spec makes the **bw backend live** — the resolver core that turns a registry
`backend: bw` entry into a decrypted value via the Bitwarden CLI. It is the Go piece
of #585 and is **fully unit-testable without a real Bitwarden vault or an unlock**
(a `BWReader` seam, mirroring the existing `Decryptor` seam). It deliberately
migrates **zero** secrets: flipping the ~20 dev/infra entries `age → bw` + rotating
+ retiring their age files needs the operator's unlocked `bw` session and is a
separate, non-blocking follow-up (#585 AC2–4).

## What

Concrete, observable changes after this PR:

1. **Registry schema — a `bw` source.** A bw secret declares `bw: { item, field }`
   (single var) or a shared `bw: { item }` plus a per-var `field` override (the
   multi-field item case, e.g. x-twitter's 7 values collapsed into one Bitwarden
   item). Mirrors the age `{ age: file }` / per-var `{ age: file }` shapes.
   Validation requires an `item` + a `field` for every bw-exposed var (parallel to
   `checkAgeSources`), fail-fast.
2. **A per-backend `Resolver` interface** (Open/Closed). `EnvFor` dispatches on
   `Entry.Backend` through a resolver map: the **age resolver** wraps the existing
   `Decryptor`; the **bw resolver** sits behind a `BWReader` seam. Adding a future
   backend (`bws`, Vault) is a new `Resolver` + one map entry — no edit to the
   resolution loop, no change to consumers.
3. **`Registry.Entries()` stops skipping bw** — it emits bw entries tagged
   `Backend: "bw", Item, Field`; the whole `--only` / file-materialization /
   fail-fast / newline-strip machinery is reused unchanged.
4. **`dotf secrets show <bw-id>` and `run --only <bw-id>` resolve from Bitwarden**
   when `bw` is unlocked; `showSource` no longer rejects bw. A **locked or
   unreachable** Bitwarden produces a clear, actionable error (never a hang, never
   plaintext to disk).
5. **Production `BWReader` = a `bw get` shell-out** (`BWGet`), the bw analog of
   `AgeDecrypt`'s `age --decrypt`. `bw serve` (a local REST daemon holding the
   unlocked vault, faster for batch reads) is a **drop-in perf upgrade behind the
   same seam**, documented in ADR-028 — not implemented here.

`registry.yaml` itself is **not** edited to flip any secret: the bw path is exercised
only by test fixtures, so this PR adds the *capability* with **zero live behaviour
change** until the migration follow-up flips the first entry.

## Out of scope

- **The migration itself** (#585 AC2–4): adding the ~20 secrets to Bitwarden,
  flipping `backend: age → bw`, rotating, retiring `sensitive/<file>.secret.age`.
  Needs the operator's unlocked `bw` session — separate, non-blocking follow-up.
- **`bw serve` daemon lifecycle** (auto-start / health-check / shutdown) — a perf
  optimisation behind the `BWReader` seam, not correctness. Follow-up.
- **Bitwarden Secrets Manager (`bws`) / HashiCorp Vault** — the infra-grade,
  identity-based path. Documented as the upgrade route the facade keeps open; **not**
  adopted (would reopen ADR-028 and add a second store; over-scaled for personal use).
- **Folder taxonomy / item-naming curation, the GitHub token split (#321), the
  offline age key + DR escrow** — the curation issue. `item` here is the **unique
  item name (or ID)**; the Bitwarden folder is org metadata, not a lookup key.

## Risks / open questions

- **Item + field resolution.** `bw get item <name-or-id>` keys on the item **name or
  ID**, not the folder — so the registry keys on `item` (unique within the vault) and
  `field`. `BWGet` fetches the item JSON once and picks the field: `password` /
  `username` / `notes` from the typed login object, anything else from custom
  `fields[]` by name. An **ambiguous** item name → `bw` errors clearly (the curated
  `Dotfiles/*` namespace keeps names unique); resolved → done.
- **Locked / non-interactive bw must never hang.** `BWGet` runs `bw --nointeraction`
  and relies on `BW_SESSION` in the environment for the unlock token; a locked vault,
  missing session, or unknown item maps to a clear error ("Bitwarden locked — run
  `bw unlock` / export `BW_SESSION`"), surfaced fail-fast like an age decrypt failure.
- **No plaintext to disk.** Env secrets stay in memory (same as age); bw **file**
  secrets materialise `0600` through the existing `materialize()` path. No new at-rest
  copy.
- **Testability boundary.** The `BWReader` seam unit-tests the resolver core with a
  map-backed fake — **no Bitwarden in CI**, exactly as `fakeDecryptor` does for age.
  `BWGet` itself (thin I/O to the `bw` binary) is verified by a **live smoke** with
  the operator's unlocked session, not in CI — the same coverage shape `AgeDecrypt`
  already has (untested real impl, tested fake).
- **Determinism invariant.** A var exposed by two secrets is already rejected by
  validation; keep that for bw (render needs one source per var).

## Acceptance criteria

- [ ] **AC1** — The registry parses + validates `bw: { item, field }` and the per-var
  `field` override; a bw secret missing an item/field source is rejected fail-fast.
  *Verify:* table-driven Go test (valid shapes + the missing-source error case).
- [ ] **AC2** — `Registry.Entries()` emits bw entries tagged `Backend: "bw", Item,
  Field` (no longer skipped), and `EnvFor` resolves them through the injected
  `BWReader`. *Verify:* Go test with a fake `BWReader` covering env + file + `--only`.
- [ ] **AC3** — `dotf secrets show <bw-id>` and `run --only <bw-id>` resolve from
  Bitwarden via the bw resolver; `showSource` no longer rejects bw. *Verify:*
  command-layer Go test with a fake `BWReader`.
- [ ] **AC4** — A locked / unreachable Bitwarden yields a clear, actionable error
  (no hang, no plaintext to disk). *Verify:* Go test where the fake `BWReader`
  returns an error → `EnvFor`/`show` fail fast with a wrapped message.
- [ ] **AC5** — `go test ./... && go vet ./... && gofmt -l` clean; `registry.yaml`
  unchanged (no secret flipped); the migration remains a documented follow-up.

## References

- ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md` (§2 backends, Phased plan
  step 3; the upgrade path: bw serve / bws / Vault behind the facade).
- Issue: `mlorentedev/dotfiles#585` (this PR delivers AC1; AC2–4 = the migration
  follow-up).
- Reuse: `cli/internal/secrets/{registry,resolve,secrets}.go`,
  `cli/internal/cmd/secrets.go` (the `ageDecryptor` seam, `registryPath()`,
  `showSource`); test style: `secrets_test.go` (`fakeDecryptor`), `registry_test.go`.
- Prior: specs `CLI-024-secrets-run-jit`, `CLI-024-secrets-registry`,
  `CLI-025-secrets-render` (all merged) — the resolution path this extends.
