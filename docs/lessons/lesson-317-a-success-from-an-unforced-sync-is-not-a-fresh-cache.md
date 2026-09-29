---
id: lesson-317
type: lesson
status: active
created: "2026-09-29"
owner: manu
tags: [lesson, secrets, bitwarden, bw-serve, caching]
---

# 317 — A success from an unforced sync is not a fresh cache

## What happened

Right after `dotf secrets backup`, `dotf doctor` reported that the escrow "no longer describes the vault". Both sides counted 182 items. The escrow's newest revision was 2026-09-29T00:30Z, and the `bw serve` daemon's newest was 2026-09-25. The difference was one item edited in the app: the `bw` CLI saw its new revision, and the daemon still served its March revision.

`POST /sync` answered `{"success":true}` and changed nothing. An unforced sync compares the account's revision date with the `lastSync` that the daemon and the CLI share. `backup` runs a CLI `bw sync`, which moved `lastSync` past the edit, so the daemon concluded it was current and kept its in-memory cache. `POST /sync?force=true` refreshed it, and the daemon's digest then matched the escrow's byte for byte.

The doctor warning was the harmless face of this. The same `Sync()` runs before every inventory read that `curate`, `reconcile` and `drift` make. `curate` pins its reviewed plan with a digest over revision dates, so a stale cache yields a digest that matches the cache and not the vault.

## The rule

When a cache shares its freshness marker with another client, an unforced refresh is a no-op whenever the other client refreshed last. Force the refresh at every point where a decision is read from the cache. A refresh that reports success proves the call ran, not that the data moved.

Verify freshness by consequence: compare a digest of `(id, revisionDate)` from each reader, never the refresh's status code.

Refs: SEC-006 (#1784); `BWServeClient.Sync` in `cli/internal/secrets/bwserve.go`.
