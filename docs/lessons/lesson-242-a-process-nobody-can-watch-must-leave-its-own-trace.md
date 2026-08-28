# Lesson 242 — A process nobody can watch must leave its own trace, and the redirect has to survive the parent

**Date:** 2026-08-27
**Context:** CLI-057 (#1315) — the `bw serve` daemon `dotf secrets unlock` starts died twice within minutes on the Windows work box and left nothing behind: no log, no pid, no listener.
**Category:** secrets, bw serve, os/exec, observability, windows

## What happened

`BWServeDaemon.Start()` ran `cmd.Start()` with `Stdout`/`Stderr` unset and
recorded no pid. That was not an oversight so much as a consequence of the
design: the daemon is detached from every console on purpose (`Setsid` on
Unix, `DETACHED_PROCESS` on Windows — lesson 237), because one unlock has to
serve every later terminal. A process detached from every console has, by
construction, no observer. When it died, the only evidence was that port 8087
stopped answering, and diagnosing it meant hand-starting an instrumented
`bw serve` and waiting for it to die again.

The fix is a bounded log the daemon's stdio appends to, a pid file beside it,
and a doctor branch that reads both when nothing answers: *daemon exited —
last lines: …* versus *never started* versus *pid alive but not answering*.

## The lesson

1. **Detachment and observability pull in opposite directions; pay for both.**
   Detaching a child is the moment its stderr loses its last reader. Redirect
   before you detach, or accept that every death is silent.

2. **The redirect must be a file descriptor the child owns, never a pipe the
   parent drains.** `os/exec` treats an `*os.File` specially: the descriptor
   is handed to the child directly and no goroutine runs in the parent. Any
   other `io.Writer` — a `bytes.Buffer`, a pipe — makes `os/exec` spawn a
   copying goroutine that lives only as long as the parent, and the parent is
   a CLI that exits in milliseconds. The naive fix would have compiled, passed
   a test that called `Wait()`, and lost every line the moment `dotf` returned.

3. **"Absent" was two facts wearing one message.** Before this change doctor
   said *no daemon running* for both a fresh box and a daemon that had just
   crashed. A pid file is the cheapest thing that separates them, and the
   liveness probe behind it needs the real Win32 call (`OpenProcess` +
   `GetExitCodeProcess`): `os.FindProcess` succeeds on Windows for any pid
   that opens, exited or not.

4. **Measure the belief, don't comment it.** "A `DETACHED_PROCESS` child with
   redirected handles still starts" was a belief until a Go test on the box
   started a real detached child through `Start()` and read its lines back
   from the log. Same rule as lesson 240.

## Related

- Lesson 237 — `CREATE_NEW_PROCESS_GROUP` is not detachment
- Lesson 240 — a belief about an environment encoded as an early return
- `cli/internal/secrets/bwserve_state.go`, `procalive_{unix,windows}.go`
