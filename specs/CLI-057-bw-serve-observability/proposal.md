---
id: "CLI-057-bw-serve-observability"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-27"
issue: "mlorentedev/dotfiles#1315"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-057-bw-serve-observability

> The `bw serve` daemon `dotf secrets unlock` starts leaves no trace when it dies: no log, no pid, nothing for doctor to read.

## Why

<!-- from issue #1315: CLI-057: the bw serve daemon dotf starts has no observability - nil stdio, no pid file, no log - so its deaths leave no trace -->

`BWServeDaemon.Start()` (`cli/internal/secrets/bwserve.go`) runs `cmd.Start()` with `Stdout`/`Stderr` unset — the null device — and records nothing about the process it spawned. On the Windows work box (2026-08-27) the daemon died twice within minutes of `dotf secrets unlock`; every wrapper then failed with "no bw serve daemon is running", and the only way to learn *why* was to hand-start an instrumented `bw serve` and wait for it to die again. A daemon whose whole purpose is to outlive the terminal that started it (WIN-012, `DETACHED_PROCESS`) has, by construction, nobody watching its stderr — so the process itself has to leave the evidence behind.

## What

- `Start()` appends the daemon's stdout+stderr to a **bounded** log file and writes a **pid file** beside it, both under the deploy dir's state area (`<DOTFILES_DIR>/state/bw-serve.log`, `bw-serve.pid`). The paths are derived from the existing deploy-dir resolution (`env.DotfilesDir` in the CLI, `cfg.DotfilesDir` in doctor) through one function in the secrets package — never a literal at a call site. The log rotates once (`.log` → `.log.1`) when it exceeds a fixed cap, so two files bound its growth.
- `dotf secrets unlock` and `dotf secrets lock` print the daemon's pid and the log path on success.
- `dotf doctor`'s bw serve section distinguishes "never started" from "started and gone": when nothing answers on the port but a pid file exists, it reports either *daemon exited* with the last log lines (pid not alive) or *pid alive but not answering* (pid alive). Liveness is probed cross-platform through the doctor `System` seam (`kill -0` on Unix; `OpenProcess` + `GetExitCodeProcess` on Windows, stdlib `syscall` only — `golang.org/x/sys` stays indirect).
- Windows stays the primary target: `DETACHED_PROCESS` is kept, and a Go test proves on this box that a detached child with redirected stdio still starts and its output lands in the log.

## Out of scope

- Restarting a dead daemon automatically (a supervisor). This PR makes the death visible; deciding what to do about it is a different change.
- Diagnosing *why* the daemon died on the work box. That needs the log this PR creates.
- Syncing the vault cache after unlock — CLI-056 (#1316), stacked on this PR.
- Changing the bind address, port, or the detach flags (WIN-012 / #1293 settled those).

## Risks / open questions

- **Detached child + redirected handles on Windows.** `DETACHED_PROCESS` creates the child with no console; file handles are inherited independently of the console, so redirection must keep working — but that is a belief until measured, which is why AC4 is a Windows-run Go test rather than a comment. Resolved by the test.
- **Log content.** `bw serve` writes startup lines and Node stack traces to stdio, not vault material; doctor still shows only the last few lines, truncated, so the report cannot become a channel for a value. Resolved: cap lines and width in the reader.
- **Rotation with the daemon holding the file open.** Renaming an open file works on Unix and on Windows for handles Go opens (`FILE_SHARE_DELETE`); if the rename fails, truncation is the fallback so the cap still holds. Resolved by design.
- **Pid reuse.** A recorded pid may be alive as a different process. Doctor says "alive but not answering" rather than claiming the daemon is up. Accepted: the message names the ambiguity.
- **Closing #1315 trips the archive gate**, and `dotf spec archive` needs an adversarial review that runs `dotf secrets run` against the live daemon this cluster is forbidden to touch. Resolved: the PR uses `Refs #1315`; review + archive are deferred to the owner (Definition of Done skip, stated in the PR body).

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] **AC1** `Start()` with a state dir spawns the daemon with stdout+stderr appended to `<state>/bw-serve.log` and writes `<state>/bw-serve.pid` holding the child's pid; with no state dir it behaves as before (no files).
- [ ] **AC2** The log is bounded: when it exceeds the cap at start time, it is rotated to `bw-serve.log.1` (replacing any previous `.1`) and a fresh log is opened; if the rename fails the log is truncated instead.
- [ ] **AC3** `dotf secrets unlock` and `dotf secrets lock` print the pid and the log path on success; when no pid file exists (daemon started by something else) they say so instead of guessing.
- [ ] **AC4** On Windows, a Go test starts a real long-running fake binary through `Start()` with the production detach attributes and asserts the log and pid files appear, then kills it.
- [ ] **AC5** `dotf doctor`: absent daemon + no pid file → the existing Info line; absent + pid file + pid dead → WARN "daemon exited" with the last log lines; absent + pid file + pid alive → WARN "alive but not answering"; running states unchanged. Asserted by status tag, and the pid-file branch is mutation-checked.
- [ ] **AC6** Liveness probe: `ProcessAlive(pid)` is true for a running child and false after it is killed and reaped, on both Unix and Windows.

## References

- Bitácora board: `mlorentedev/dotfiles#1315`
- Related: #1293 (WIN-012 acceptance, the death this makes visible), #1316 (CLI-056, stacked)
- Related ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md` (the facade the daemon serves)
- Related lessons: 237 (`CREATE_NEW_PROCESS_GROUP` is not detachment), 235 ("I cannot reproduce it" is a statement about the instrument)
