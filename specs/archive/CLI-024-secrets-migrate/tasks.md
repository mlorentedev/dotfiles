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

- [x] **Reuse C3 write core** — `applySet` already takes the value as a parameter, so
  `migrate` calls it directly with the age-resolved value. No refactor, no `set` change.
- [x] **`migrate` command skeleton + scope guards** — resolve by env-var id; reject with
  specific errors: a file secret, a `ci:*` consumer, an `age:` source shared with another
  entry (split-pending → C9), a missing `bw:` block, and floor. `--yes`/`--dry-run` flags.
- [x] **Idempotent short-circuit** — already-`bw` → `Loader.Verify` → exit 0, no writes.
- [x] **age read → bw write** — `ageValue` resolves via the loader's age resolver
  (`EnvFor`); written via `applySet` to the target the entry's `bw:` block declares.
- [x] **Parity gate** — re-read bw, compare under `normalizeValue`; mismatch → abort
  BEFORE any registry change, non-leaking length diff.
- [x] **Registry flip last** — `secrets.FlipRegistryToBW` (`SetBackendBW` + exported
  `AtomicWrite`); then final `Verify` via bw.
- [x] **Tests** — AC1 end-to-end (create path, registry flipped); AC2 parity mismatch
  aborts (fake `tamper`); AC3 idempotent already-bw; AC4 five scope guards; AC5 `--dry-run`.

## Closing

- [ ] Every AC covered by ≥1 test (live bw write is the canary smoke, #612 C8)
- [ ] `features.json` carries non-vacuous verification commands
- [ ] `go test ./...` green; `go vet` + `golangci-lint --exclude-use-default=false`
  (mimics CI v2) clean; `go build ./...` ok
- [ ] Additive: only the new command + the write-core refactor; `set` unchanged
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec + #612 C4
