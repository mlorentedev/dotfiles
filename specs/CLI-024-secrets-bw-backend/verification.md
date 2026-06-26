---
tags: [spec, verification, secrets, cli, go, bitwarden]
created: "2026-06-26"
---

# Verification - CLI-024-secrets-bw-backend

## Evidence

All AC verified by table-driven Go tests (no Bitwarden vault, no unlock — the
`BWReader`/`Decryptor` seams inject fakes). Run from `cli/`.

- [x] **AC1** (bw schema parses + validates; missing source rejected) ->
  `TestParseRegistry_BwShapes` (single / multi-field / file shapes),
  `TestParseRegistry_BwValidation` (missing bw block / item / env field / file field).
- [x] **AC2** (`Entries()` emits bw entries tagged Backend/Item/Field; `EnvFor`
  resolves via the injected `BWReader`) -> `TestRegistry_Entries_IncludesBwBackend`,
  `TestEnvFor_BwEnvSecret`, `TestEnvFor_BwFileSecret_Materialized`,
  `TestEnvFor_MixedBackends_OnlyFilter`.
- [x] **AC3** (`show`/`run --only` resolve bw; `showSource` no longer rejects bw) ->
  `TestSecretsShow_ResolvesBwBackend`, `TestSecretsRun_ResolvesBwBackend` (cmd pkg).
- [x] **AC4** (locked/unreachable Bitwarden → clear, actionable error; no hang, no
  plaintext to disk) -> `TestEnvFor_BwNilReader_ClearError` (asserts the "bw backend
  unavailable" message), `TestEnvFor_BwReaderError_FailsFast`,
  `TestEnvFor_UnknownBackend_Errors`. `BWGet` runs `bw --nointeraction` (cannot block).
- [x] **AC5** (suite + vet + fmt clean; registry.yaml unchanged) -> see Test status;
  `git diff` touches no `secrets/registry.yaml` (no secret flipped).
- [x] Bonus: `fieldFromItem` JSON extraction (login / notes / custom fields / unknown
  / bad JSON) -> `TestFieldFromItem`.

## Test status

- `go test ./internal/secrets/ ./internal/cmd/ -count=1` -> **ok** (both packages).
  bw-focused run: 12 secrets-pkg tests + `TestSecretsShow_ResolvesBwBackend` +
  `TestSecretsRun_ResolvesBwBackend` all PASS.
- `go vet ./...` -> clean. `go build ./...` -> clean.
- `gofmt -l` on the staged (LF) blobs -> clean for every changed file
  (`git show :<file> | gofmt -l` empty). Windows working-tree shows CRLF, but
  `git ls-files --eol` reports `i/lf` — git stores LF, so CI's Linux gofmt passes.
- No regressions: the full `go test ./...` is green **except** `internal/mem`
  `TestVaultHealth/...clean_stub`, which **fails identically on a clean `main`
  checkout** (environment-dependent vault-health stub on this Windows box) — not
  touched by this change.
- Live smoke against a real unlocked Bitwarden: **deferred to the migration
  follow-up** (needs the operator's `bw unlock` session). `BWGet` is thin I/O,
  untested in CI exactly as `AgeDecrypt` is; `fieldFromItem` (the parsing logic) is
  unit-tested.

## Decisions made during implementation

- **`item` keys on the Bitwarden item name/id, not a folder path.** `bw get item`
  resolves by name/id; the folder is org metadata (curation issue). Refined the
  issue's `bw: {folder, item, field}` sketch to `bw: {item, field}`.
- **Resolver interface over a `switch`** — the user asked for the scalable/maintainable
  shape. A per-backend `Resolver` registered in a map makes a future `bws`/Vault
  backend a new impl + one map entry (Open/Closed), with no edit to the resolution
  loop. `""` backend maps to age for caller back-compat.
- **`BWGet` = `bw get` shell-out**, the boring/correct mirror of `AgeDecrypt`'s
  `age --decrypt`. `bw serve` (local REST, faster batch reads) is a drop-in upgrade
  behind the same seam, deferred (ADR-028). Documented that serve must stay
  localhost-only (unauthenticated by design).
- **`ls --pairs` skips bw entries** (no age source to emit) so the age-based
  github-secrets-manager push path never emits a malformed `VAR\t` line; the CI-push
  path for bw secrets is rethought in the migration follow-up.
- **Zero migration in this PR** — `registry.yaml` is untouched; the bw path is
  exercised only by fixtures, so the capability lands with no live behaviour change.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? **no** — the design follows ADR-028; no
  new project-level gotcha beyond what the ADR records.
- [ ] ADR-worthy decision? **no** — ADR-028 already decided bw-as-SSOT; this is its
  implementation. (The `bw serve` localhost-only / `bw`-vs-`bws` notes are captured in
  the ADR upgrade-path section and the runbook, not a new ADR.)
- [ ] New pattern candidate for `00_meta/patterns/`? **no** — repo-specific.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-024-secrets-bw-backend/` -> `specs/archive/CLI-024-secrets-bw-backend/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
