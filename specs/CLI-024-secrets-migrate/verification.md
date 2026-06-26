---
tags: [spec, verification, secrets, cli, go, bitwarden, migration]
created: "2026-06-26"
---

# Verification - CLI-024-secrets-migrate

## Evidence

- [x] **AC1 — migrates end-to-end** -> `TestSecretsMigrate_EndToEnd` (age value written to
  bw via the create path, registry flipped to `backend: bw`, output `migrated`).
- [x] **AC2 — parity mismatch aborts before flip** -> `TestSecretsMigrate_ParityMismatchAbortsBeforeFlip`
  (fake `tamper` makes the bw read-back differ; error + registry stays `backend: age`).
- [x] **AC3 — idempotent already-bw** -> `TestSecretsMigrate_Idempotent_AlreadyBw`
  (re-verify, `already bw`, zero writes, registry unchanged).
- [x] **AC4 — scope guards** -> `TestSecretsMigrate_ScopeGuards` (file / ci:* / shared-age /
  missing-bw / floor each error with no writes).
- [x] **AC5 — `--dry-run` inert** -> `TestSecretsMigrate_DryRunInert` (`would create` +
  `would flip registry`, no writes, registry unchanged).
- [x] **AC6 — clean + additive** -> full suite green; `go vet` + `golangci-lint
  --exclude-use-default=false` (mimics CI v2) exit 0; gofmt-clean on LF blobs.

## Test status

- `cd cli && go test ./...` -> all packages `ok`. migrate suite: 5 tests PASS
  (verbose-confirmed).
- Lint: `golangci-lint run --exclude-use-default=false ./internal/...` exit 0; `go vet`
  clean. gofmt-clean on the LF index blobs of every changed file (the local CRLF noise is
  the Windows checkout artifact, not the committed content).
- Live bw write + the real registry cutover are exercised by the canary smoke (#612 C8)
  with the operator's `bw unlock` — not CI.

## Decisions made during implementation

- **`applySet` was already the reusable write core** — it takes the value as a parameter,
  so `migrate` calls it directly with the age-resolved value; no refactor needed.
- **`AtomicWrite` exported** from render.go (was `atomicWrite`) and reused by the new
  `secrets.FlipRegistryToBW` — one atomic-write impl for both config render and the
  registry cutover.
- **Parity reuses the run-time transform** (`normalizeValue`/`EnvFor` `stripNewlines`), so
  parity = "the consumer sees the same value", not raw bytes.
- **Scope guards are explicit + specific** (file / ci:* / shared-age / missing-bw / floor),
  each with its own message — the shared-age guard is what stops the github split from
  copying one token to two items via plain `migrate`.
- **Ordering**: write+parity before the registry flip (the last mutation), so a pre-flip
  failure is a safe no-op and re-runs are idempotent.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? no new one — the create-vs-locked lesson (from C3) and
  the registry-flip-last ordering are already captured (C3 PR #626 / ADR-028 addendum).
- [ ] ADR-worthy? no — within ADR-028 + its addendum.
- [ ] New pattern for `00_meta/`? no — repo-specific.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-024-secrets-migrate/` -> `specs/archive/CLI-024-secrets-migrate/`
- [ ] Bitácora board ticket (#612 C4) updated with the PR link (ADR-018)
