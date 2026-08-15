---
id: "CLI-024-secrets-bw-serve"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-15"
issue: "mlorentedev/dotfiles#622"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-024-secrets-bw-serve

> **Naming**: file lives at `<repo>/specs/CLI-024-secrets-bw-serve/proposal.md`. `CLI-024-secrets-bw-serve` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #622: dotf secrets: manage bw serve behind the bw seam (unlock inside the CLI, no ambient BW_SESSION) -->

Every `dotf secrets` call against a `bw`-backed entry today shells out to the Bitwarden
CLI and requires an ambient `BW_SESSION` the operator exported by hand from a prior
`bw unlock`. This breaks down hard under multi-agent/multi-worktree use (the actual
trigger: an agent session blocked mid-review because it had no way to unlock the vault
itself, and handing an AI agent the master password is a hard no per ADR-028): each
new terminal/session needs its own manual unlock-and-export, non-scalable and fragile.
It also has a real, now-measured performance cost — each `bw` shell-out pays ~1.5-1.7s
of process-spawn overhead, so `dotf secrets verify` over the current 28 bw-backed
entries costs 14-50s wall-clock. The OPS-021 spike (#675, decision on #585) measured a
running `bw serve` daemon answering the same call in ~9ms, and confirmed `bw serve`
exposes its own `POST /unlock` (password in the request body) — a code path independent
of the `bw unlock --raw` CLI subcommand's documented non-interactive bugs. That closes
both problems with one change: the master password crosses one `dotf`-owned prompt,
once, and never becomes an ambient shell variable or an agent-held secret again.

## What

`dotf secrets` gains a `dotf`-managed `bw serve` lifecycle:

- `dotf secrets unlock`: prompts for the master password at a real terminal (echo off,
  like `sudo`), starts the local `bw serve` daemon if it is not already running, and
  POSTs the password once to its `/unlock` endpoint. The password is held only in that
  one process's memory for the duration of the prompt-and-POST; it is never written to
  disk, never exported as `BW_SESSION`, never logged.
- A serve-backed `BWReader` implementation behind the existing seam
  (`cli/internal/secrets/bw.go`), selected automatically when the daemon is reachable
  and unlocked. No read-path consumer (`run`, `show`, `verify`) changes — that is the
  point of the seam. `BWWriter`/`BWCreator`/`BWFolderResolver` (the `set`/`migrate`/
  `render` write path) stay on the CLI shellout for this PR — see Out of scope.
- Falls back to today's CLI-shellout `BWGet`/`BWPut` when the daemon is not running —
  additive, not a breaking change to any existing workflow.
- `dotf doctor` reports daemon state (not running / running-locked / running-unlocked)
  as a new check.

## Out of scope

- `rbw` as an alternative backend — closed by the OPS-021 spike (#675); `bw serve` won
  on both performance and unlock-ergonomics grounds, with zero new dependency.
- Serve-backed `BWWriter`/`BWCreator`/`BWFolderResolver` (the `set`/`migrate`/`render`
  write path) — this PR is read-path only (`BWReader`, covering `run`/`show`/`verify`,
  where the measured pain and the AC7-blocking friction actually live). The write path
  is invoked far less often and stays on the proven CLI-shellout `BWPut`; folding it
  into the same daemon is a natural fast-follow, not a reason to widen this PR.
- #971 (migrate deletes the `age:` DR pointer) and #972 (three `Entries()` consumers
  read `File` without checking `Backend`, including the `BITACORA_PAT` expiry-check
  regression) — real, already-ticketed bugs in the package this spec touches, but a
  different fix surface (registry write path / doctor checks, not the bw read/unlock
  path). Picked up as separate follow-up work, not bundled here.
- Auto-starting the daemon at login (a `systemd --user` unit) — the MVP is an explicit
  `dotf secrets unlock`; auto-start is a UX polish follow-up once the manual path is
  proven.
- Windows-specific daemon lifecycle validation — the code path is written cross-platform
  from the start (no OS-specific branch beyond what `os/exec` already requires), but
  live empirical validation on Windows is deferred to a dedicated Windows session per
  this project's standing batching practice, not held up in this spec.

## Risks / open questions

- **Session idle/lock policy — decided.** No automatic re-lock for the MVP: once
  `dotf secrets unlock` succeeds, the daemon stays unlocked until an explicit
  `dotf secrets lock` or the process/machine stops (reboot, logout). Simplest surface
  (no timeout/timer logic) and matches the actual pain this spec fixes — staying
  unlocked across conversations/sessions on the same machine. An idle-timeout policy is
  a follow-up if the trust model ever needs it, not part of this PR.
- **Single shared daemon across sessions/worktrees is the explicit intent, not a side
  effect (THEORETICAL, no incident yet).** One unlock is meant to serve every `dotf`
  invocation on the machine, across every worktree and every parallel agent session —
  that is what removes the ambient-`BW_SESSION`-per-session tedium this spec exists to
  fix. It also means any local process that can reach `localhost:8087` can read the
  whole unlocked vault for as long as it stays unlocked — already true of `bw serve`
  today, not a new exposure this spec introduces, but worth stating plainly.
- **`bw` version guard:** the `/unlock` endpoint's shape was confirmed empirically
  against the currently-installed `bw` 2026.5.0 (OPS-021 spike). Needs a version check
  or a graceful failure mode if the daemon predates that endpoint.
- **Port/bind safety:** must bind `127.0.0.1` only (never `--hostname all`), matching
  the existing warning in `cli/internal/secrets/bw.go`. Port collision handling
  (another process already on 8087) is an open detail, not yet a blocker.

## Acceptance criteria

- [ ] AC1: `dotf secrets unlock` prompts for the master password with echo disabled,
      starts `bw serve` if not running, and successfully unlocks via `POST /unlock` —
      verified end-to-end without the password ever appearing in shell history,
      process args (`ps`), or written to disk.
- [ ] AC2: With the daemon running and unlocked, `dotf secrets verify` over the current
      registry completes in under 2 seconds (down from the measured 14-50s CLI-shellout
      baseline) — a benchmark test, not just a manual timing.
- [ ] AC3: With no daemon running, every existing `dotf secrets` command (`run`, `show`,
      `verify`, `migrate`, `set`, `render`) behaves exactly as it does today (CLI
      shellout fallback) — full existing test suite passes unmodified.
- [ ] AC4: `dotf doctor` reports the daemon's state (absent / locked / unlocked) as a
      distinct check, not conflated with the existing "bw backend unavailable" error.
- [ ] AC5: The daemon binds `127.0.0.1` only; a test asserts the bind address, not just
      documents the intent.
- [ ] AC6: `dotf secrets lock` stops/locks the daemon explicitly; `dotf secrets unlock`
      run twice while already unlocked is idempotent (no error, no duplicate daemon).

## References

- Bitácora board: #622 (this spec's gate), #585 (original bw backend, spike decision
  comment), #675 (OPS-021 spike, closed with the decision), #612 (secrets lifecycle
  epic).
- Related ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md`.
- Seam + deferral note: `cli/internal/secrets/bw.go` (`BWGet`, `BWPut`).
