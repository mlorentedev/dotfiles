---
id: "BUG-082-status-poisons-item-reads"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-16"
issue: "mlorentedev/dotfiles#988"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# BUG-082-status-poisons-item-reads

> **Naming**: file lives at `<repo>/specs/BUG-082-status-poisons-item-reads/proposal.md`. `BUG-082-status-poisons-item-reads` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #988: BUG-082: bw serve returns non-JSON on /object/item/<id> under batch
reads, so secrets verify is unusable and non-deterministic -->

`GET /status` breaks `bw serve`. For roughly half a second after one, every
`GET /object/item/<id>` returns HTTP 500 — measured deterministically at 0/10 item reads
failing before a status call and 10/10 immediately after. Backend selection probed
`/status` and then immediately read an item, so **the probe broke the very call it was
authorising**. Before pinning, `BWFallbackReader.Field` called `Status()` before *every*
field read, so resolving 33 secrets meant 33 poisonings and `dotf secrets run` almost
never succeeded. This blocks the SDD archive path outright, because `dotf spec review`
launches its reviewer through `dotf secrets run`.

## What

Backend selection stops asking for a status string and asks the question it actually
needs answered — *will you serve a read?* — via `GET /list/object/folders`.

- `dotf secrets run` / `verify` / `show` succeed reliably against an unlocked daemon.
- Any answer other than a clean success selects the CLI shellout, unchanged.
- No `dotf secrets` code path calls `GET /status` any more.

## Out of scope

- **`checks_bw_serve` in `cli/internal/doctor/`**, which also calls `Status()` — so
  `dotf doctor` poisons the daemon for ~0.5s as a side effect of checking it. Real, and
  another session's lane (cc #1015). Named here so it is not lost.
- **`dotf secrets unlock`'s own use of `Status()`.** It legitimately needs lock state and
  performs no item read afterwards, so the window harms nothing.
- **A retry/backoff on HTTP 500.** Treating the symptom would mask the trigger and make
  the next regression invisible. Removing the trigger is the fix.
- **Fixing it upstream.** Reported to bitwarden/clients#20951 with the repro; we do not
  control that release.

## Risks / open questions

- **`/list/object/folders` on a LOCKED daemon is not directly verified** — locking the
  shared daemon would have disrupted two other live sessions. Mitigated structurally
  rather than by assumption: *any* non-success answer selects the shellout, so the
  behaviour is correct whether a locked daemon returns an error envelope, a 4xx, or a
  transport failure. The fake models the real refusal.
- **The probe endpoint could itself become a poisoner** in a future `bw serve`. That is
  why the guard asserts "selection never calls `/status`" rather than "selection calls
  folders" — the invariant is about what must not happen.
- **The upstream race is not fixed**, only its reliable trigger avoided. Concurrent load
  can presumably still hit it; we no longer cause it deterministically.

## Acceptance criteria

- [ ] AC1 — `dotf secrets run -- true` succeeds **10/10** consecutive invocations against
      an unlocked daemon with no `BW_SESSION` (was 2/10).
- [ ] AC2 — `dotf secrets verify` resolves all 33 registry secrets with zero failures,
      repeatedly.
- [ ] AC3 — a test asserts backend selection never issues `GET /status`, in every daemon
      state, without needing a live daemon.
- [ ] AC4 — the test fake refuses data endpoints when locked, matching the real daemon,
      so AC3's selection logic is exercised against realistic behaviour.

## References

- Bitácora: `mlorentedev/dotfiles#988`
- Upstream: [bitwarden/clients#20951](https://github.com/bitwarden/clients/issues/20951),
  plus [our trigger report](https://github.com/bitwarden/clients/issues/20951#issuecomment-5304990287)
- `specs/BUG-084-secrets-write-bw-serve/` — introduced `SelectBWBackend`, the single
  probe site this changes (#1007, merged)
- #1015 — `checks_bw_mapping` inferring absence from a daemon read; same family
