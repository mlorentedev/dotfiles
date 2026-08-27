# Lesson 237 — CREATE_NEW_PROCESS_GROUP is not detachment: a console child dies with its terminal

**Date:** 2026-08-27
**Context:** WIN-012 (#1293) — `pi` and `opencode` "do not work" on the Windows work box; they work on the Linux one.
**Category:** windows, processes, secrets, daemons

## What happened

Both agent wrappers run `dotf secrets run --only NAN_API_KEY,... -- <agent>`,
which resolves the keys through the `bw serve` daemon before exec'ing the
binary. `dotf secrets unlock` starts that daemon. On Windows it was spawned with
`CREATE_NEW_PROCESS_GROUP` only, and the file's own comment recorded the
lifecycle as *"explicitly deferred to empirical validation on Windows"*.

The validation happened by accident: unlock in one terminal, close it, and every
wrapper in every other terminal failed at once with *"no bw serve daemon is
running"*. Nothing announced the change.

## The actual defect

Two different properties were being conflated:

- `CREATE_NEW_PROCESS_GROUP` re-routes `CTRL_C_EVENT` so a Ctrl+C to the CLI
  does not also hit the child. It does **not** detach the child from the
  console.
- A console-subsystem child created without `DETACHED_PROCESS` (or
  `CREATE_NO_WINDOW`) stays attached to its parent's console. When that console
  closes, Windows delivers `CTRL_CLOSE_EVENT` to every attached process and
  terminates it after the handler timeout.

The CLI exiting was never the problem — stdio was already NUL and Windows has no
parent→child kill. The terminal closing was. Linux never showed it because
`bwserve_unix.go` uses `Setsid`, which does detach.

## The lesson

On Windows, "survives the parent" and "survives the console" are separate
guarantees with separate flags. Test the one you actually need, by effect: a
child spawned with the attribute reports `GetConsoleWindow() == NULL`; a control
child without it reports a handle; where the test process itself has no console
(Git Bash pty, some runners) both read NULL and the test says so instead of
passing vacuously.

A deferral written in a comment is a decision nobody made. It needs a ticket or
a test that fails until the validation happens.

## Guard

`cli/internal/secrets/bwserve_windows_test.go` (`//go:build windows`), plus a
doctor WARN naming `dotf secrets unlock` when the wrappers' keys are bw-backed
and no unlocked daemon is reachable — the precondition every green file/PATH
predicate in that section had been standing in for.
