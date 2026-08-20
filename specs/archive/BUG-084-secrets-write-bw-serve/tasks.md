---
tags: [spec, tasks, templates]
created: "2026-08-15"
---

# Tasks - BUG-084-secrets-write-bw-serve

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `fix/bug-084-secrets-write-bw-serve`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] Open questions resolved before implementing: the daemon's write surface was
      enumerated from the shipped `@bitwarden/cli` 2026.5.0 bundle rather than assumed
      (the issue asserted `PUT /object/item/<id>` and also assumed `backup` could move
      with it — the first is right, the second is not), and the pin-vs-per-call backend
      question was put to the operator and answered "pin once per command".

## Implementation

- [x] [AC5] Enumerate the real `bw serve` router table; establish that export has no
      endpoint and revise scope accordingly (recorded on #993, comment 5304815767)
- [x] [AC1] `BWServeWriter.SetField` — read-modify-write via `GET` + `PUT /object/item/:id`,
      reusing `setItemField` so field placement cannot diverge from `BWPut`
- [x] [AC2] `BWServeWriter.CreateItem` — `POST /object/item`, reusing `newItemBody`
- [x] [AC2] `BWServeWriter.ResolveFolder` — `GET /list/object/folders` + `POST /object/folder`
- [x] [AC5] `BWBackend` + `SelectBWBackend` — one probe, matched read/write pair
- [x] [AC3] `lockHint` decorator on the shellout pair, keyed to the observed daemon state
- [x] [AC4] `exportLockHint` on `BWExport.Export` — names the BW_SESSION form, and
      explicitly says `dotf secrets unlock` will NOT fix export
- [x] Wire `bwRead()` / `bwWrite()` accessors in `internal/cmd`, pinning per process
- [x] Extend the shared `fakeBWServe` with the write endpoints (one fake, both paths)

## Verification

- [x] `go build ./... && go vet ./... && go test ./...` — all green
- [x] `golangci-lint run` at the pinned 2.12.2 — 0 issues
- [x] AC5 guard mutation-checked: reintroducing the split-brain makes it fail red
- [x] AC4 observed live, before and after, against a real unlocked daemon
- [x] AC1/AC2 live write against a canary item — authorised by the operator and run.
      It FAILED on its first pass, which is why it was worth insisting on: the write
      landed on the server while the daemon kept serving its cached value, so the
      read-back returned the previous fingerprint. Fixed by syncing after every write
      (`syncAfterWrite`); the canary now passes with no explicit sync of its own.
