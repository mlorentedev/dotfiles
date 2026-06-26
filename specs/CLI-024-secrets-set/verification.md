---
tags: [spec, verification, secrets, cli, go, bitwarden]
created: "2026-06-26"
---

# Verification - CLI-024-secrets-set

## Evidence

- [x] **AC1 — Idempotent no-op** -> `TestSecretsSet_IdempotentUnchanged`
  (cur == new value -> `unchanged`, zero writer calls).
- [x] **AC2 — Write on change, per-shape normalization** ->
  `TestSecretsSet_UpdateOnChange_EnvTrimmed` (env trailing-newline trimmed),
  `TestSecretsSet_FileShape_BytesExact` (file value byte-exact, newline preserved).
- [x] **AC3 — Field disambiguation** -> `TestSecretsSet_Disambiguation` (multi-var with
  no `[var]` errors listing vars; `[var]` writes the matching field).
- [x] **AC4 — Create-absent gated** -> `TestSecretsSet_CreateAbsent_Gated`
  (`ErrBWItemNotFound` + `--yes` -> `CreateItem`; no `--yes` non-interactive -> error,
  no create; locked vault -> fail loud, never create) +
  `secrets.TestIsNotFound`, `secrets.TestFieldFromItem_FieldNotFoundSentinel` (the
  sentinels the dispatch keys on) + `secrets.TestNewItemBody` (create JSON construction).
- [x] **AC5 — Empty refused + dry-run inert** -> `TestSecretsSet_EmptyRefused`,
  `TestSecretsSet_DryRunInert` (would update / would create, zero writes).
- [x] **AC6 — Clean + additive** -> full suite green; `go vet` + `golangci-lint` exit 0;
  gofmt-clean on LF blobs; `registry.yaml` untouched; `secrets.BWTarget` unit-tested
  (`TestBWTarget`).

## Test status

- Test suite: `cd cli && go test ./...` -> all packages `ok` (incl. `internal/cmd`,
  `internal/secrets`). `set` suite: 7 tests PASS (verbose-confirmed, none skipped).
- Lint: `golangci-lint run ./internal/cmd/... ./internal/secrets/...` -> exit 0;
  `go vet ./...` -> clean.
- Format: `gofmt -d` on the LF index blobs of all changed/new `.go` files -> empty
  (clean). NOTE: local `gofmt -l` flags many pre-existing files because this Windows
  checkout has CRLF working-tree line endings (`core.autocrlf=true`, `.go` has no
  explicit `eol=`); the index stores LF (`git ls-files --eol` = `i/lf`), so CI (Linux,
  LF) and the committed content are gofmt-clean. Pre-existing, not introduced here.
- Build/smoke: `go build ./...` ok; `dotf secrets set --help` shows the wired command.
- Live write smoke (BWPut.SetField / CreateItem shell-outs): deferred to the canary
  (#612 C8) with the operator's `bw unlock` — same coverage shape as BWGet/AgeDecrypt.
- No regressions: existing secrets/cmd tests unchanged and green.

## Decisions made during implementation

- **One field per invocation** reconciles "all three shapes" with the atomic-PR cap:
  token / multi-field / file collapse to one write path, differing only in
  `[var]` disambiguation and env-trim-vs-file-exact normalization.
- **Create-vs-locked by sentinel, not string-match at the call site.** `BWGet.Field`
  wraps `ErrBWItemNotFound` only on bw's "Not found." message; a locked/unauthenticated
  vault yields a different message -> falls to the fail-loud default branch. `applySet`
  switches on the sentinel, so a locked vault can never reach the create path. Belt-and
  -braces: create still needs confirm or `--yes`.
- **`ErrBWFieldNotFound`** distinguishes a present item missing the field (append, no
  current value) from a missing item (create) — the third branch the audit cares about.
- **stdin is consumed by the value**, so non-interactive create requires `--yes` (no
  channel to confirm on); the TTY path reads value (hidden) then confirm separately.
- **New dep `golang.org/x/term`** for cross-platform hidden input (Windows included).
- **`CreateItem` reuses `setItemField`** over a minimal login template, so a created and
  an edited item place a field identically.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? **yes** — the create-vs-locked
  discrimination (a locked vault must never be read as an absent item, or a write path
  spawns duplicates); pairs with the `bw` CLI message-match fragility. Capture on merge.
- [ ] ADR-worthy decision? no — within ADR-028's envelope.
- [ ] New pattern candidate for `00_meta/patterns/`? no — repo-specific to the bw seam.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-024-secrets-set/` -> `specs/archive/CLI-024-secrets-set/`
- [ ] Bitácora board ticket (#612 C3) updated with PR link (ADR-018)
- [ ] Promotions above executed (the `docs/lessons.md` entry)
