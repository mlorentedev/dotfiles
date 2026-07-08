---
id: "CLI-024-secrets-verify"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-26"
issue: "mlorentedev/dotfiles#612"
tags: [spec, proposal, secrets, cli, go]
template_version: "1.0"
---

# CLI-024-secrets-verify

## Why

The age→bw migration (#612) needs to be **reproducible and verifiable**: after
flipping a secret to `backend: bw` you must be able to assert, from the CLI, that it
still resolves — without printing the value or materializing a file. Today there is
no such check. `dotf doctor` only verifies tool presence (bw/age installed), not that
each registry secret actually resolves; `show` prints the value (leak, and single-env
only); `run` needs a child command. `dotf secrets verify` is the read-only health
check that makes the migration safe to drive incrementally and the registry's
invariant ("every declared secret resolves") testable. It is the foundation of #612
Phase C — zero write risk, and it exercises every backend and shape, so it doubles as
the idempotency/repro gate before any write command lands.

## What

`dotf secrets verify [<id>...]` resolves each selected registry secret through the
existing backend resolvers and reports a per-var status — **never the value**:

- **OK** — resolved to a non-empty value.
- **MISSING** — genuinely absent on this machine (`ErrSecretAbsent`: the age file
  isn't provisioned here). Reported, non-fatal.
- **FAILED** — a real failure (wrong age key, locked/missing Bitwarden session, empty
  value, item/field typo, unknown backend), shown with its specific cause.

No arguments → verify all entries. One or more ids/var-names → verify just those
(reusing the `--only` selector semantics). Output is one aligned line per var
(`STATUS  VAR  backend [: error]`) plus a summary (`N ok, M missing, K failed`). Exit
code is non-zero when any secret **FAILED** (MISSING alone is tolerated — a machine
need not hold every secret); `--require-all` also fails on MISSING.

A new `Loader.Verify(entry)` resolves the bytes via the backend resolver **without
materializing file secrets and without returning the value**, applying the same
empty-value rejection as `run` — so verify has no side effects (no files written, no
secret printed) and matches `run`'s resolution semantics exactly.

## Out of scope

- Any write/mutation (`set`/`migrate`/`rotate`/`retire`) — later in #612 Phase C.
- A `doctor` integration / rotation-staleness check (`rotated_at`) — #612 M4.
- Caching multi-field bw items into one fetch (#612 B-perf) — verify reuses the
  current per-entry resolve; a multi-field item is fetched per var (acceptable for a
  health check).

## Risks / open questions

- **No value leak.** `Verify` must discard the resolved bytes and never write them;
  it returns only an error (or nil). File secrets are resolved but **not** materialized
  (the resolver returns bytes; materialization lives in `EnvFor`, which verify does
  not call).
- **Locked Bitwarden → all bw secrets FAILED.** Correct (verify reports the real
  state); the operator scopes with `verify <id>` or unlocks first. Document it.
- **Exit semantics.** FAILED → non-zero (a health check that hides failures is
  useless); MISSING → exit 0 by default (partial provisioning is legitimate),
  `--require-all` to tighten.

## Acceptance criteria

- [ ] **AC1** — `Loader.Verify(entry)` returns nil for a resolvable non-empty secret,
  `ErrSecretAbsent` (wrapped) for an absent one, and a real error otherwise; it
  materializes **no** file and exposes **no** value. *Verify:* Go tests (fake
  decryptor / fake BWReader: ok / empty / absent / error; assert no file written for a
  file entry).
- [ ] **AC2** — `dotf secrets verify` reports OK/MISSING/FAILED per var and a summary,
  prints no secret values, and exits non-zero iff any FAILED. *Verify:* command Go
  test (mixed registry; assert statuses, no value in output, exit code).
- [ ] **AC3** — `dotf secrets verify <id>` scopes to the selected secret(s) via the
  `--only` selector; an unknown id errors. *Verify:* Go test.
- [ ] **AC4** — `go test ./... && go vet && gofmt` clean; no existing behaviour changed
  (verify is additive).

## References

- Issue / backlog: `mlorentedev/dotfiles#612` (Phase C, C1).
- Reuse: `cli/internal/secrets/resolve.go` (`Resolver`/`resolvers`, `ErrSecretAbsent`),
  `cli/internal/cmd/secrets.go` (`secretLoader`, `resolveOnly`, `registryPath`).
- ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md`.
