---
tags: [spec, tasks, secrets, cli, go, bitwarden, migration]
created: "2026-06-26"
---

# Tasks - CLI-024-secrets-migrate

> TDD order. One task = one focused commit.
> **Gated on #621 (C3) merging**: tasks 1–2 below need C3's bw write core. Rebase this
> branch onto main once #621 lands, then implement.

## Setup

- [x] Branch from main: `feat/secrets-migrate`
- [x] `proposal.md` complete; acceptance criteria testable
- [x] #621 (C3 `set`) merged + this branch rebased onto main (has the write core)

## Implementation

- [ ] **Refactor C3 write core** into a reusable helper that takes an explicit value
  (`writeBWValue(item, field, value, isFile, dryRun, assumeYes)`), so both `set` (value
  from stdin/prompt) and `migrate` (value from age) share one create/update/parity path.
  No behaviour change to `set`; its tests stay green.
- [ ] **`migrate` command skeleton + scope guards** — resolve by env-var id; reject with
  specific errors: a file secret, a `ci:*` consumer, an `age:` source shared with another
  entry (split-pending → C9), and a missing `bw:` block. `--item`/`--field`/`--yes`/
  `--dry-run` flags. Test the guards first (red).
- [ ] **Idempotent short-circuit** — already-`bw` → `Loader.Verify` → exit 0, no writes.
- [ ] **age read → bw write** — resolve the age value via the loader's age resolver;
  write it via the shared core to the target the entry's `bw:` block declares (model A1,
  the SSOT). Missing `bw:` block → error; `--item`/`--field` only as an override.
- [ ] **Parity gate** — re-read bw, compare resolved values (env `stripNewlines`; file
  exact); mismatch → abort BEFORE any registry change, non-leaking diff.
- [ ] **Registry flip last** — `SetBackendBW` → atomic write of `secrets/registry.yaml`
  (reuse render's `atomicWrite`); then final `Verify` via bw.
- [ ] **Tests** — AC1 migrate end-to-end (fake age resolver + fake bw writer/reader,
  assert registry bytes flipped + age file untouched); AC2 parity mismatch aborts; AC3
  idempotent already-bw; AC4 scope guards per shape; AC5 `--dry-run` inert.

## Closing

- [ ] Every AC covered by ≥1 test (live bw write is the canary smoke, #612 C8)
- [ ] `features.json` carries non-vacuous verification commands
- [ ] `go test ./...` green; `go vet` + `golangci-lint --exclude-use-default=false`
  (mimics CI v2) clean; `go build ./...` ok
- [ ] Additive: only the new command + the write-core refactor; `set` unchanged
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec + #612 C4
