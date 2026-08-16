---
tags: [spec, verification, templates]
created: "2026-08-15"
---

# Verification - BUG-084-secrets-write-bw-serve

## Evidence

| AC | Claim | Proof |
|---|---|---|
| AC1 | `set` succeeds against an unlocked daemon with no `BW_SESSION`, existing item | `TestBWServeWriter_SetField_UpdatesAndPreserves`, `TestBWServeWriter_SetField_MatchesBWPutShape` — **live canary write still pending** (see Gaps) |
| AC2 | same for a new item, incl. folder resolution | `TestBWServeWriter_CreateItem_SendsRawJSON`, `TestBWServeWriter_ResolveFolder` — **live canary write still pending** |
| AC3 | no daemon + no session names `dotf secrets unlock` | `TestSelectBWBackend_ShelloutNamesTheRealRemediation`, `TestSelectBWBackend_ShelloutHalvesBothDecorate` |
| AC4 | `backup` names the `BW_SESSION` form and says why | `TestExportLockHint_NamesTheOnlyInvocationThatWorks` + live capture below |
| AC5 | read, write **and sync** agree on backend | `TestSelectBWBackend_ReadAndWriteAlwaysAgree`, `TestSelectBWBackend_ProbesOnce`, mutation-checked |
| AC6 | observed red before, green after | live capture below |

### AC4 observed live (the headline evidence)

Preconditions: `BW_SESSION` unset; `bw serve` daemon **running and unlocked**. That
combination is the whole point — the daemon being unlocked is exactly why the old
message misled.

```
######## BEFORE - installed 0.41.0 ########
Error: backup: bw export: Vault is locked.

######## AFTER - this branch ########
Error: backup: bw export: bitwarden vault is locked: the escrow cannot go through the
bw serve daemon - `bw serve` exposes no export endpoint, so this is the one
`dotf secrets` operation that needs the bw CLI's own session. `dotf secrets unlock`
will NOT fix it. Run:
    BW_SESSION="$(bw unlock --raw)" dotf secrets backup
which confines the session to that one process rather than exporting it: Vault is locked.
```

The "before" line is the message under which the DR escrow silently never existed
(#997): true, useless, and pointing nowhere.

### Read path re-verified live (regression caught in review)

The first implementation changed `bwReader`'s production default to nil but left one
bare-var use in `secretLoader()` (`BW: bwReader`), which the call-site rewrite did not
match. Every unit test still passed, because they all inject `bwReader`; the live smoke
had only exercised `backup`, which does not go through the Loader. The one production
path not run was the one that broke — a read path that worked *before* this change.

Fixed to `BW: bwRead()`, and the gap that hid it closed by actually running it:

```
$ env -u BW_SESSION dotf secrets verify     # no ambient session; daemon unlocked
...
33 ok, 0 missing, 0 failed
exit=0
```

That is also the AC1 read-half evidence: 33 secrets resolved through the pinned daemon
backend with no `BW_SESSION` anywhere.

### AC5 mutation-checked

A guard never observed failing is not evidence (#898). The split-brain was reintroduced
(daemon reader + shellout writer) and the guard caught it:

```
--- FAIL: TestSelectBWBackend_ReadAndWriteAlwaysAgree/unlocked_daemon_is_used_for_both_halves
    bwbackend_test.go:68: SPLIT BRAIN: reader daemon=true but writer daemon=false
        - this is exactly BUG-084
```

Restored, suite green again.

## Test status

- `go build ./... && go vet ./... && go test ./...` — every package ok
- `golangci-lint run` (pinned 2.12.2, matching `versions.conf`) — **0 issues**
- No regressions: the existing `internal/cmd` suite passes unchanged, including the
  tests that inject `bwReader` / `bwWriter` directly — both remain test seams.

## Gaps (stated, not hidden)

- **AC1/AC2 have no live write yet.** They are proven against an httptest double of the
  daemon, not against Bitwarden. Closing them means writing to a real vault item, which
  is an operator decision, not an agent's — deliberately left for explicit authorization
  rather than performed unilaterally. The repo convention for this is the canary item
  (#612 C8).

## Decisions made during implementation

- **Backend pinned per command, not chosen per call** (operator decision). The read
  path's `BWFallbackReader` re-probes on every call, which self-heals if the daemon
  appears mid-run but lets a daemon that *locks* mid-command split that command across
  two subjects — a narrower instance of this very bug. For `rotate` (read → write →
  read back) a consistent subject beats self-healing: failing whole is better than
  half-applying. `BWFallbackReader` is left in place, unused by the pinned path, since
  it is still the documented read-only seam.
- **`backup` scoped out on evidence, not preference.** `bw serve` has no export route.
  Recorded on #993 with the enumerated router table so the next person does not re-derive
  it.
- **The escrow is NOT assembled from `/list/object/items`.** It would probably produce an
  importable document; *probably* is not a property a DR escrow may have. Filed
  separately with a `bw import` round-trip as its acceptance criterion.
- **Sync was folded into the pinned backend too.** `bwSyncer` defaulted to the daemon
  unconditionally, so a shellout-backed command synced the daemon's cache while reading
  the CLI's — the same split, one path over, inside the very PR fixing it. `rotate`
  writes, syncs, then reads back to prove the write took; syncing the wrong cache makes
  that proof worthless. The AC5 guard now asserts all three halves, not two.
- **Lock hints decorate both halves**, not just the writer: `set` reads before it writes,
  so a locked vault surfaces on the read first.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? **Yes** — "a seam moved halfway is worse
      than a seam not moved": the read path moving alone produced a system where the
      daemon was unlocked, reads worked, and writes reported a locked vault, so every
      diagnostic pointed at the wrong subject. The generalizable rule is that when a
      backend is swapped behind an interface, the *selection* must be one decision, not
      one per caller.
- [ ] ADR-worthy? No — this implements ADR-028 as already ratified; it decides nothing new.
- [ ] New pattern for `00_meta/patterns/`? Not yet. If a third instance of
      "which session is locked?" appears outside this repo, revisit.
