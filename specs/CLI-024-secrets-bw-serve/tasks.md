---
tags: [spec, tasks, templates]
created: "2026-08-15"
---

# Tasks - CLI-024-secrets-bw-serve

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/CLI-024-secrets-bw-serve`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (lock policy decided: no auto-relock for MVP)

## Implementation

- [x] [P] [AC5] Write failing test: `BWServeDaemon.Start()` invokes its start command with `--hostname 127.0.0.1` (never `all`/`0.0.0.0`) — asserted on the constructed args, not a live process.
- [x] [AC5] Implement `BWServeDaemon` (`cli/internal/secrets/bwserve.go`): `Start()` (injectable `startCmd func() *exec.Cmd` seam, `Setsid` detach so the daemon survives the CLI process exiting, polls `/status` with a bounded timeout until reachable), `Running() bool`, `Status() (daemonState, error)` — mirrors the `BWReader`/`Decryptor` seam pattern already in this package (fakeable, no real `bw` binary needed in tests).
- [x] Refactor: extract the HTTP client + base URL construction shared by daemon control and the reader below (`BWServeClient`, one `call()` core for all four endpoints).
- [x] [P] [AC1] Write failing test: `BWServeDaemon.Unlock(password)` POSTs `{"password": ...}` to `/unlock`, returns a typed error on a non-2xx/`success:false` response, and never logs or returns the password itself in the error.
- [x] [AC1] Implement `Unlock`/`Lock` against a `httptest.Server` fake replaying the real response shapes observed in the OPS-021 spike (`{"success":false,"message":"..."}` / `{"success":true,...}`).
- [x] [P] [AC1] Write failing test: `dotf secrets unlock` (`cli/internal/cmd/secrets_unlock.go`) reads the password via the existing hidden-input seam (`readPassword`, already used by `secrets set` — zero new dependency), starts the daemon if not running, calls `Unlock`, and the password never appears in the command's own logs/output on success or failure.
- [x] [AC1] [AC6] Implement `secrets unlock` + `secrets lock` (`Lock()` → `POST /lock`, idempotent: unlocking an already-unlocked daemon is a no-op success, not an error).
- [x] [P] [AC2] Write failing test: a serve-backed `BWServeReader.Field(item, field)` against an `httptest.Server` fake returns the same value shape `BWGet.Field` does (same `fieldFromItem` extraction: login/notes/custom field), so it is a drop-in behind the `BWReader` seam.
- [x] [AC2] Implement `BWServeReader` (`cli/internal/secrets/bwserve.go`), reusing `fieldFromItem` rather than duplicating field-extraction logic.
- [x] [AC2] [AC3] Write failing test: `BWFallbackReader` selects the serve daemon when reachable+unlocked, and falls back to today's `BWGet` shellout when locked or absent — both paths covered. (Landed as a new `BWFallbackReader` type wired at `cmd/secrets.go`'s `bwReader` default, not inside `resolve.go`'s `resolvers()` map itself — `resolvers()` already delegates to whatever `BWReader` the `Loader.BW` field holds, so the dispatch table needed no change; neither consumer (`run`/`show`/`verify`) changes either way.)
- [x] [AC2] [AC3] Implement the fallback selection (`BWFallbackReader`).
- [x] Refactor for clarity; re-run the full Go suite (`go test ./... -count=1`) + `golangci-lint run`.
- [x] [AC4] Write failing test: a new doctor check (`cli/internal/doctor/checks_bw_serve.go`) reports absent/locked/unlocked daemon state as a distinct section (not merged into `checkBitwardenReach`'s existing tiers).
- [x] [AC4] Implement the check. Also fixed a real nil-seam panic this surfaced in `TestRun_RegistersTheBitwardenReachSection` (the shared `newSys()` test fixture needed a default for the new `BWServeStatus` seam, same pattern as `BWBackedSecrets`/`AgeRoundTrip`).
- [ ] [AC2] Live (this operator's unlocked session, password typed by the operator, never by the agent): benchmark `dotf secrets verify` against a running unlocked daemon; record the wall-clock against the 14-50s CLI-shellout baseline measured in the OPS-021 spike.
- [ ] [AC1] [AC6] Live (same session): `dotf secrets unlock` end-to-end, confirm the password never appears in `ps`, shell history, or any written file; `dotf secrets lock`; re-run `unlock` twice to confirm idempotency.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] Type checks pass
- [ ] Lint passes
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-024-secrets-bw-serve/features.json`):

```json
[
  {
    "id": "CLI-024-secrets-bw-serve-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
