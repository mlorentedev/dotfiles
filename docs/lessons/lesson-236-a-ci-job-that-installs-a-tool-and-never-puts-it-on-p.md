# Lesson 236 — A CI job that installs a tool and never puts it on PATH certifies nothing it was built to check

**Date:** 2026-08-27
**Context:** TEST-003 (#1298) — `test-windows` green on every PR while a real Windows box failed four `dotf doctor` checks after the same setup.
**Category:** ci, windows, guards, false negatives

## What happened

The `test-windows` job runs `setup-windows.ps1` end to end. Setup installs
`dotf` into `~/.local/bin`, and nothing ever added that directory to
`$GITHUB_PATH`. The real log of a green run (33051778423) reads, in order:

```text
dotf 0.51.0 installed to C:\Users\runneradmin\.local\bin\dotf.exe
... dotf secrets render unavailable/failed
... dotf not on PATH -- skipping pi config deploy
[WARNING] dotf not found on PATH, skipping post-setup diagnostics
Setup Complete!
```

Every `dotf`-gated block degraded exactly as designed — with a warning — and
the job passed. `dotf doctor` had **never run in CI**. The same day, on the
Windows work box, doctor reported four FAILs after a clean setup (WIN-007,
WIN-008, WIN-011 and a false DR FAIL), all of which the job would have shown
had the tool been reachable.

Three independent layers had to be removed before the job could see any of it:

| Layer | What it hid |
|---|---|
| `~/.local/bin` never on `GITHUB_PATH` | every `dotf` call, including the doctor |
| doctor's exit code swallowed by setup (by design, C9) and a bats test asserting the swallow | a red doctor, had it run |
| `DOTFILES_DIR: ${{ github.workspace }}` on the setup step | the mirror gap (WIN-007): doctor would have read `harness/` from the checkout and passed |

## The lesson

"Installed" is a statement about a directory. The property the job needs is
"reachable from the process that has to call it", and that is only established
by effect — run the tool from the same step and check its exit status.

A verifier that degrades gracefully in production must be a **gate** in CI, and
it must run in the environment a real machine has (the default deploy dir, not
the convenient one). Otherwise CI certifies the tree nobody reads.

The fix (#1308, in review as this lesson is written) builds `dotf` from the PR under test (the released binary lags the
tree, as the integration container already knew), puts it on `GITHUB_PATH`
before setup, drops the `DOTFILES_DIR` override, and adds a post-setup
`dotf doctor` step that fails the job. Setup itself stays non-fatal.

## Guard

`tests/ci-windows-doctor-gate.bats` asserts the build step precedes setup, the
override is gone, and the gate step exists. The gate is the guard for
everything downstream.
